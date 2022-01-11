package watch

import (
	"jrasp-daemon/defs"
	"jrasp-daemon/environ"
	"jrasp-daemon/java_process"
	"jrasp-daemon/userconfig"
	"jrasp-daemon/utils"
	"jrasp-daemon/zlog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/process"
)

// Watch 监控Java进程
type Watch struct {
	// 环境变量与配置
	env     *environ.Environ
	cfg     *userconfig.Config
	selfPid int32 // jrasp-daemon进程自身pid

	scanTicker             *time.Ticker          // 注入定时器
	PidExistsTicker        *time.Ticker          // 进程存活检测定时器
	ProcessInjectTicker    *time.Ticker          // Java进程注入定时器
	LogReportTicker        *time.Ticker          // 进程信息定时上报
	HeartBeatReportTicker  *time.Ticker          // 心跳定时器
	ProcessSyncMap         sync.Map              // 保存监听的java进程
	JavaProcessHandlerChan chan *process.Process // java 进程处理chan
}

func NewWatch(cfg *userconfig.Config, env *environ.Environ) *Watch {
	w := &Watch{
		env:                    env,
		cfg:                    cfg,
		selfPid:                int32(os.Getpid()),
		LogReportTicker:        time.NewTicker(time.Hour * time.Duration(cfg.LogReportTicker)),
		scanTicker:             time.NewTicker(time.Second * time.Duration(cfg.ScanTicker)),
		PidExistsTicker:        time.NewTicker(time.Second * time.Duration(cfg.PidExistsTicker)),
		ProcessInjectTicker:    time.NewTicker(time.Second * time.Duration(cfg.ProcessInjectTicker)),
		HeartBeatReportTicker:  time.NewTicker(time.Minute * time.Duration(cfg.HeartBeatReportTicker)),
		JavaProcessHandlerChan: make(chan *process.Process, 500),
	}
	return w
}

// JavaProcessFilter 相当于`jps`工具的实现
func (w *Watch) JavaProcessFilter() {
	zlog.Infof(defs.WATCH_DEFAULT, "scan java process start...", "scan period:%d(s)", w.cfg.ScanTicker)
	for {
		select {
		case _, ok := <-w.scanTicker.C:
			if !ok {
				return
			}
			pids, err := process.Pids()
			if err != nil {
				continue
			}
			w.checkIsJavaProcess(pids)
		case _, ok := <-w.PidExistsTicker.C:
			if !ok {
				return
			}
			w.removeExitedJavaProcess()
		}
	}
}

func (w *Watch) DoAttach() {
	for {
		select {
		case p, ok := <-w.JavaProcessHandlerChan:
			if !ok {
				zlog.Errorf(defs.WATCH_DEFAULT, "chan shutdown", "java process handler chan closed")
			}
			go w.getJavaProcessInfo(p)
		case _, ok := <-w.ProcessInjectTicker.C:
			if !ok {
				return
			}
			w.ProcessSyncMap.Range(func(pid, p interface{}) bool {
				if w.checkExisted(pid) {
					return true // continue
				}
				javaProcess := (p).(*java_process.JavaProcess)
				if javaProcess.IsInject() {
					if w.cfg.IsDisable() {
						// 禁用模式,java agent 立即退出
						javaProcess.ExitInjectImmediately()
					}
					// 如果已经注入(成功注入/退出注入)并且是开启注入状态,继续保持注入
				} else {
					w.DynamicInject(javaProcess)
				}
				return true // continue
			})
		}
	}
}

func (w *Watch) JavaStatusTimer() {
	for {
		select {
		case _, ok := <-w.LogReportTicker.C:
			if !ok {
				return
			}
			w.logJavaInfo()
		case _, ok := <-w.HeartBeatReportTicker.C:
			if !ok {
				return
			}
			// todo 上报以及注入的进程状态
			zlog.Infof(defs.HEART_BEAT, "[heartBeat]", "pid=%d", w.selfPid)
		}
	}
}

func (w *Watch) logJavaInfo() {
	w.ProcessSyncMap.Range(func(pid, p interface{}) bool {
		exists, err := process.PidExists(pid.(int32))
		if err != nil || !exists {
			// 出错或者不存在时，删除
			w.ProcessSyncMap.Delete(pid)
			// todo 对应的run/pid目录确认删除
			zlog.Infof(defs.WATCH_DEFAULT, "[ScanProcess]", "remove process[%d] watch, process has shutdown", pid)
		} else {
			processJava := (p).(*java_process.JavaProcess)
			zlog.Infof(defs.WATCH_DEFAULT, "[LogReport]", utils.ToString(processJava))
		}
		return true
	})
}

// 进程状态、配置等检测
func (w *Watch) getJavaProcessInfo(procss *process.Process) {
	// 判断是否已经检查过了
	_, f := w.ProcessSyncMap.Load(procss.Pid)
	if f {
		// todo 判断进程启动时间,防止进程退出后再次启动使用相同pid，10秒内重启的进程
		zlog.Debugf(defs.WATCH_DEFAULT, "java process has been monitored", "javaPid:%d", procss.Pid)
		return
	}

	javaProcess := java_process.NewJavaProcess(procss, w.cfg, w.env)

	// cmdline 信息
	javaProcess.SetCmdLines()

	// 设置java进程启动时间
	javaProcess.SetStartTime()

	// 设置注入状态信息
	javaProcess.SetInjectStatus()

	zlog.Infof(defs.WATCH_DEFAULT, "find a java process", utils.ToString(javaProcess))

	// 进程加入观测集合中
	w.ProcessSyncMap.Store(javaProcess.JavaPid, javaProcess)
}

func (w *Watch) removeExitedJavaProcess() {
	w.ProcessSyncMap.Range(func(pid, v interface{}) bool {
		exists, err := process.PidExists(pid.(int32))
		if err != nil || !exists {
			// 出错或者不存在时，删除
			w.ProcessSyncMap.Delete(pid)
			zlog.Infof(defs.WATCH_DEFAULT, "[ScanProcess]", "remove process[%d] watch, process has shutdown", pid)
		}
		return true
	})
}

func (w *Watch) checkExisted(pid interface{}) bool {
	exists, err := process.PidExists(pid.(int32))
	if err != nil || !exists {
		// 出错或者不存在时，删除
		w.ProcessSyncMap.Delete(pid)
		zlog.Infof(defs.WATCH_DEFAULT, "[ScanProcess]", "remove process[%d] watch, process has shutdown", pid)
		return true // continue
	}
	return false
}

func (w *Watch) DynamicInject(javaProcess *java_process.JavaProcess) {
	if w.cfg.IsDynamicMode() {
		err := javaProcess.Attach()
		if err != nil {
			// java_process 执行失败
			zlog.Errorf(defs.WATCH_DEFAULT, "[BUG] attach to java failed", "taget jvm[%d],err:%v", javaProcess.JavaPid, err)
			javaProcess.MarkFailedInjected()
		} else {
			// load agent 之后，标记为[注入状态]，防止 agent 错误再次发生，人工介入排查
			javaProcess.MarkSuccessInjected()
		}
	}
}

func (w *Watch) checkIsJavaProcess(pids []int32) {
	for _, pid := range pids {
		p, err := process.NewProcess(pid)
		if err != nil {
			continue
		}
		exe, err := p.Exe()
		if err != nil {
			continue
		}
		if !IsJavaProcess(exe) {
			continue
		}
		w.JavaProcessHandlerChan <- p
	}
}

func IsJavaProcess(exe string) bool {
	return strings.HasSuffix(exe, "bin/java")
}
