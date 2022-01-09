package main

import (
	"fmt"
	"jrasp-daemon/cfg"
	"jrasp-daemon/common"
	"jrasp-daemon/environ"
	"jrasp-daemon/log"
	"jrasp-daemon/nacos"
	"jrasp-daemon/oss"
	"jrasp-daemon/utils"
	"jrasp-daemon/watch"
	"net/http"
	"os/signal"
	"syscall"
)

func init() {
	signal.Notify(common.Sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
}

func main() {

	fmt.Print(common.LOGO)

	// 环境变量初始化
	env, err := environ.NewEnviron()
	if err != nil {
		fmt.Printf("env init error %s\n", err.Error())
		return
	}

	// 配置初始化
	conf, err := cfg.InitConfig()
	if err != nil {
		fmt.Printf("config init error %s\n", err.Error())
		return
	}

	// zap日志初始化
	log.InitLog(conf.LogLevel, conf.LogPath, env.HostName)

	// jrasp-daemon 启动标志
	log.Infof(common.START_UP, "jrasp-daemon startup", "enableHook:%t", conf.EnableHook)

	// 日志配置值
	log.Infof(common.LOG_VALUE, "log value", "logLevel:%d,logPath:%s", conf.LogLevel, conf.LogPath)

	// 环境信息打印
	log.Infof(common.ENV_VALUE, "env value", utils.ToString(env))

	// 配置信息打印
	log.Infof(common.CONFIG_VALUE, "config value", utils.ToString(conf))

	// 配置客户端初始化
	nacos.NacosInit(conf, env)

	// OSS 客户端初始化和可执行文件下载
	ossClient := oss.NewTxOssClient(conf, env)

	// 下载最新的可执行文件
	ossClient.UpdateDaemonFile()

	ossClient.DownLoadModuleFiles()

	newWatch := watch.NewWatch(conf, env)

	// 进程扫描
	go newWatch.ScanProcess()

	// 进程注入
	go newWatch.DoAttach()

	// 进程状态定时上报
	go newWatch.LogReport()

	// start pprof for debug
	go debug(conf)
	<-common.Sig
}

func debug(conf *cfg.Config) {
	if conf.EnablePprof {
		err := http.ListenAndServe(fmt.Sprintf(":%d", conf.PprofPort), nil)
		if err != nil {
			log.Errorf(common.DEBUG_PPROF, "pprof ListenAndServe failed", "err:%s", err.Error())
		}
	}
}
