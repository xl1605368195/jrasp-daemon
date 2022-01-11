package userconfig

import (
	"fmt"
	"github.com/spf13/viper"
)

// AgentMode 运行模式
type AgentMode string

const (
	MONITOR AgentMode = "monitor" // monitor模式
	NORMAL  AgentMode = "normal"  // normal模式
	DYNAMIC AgentMode = "dynamic" // dynamic模式
	DISABLE AgentMode = "disable" // disbale模式
)

type Config struct {
	// java agent 运行模式
	AgentMode AgentMode `json:"agentMode"`

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

	// 阻断相关参数的
	EnableBlock bool `json:"enableBlock"` // 阻断总开关，关闭之后，各个模块都关闭阻断；开启之后，还需要开启模块对应的阻断参数
	// 命令执行相关参数
	EnableRceBlock bool     `json:"enableRceBlock"` // rce阻断配置
	RceWhiteList   []string `json:"rceWhiteList"`   // rce命令执行白名单

	// nacos 配置
	NamespaceId string `json:"namespaceId"` // 命名空间
	DataId      string `json:"dataId"`      // 配置id
	IpAddrs []string `json:"ipAddrs"`       // nacos 服务端ip列表

	// oss 配置
	BucketURLStr string `json:"bucketURLStr"`
	SecretID     string `json:"secretID"`
	SecretKey    string `json:"secretKey"`

	// jrasp-daemon 自身配置
	ExeOssFileName string `json:"exeOssFileName"` // 相对于bucketURLStr的路径
	ExeOssFileHash string `json:"exeOssFileHash"` // 可执行文件的hash

	// module列表
	ModuleList []Module `json:"moduleList"` // 全部jar包
}

// module信息
type Module struct {
	ModuleName        string `json:"name"`              // 名称，如tomcat.jar
	DownLoadURL       string `json:"downLoadURL"`       // 下载链接
	Md5               string `json:"md5"`               // 插件hash
	MiddlewareVersion string `json:"middlewareVersion"` // 目标中间件版本
	ClassName         string `json:"className"`         // 目标中间件版本关键类,用来查询jar包版本
}

func InitConfig() (*Config, error) {
	var (
		v   *viper.Viper
		err error
		c   Config
	)

	v = viper.New()
	v.SetConfigName("config") // 文件名称
	v.SetConfigType("yml")    // 文件类型

	// 安装目录下的cfg
	v.AddConfigPath("../cfg")
	v.AddConfigPath("./cfg")

	setDefaultValue(v) // 设置系统默认值
	// 读取配置文件值，并覆盖系统默尔值
	if err = v.ReadInConfig(); err != nil {
		return nil, err
	}

	// 配置对象
	err = v.Unmarshal(&c)
	if err != nil {
		fmt.Printf("unmarshal json failed: %v\n", err)
	}
	return &c, nil
}

// 给参数设置默认值
func setDefaultValue(vp *viper.Viper) {
	vp.SetDefault("AgentMode", NORMAL)
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

	vp.SetDefault("EnableBlock", false)
	vp.SetDefault("EnableRceBlock", false)

	vp.SetDefault("AttachTime", -1)

	vp.SetDefault("NamespaceId", "aab3be32-0588-4c4c-88da-0f5e39ee9447")
	vp.SetDefault("IpAddrs", []string{"111.229.199.6"}) // 目前仅有一个
	vp.SetDefault("DataId", "")

	// 腾讯oss 配置
	vp.SetDefault("BucketURLStr", "https://jrasp-1254321150.cos.ap-shanghai.myqcloud.com")
	vp.SetDefault("SecretID", "AKID9C3jDCylGajjX9snEYgbLtGRWaaPZTil")
	vp.SetDefault("SecretKey", "BNoJrFSTJxiXGm7TkCTYR7av77uZ7Uec")

	// 可执行文件配置,默认为空，不需要更新
	vp.SetDefault("ExecOssFileName", "")
	vp.SetDefault("ExecOssFileHash", "")
}

// IsDynamic 是否是动态注入模式
func (config *Config) IsDynamicMode() bool {
	return config.AgentMode == DYNAMIC
}

// IsMonitor 是否是监控模式
func (config *Config) IsMonitorMode() bool {
	return config.AgentMode == MONITOR
}

// IsNormal 是否是正常模式
func (config *Config) IsNormalMode() bool {
	return config.AgentMode == NORMAL
}

// IsDisable 是否是禁用模式
func (config *Config) IsDisable() bool {
	return config.AgentMode == DISABLE
}
