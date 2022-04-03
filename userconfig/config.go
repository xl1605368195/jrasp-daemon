package userconfig

import (
	"fmt"
	"github.com/spf13/viper"
)

// AgentMode 运行模式
type AgentMode string

const (
	STATIC  AgentMode = "static"  // static模式：  被动注入
	DYNAMIC AgentMode = "dynamic" // dynamic模式： 主动注入
	DISABLE AgentMode = "disable" // disbale模式: (主动/被动)注入的退出、禁止注入
)

type Config struct {
	// java agent 运行模式
	AgentMode AgentMode `json:"agentMode"`  // 需要显示配置

	// 激活激活时间如: 15:10
	ActiveTime string `json:"activeTime"`

	// http token 鉴权配置
	Namespace  string `json:"namespace"`
	EnableAuth bool   `json:"enableAuth"`
	Username   string `json:"username"`
	Password   string `json:"password"`

	// 日志配置
	LogLevel int    `json:"logLevel"`
	LogPath  string `json:"logPath"`

	// 性能诊断配置
	EnablePprof bool `json:"enablePprof"`
	PprofPort   int  `json:"pprofPort"`

	// 进程扫描定时器配置
	LogReportTicker       uint32 `json:"logReportTicker"`
	ScanTicker            uint32 `json:"scanTicker"`
	PidExistsTicker       uint32 `json:"pidExistsTicker"`
	ProcessInjectTicker   uint32 `json:"processInjectTicker"`
	HeartBeatReportTicker uint   `json:"heartBeatReportTicker"`
	DependencyTicker      uint32 `json:"dependencyTicker"`

	// nacos 配置
	NamespaceId string   `json:"namespaceId"` // 命名空间
	DataId      string   `json:"dataId"`      // 配置id
	IpAddrs     []string `json:"ipAddrs"`     // nacos 服务端ip列表

	// oss 配置
	BucketURLStr string `json:"bucketURLStr"`
	SecretID     string `json:"secretID"`
	SecretKey    string `json:"secretKey"`

	// jrasp-daemon 自身配置
	ExeOssFileName string `json:"exeOssFileName"` // 相对于bucketURLStr的路径
	ExeOssFileHash string `json:"exeOssFileHash"` // 可执行文件的hash

	// module列表
	ModuleConfigMap map[string]ModuleConfig `json:"moduleConfigMap"` // 模块配置消息
}

// ModuleConfig module信息
type ModuleConfig struct {
	ModuleName  string            `json:"moduleName"`  // 名称，如tomcat.jar
	RouterPath  string            `json:"routerPath"`  // 参数路由路径
	ModuleType  string            `json:"moduleType"`  // 模块类型：hook、algorithm
	DownLoadURL string            `json:"downLoadURL"` // 下载链接
	Md5         string            `json:"md5"`         // 插件hash
	Parameters  map[string]string `json:"parameters"`  // 参数列表
}

func InitConfig() (*Config, error) {
	var (
		v   *viper.Viper
		err error
		c   Config
	)

	v = viper.New()
	v.SetConfigName("config") // 文件名称
	v.SetConfigType("json")   // 文件类型

	// 安装目录下的cfg
	v.AddConfigPath("../cfg")
	v.AddConfigPath("./cfg")

	setDefaultValue(v) // 设置系统默认值
	// 读取配置文件值，并覆盖系统默尔值
	if err = v.ReadInConfig(); err != nil {
		fmt.Print("use default config,can not read config file")
	}

	// 配置对象
	err = v.Unmarshal(&c)
	if err != nil {
		fmt.Printf("unmarshal config failed: %v\n", err)
	}
	return &c, nil
}

// 给参数设置默认值
func setDefaultValue(vp *viper.Viper) {
	vp.SetDefault("AgentMode", STATIC)
	vp.SetDefault("Namespace", "jrasp")
	vp.SetDefault("EnableAttach", false)
	vp.SetDefault("EnableAuth", true)
	vp.SetDefault("LogLevel", 0)
	vp.SetDefault("LogPath", "../logs/jrasp-daemon.log")
	vp.SetDefault("EnablePprof", false)
	vp.SetDefault("PprofPort", 6753)
	vp.SetDefault("Password", "123456")
	vp.SetDefault("Username", "admin")
	vp.SetDefault("EnableDeleyExit", false)
	vp.SetDefault("EnableResourceCheck", false)

	vp.SetDefault("LogReportTicker", 6)
	vp.SetDefault("ScanTicker", 30)
	vp.SetDefault("PidExistsTicker", 10)
	vp.SetDefault("ProcessInjectTicker", 30)
	vp.SetDefault("HeartBeatReportTicker", 5)
	vp.SetDefault("DependencyTicker", 12*60*60)

	vp.SetDefault("EnableBlock", false)
	vp.SetDefault("EnableRceBlock", false)

	vp.SetDefault("AttachTime", -1)

	vp.SetDefault("NamespaceId", "") // default 空间
	// dev 环境：111.229.199.6
	// prod 环境：139.224.220.2:8848,106.14.26.4:8848,47.101.64.183:8848
	vp.SetDefault("IpAddrs", []string{"139.224.220.2","106.14.26.4","47.101.64.183"})
	vp.SetDefault("DataId", "")

	// 腾讯oss 配置
	vp.SetDefault("BucketURLStr", "")
	vp.SetDefault("SecretID", "")
	vp.SetDefault("SecretKey", "")

	// 可执行文件配置,默认为空，不需要更新
	vp.SetDefault("ExecOssFileName", "")
	vp.SetDefault("ExecOssFileHash", "")
}

// IsDynamicMode IsDynamic 是否是动态注入模式
func (config *Config) IsDynamicMode() bool {
	return config.AgentMode == DYNAMIC
}

// IsStaticMode IsNormal 是否是正常模式
func (config *Config) IsStaticMode() bool {
	return config.AgentMode == STATIC
}

// IsDisable 是否是禁用模式
func (config *Config) IsDisable() bool {
	return config.AgentMode == DISABLE
}
