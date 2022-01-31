package main

import (
	"fmt"
	"jrasp-daemon/defs"
	"jrasp-daemon/environ"
	"jrasp-daemon/nacos"
	"jrasp-daemon/oss"
	"jrasp-daemon/userconfig"
	"jrasp-daemon/utils"
	"jrasp-daemon/watch"
	"jrasp-daemon/zlog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var Sig = make(chan os.Signal, 1)

func init() {
	signal.Notify(Sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
}

func main() {

	fmt.Print(defs.LOGO)

	// 环境变量初始化
	env, err := environ.NewEnviron()
	if err != nil {
		fmt.Printf("env init error %s\n", err.Error())
		return
	}

	// 配置初始化
	conf, err := userconfig.InitConfig()
	if err != nil {
		fmt.Printf("userconfig init error %s\n", err.Error())
		return
	}

	// zap日志初始化
	zlog.InitLog(conf.LogLevel, conf.LogPath, env.HostName)

	// jrasp-daemon 启动标志
	zlog.Infof(defs.START_UP, "daemon startup", `{"agentMode":"%s"}`, conf.AgentMode)

	// 日志配置值
	zlog.Infof(defs.LOG_VALUE, "log config value", "logLevel:%d,logPath:%s", conf.LogLevel, conf.LogPath)

	// 环境信息打印
	zlog.Infof(defs.ENV_VALUE, "env config value", utils.ToString(env))

	// 配置信息打印
	zlog.Infof(defs.CONFIG_VALUE, "user config value", utils.ToString(conf))

	// 配置客户端初始化
	nacos.NacosInit(conf, env)

	// OSS 客户端初始化和可执行文件下载
	ossClient := oss.NewTxOssClient(conf, env)

	// 下载最新的可执行文件
	ossClient.UpdateDaemonFile()

	// 下载模块插件
	ossClient.DownLoadModuleFiles()

	newWatch := watch.NewWatch(conf, env)

	// jpf工具
	go newWatch.JavaProcessFilter()

	// 进程注入
	go newWatch.DoAttach()

	// 进程状态定时上报
	go newWatch.JavaStatusTimer()

	// start pprof for debug
	go debug(conf)

	// block main
	<-Sig
}

func debug(conf *userconfig.Config) {
	if conf.EnablePprof {
		err := http.ListenAndServe(fmt.Sprintf(":%d", conf.PprofPort), nil)
		if err != nil {
			zlog.Errorf(defs.DEBUG_PPROF, "pprof ListenAndServe failed", "err:%s", err.Error())
		}
	}
}
