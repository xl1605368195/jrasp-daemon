package nacos

import (
	"io/ioutil"
	"jrasp-daemon/defs"
	"jrasp-daemon/environ"
	"jrasp-daemon/userconfig"
	"jrasp-daemon/zlog"
	"os"
	"path/filepath"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func NacosInit(cfg *userconfig.Config, env *environ.Environ) {

	clientConfig := constant.ClientConfig{
		NamespaceId:         cfg.NamespaceId,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "/tmp/nacos/log",
		CacheDir:            "/tmp/nacos/cache",
		RotateTime:          "24h",
		MaxAge:              3,
		LogLevel:            "error",
	}

	var serverConfigs []constant.ServerConfig

	for i := 0; i < len(cfg.IpAddrs); i++ {
		serverConfig := constant.ServerConfig{
			IpAddr:      cfg.IpAddrs[i],
			ContextPath: "/nacos",
			Port:        8848,
			Scheme:      "http",
		}
		serverConfigs = append(serverConfigs, serverConfig)
	}

	// 将服务注册到nacos
	namingClient, _ := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)

	registerStatus, err := namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          env.HostName,
		Port:        8848,
		ServiceName: env.HostName,
		Weight:      10,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    map[string]string{"raspVersion": defs.JRASP_DAEMON_VERSION},
		ClusterName: "DEFAULT",       // 默认值DEFAULT
		GroupName:   "DEFAULT_GROUP", // 默认值DEFAULT_GROUP
	})
	if err != nil {
		zlog.Warnf(defs.NACOS_INIT, "[registerStatus]", "registerStatus:%t,err:%v", registerStatus, err)
	}

	configClient, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)

	// dataId配置值为空时，使用主机名称
	var dataId = ""
	if cfg.DataId == "" {
		dataId = env.HostName
	}

	//获取配置
	err = configClient.ListenConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  "DEFAULT_GROUP",
		OnChange: func(namespace, group, dataId, data string) {
			zlog.Infof(defs.NACOS_LISTEN_CONFIG, "[ListenConfig]", "group:%s,dataId=%s,data=%s", group, dataId, data)
			err = ioutil.WriteFile(filepath.Join(env.InstallDir, "cfg", "config.json"), []byte(data), 0600)
			if err != nil {
				zlog.Warnf(defs.NACOS_LISTEN_CONFIG, "[ListenConfig]", "write file to config.json,err:%v", err)
			}
			zlog.Infof(defs.NACOS_LISTEN_CONFIG, "[ListenConfig]", "config update,jrasp-daemon will exit(0)...")
			os.Exit(0)
		},
	})

	if err != nil {
		zlog.Warnf(defs.NACOS_INIT, "[ListenConfig]", "configClient.ListenConfig,err:%v", err)
	}
	zlog.Infof(defs.NACOS_INIT, "[NacosInit]", "nacos init success")
}
