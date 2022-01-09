package nacos

import (
	"io/ioutil"
	"jrasp-daemon/cfg"
	"jrasp-daemon/common"
	"jrasp-daemon/environ"
	"jrasp-daemon/log"
	"os"
	"path/filepath"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
)

func NacosInit(cfg *cfg.Config, env *environ.Environ) {

	clientConfig := constant.ClientConfig{
		NamespaceId:         cfg.NamespaceId,
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              "../logs",
		CacheDir:            "../logs/cache",
		RotateTime:          "24h",
		MaxAge:              3,
		LogLevel:            "warn",
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
		Metadata:    map[string]string{"raspVersion": common.JRASP_DAEMON_VERSION},
		ClusterName: "DEFAULT",       // 默认值DEFAULT
		GroupName:   "DEFAULT_GROUP", // 默认值DEFAULT_GROUP
	})
	if err != nil {
		log.Warnf(common.NACOS_INIT, "[registerStatus]", "registerStatus:%t,err:%v", registerStatus, err)
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
			log.Infof(common.NACOS_LISTEN_CONFIG, "[ListenConfig]", "group:%s,dataId=%s,data=%s", group, dataId, data)
			err = ioutil.WriteFile(filepath.Join(env.RaspHome, "cfg", "config.json"), []byte(data), 0600)
			if err != nil {
				log.Warnf(common.NACOS_LISTEN_CONFIG, "[ListenConfig]", "write file to config.json,err:%v", err)
			}
			log.Infof(common.NACOS_LISTEN_CONFIG, "[ListenConfig]", "jrasp-daemon will exit...")
			os.Exit(0)
		},
	})

	if err != nil {
		log.Warnf(common.NACOS_INIT, "[ListenConfig]", "configClient.ListenConfig,err:%v", err)
	}
	log.Infof(common.NACOS_INIT, "[NacosInit]", "nacos init success")
}
