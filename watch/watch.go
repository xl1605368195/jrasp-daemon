package watch

import (
	"jrasp-daemon/cfg"
	"jrasp-daemon/common"
	"jrasp-daemon/environ"
	"jrasp-daemon/java_process"
	"jrasp-daemon/log"
	"jrasp-daemon/utils"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/process"
)

// Watch 监控Java进程
type Watch struct {
	// 环境变量与配置
	env         *environ.Environ
	cfg         *cfg.Config
	selfPid     int32  // jrasp-daemon进程自身pid
	injectCount uint32 // 统计已经注入的java进程数量

	scanTicker            *time.Ticker          // 注入定时器
	PidExistsTicker       *time.Ticker          // 进程存活检测定时器
	ProcessInjectTicker   *time.Ticker          // Java进程注入定时器
	LogReportTicker       *time.Ticker          // 进程信息定时上报
	HeartBeatReportTicker *time.Ticker          // 心跳定时器
	ProcessMap            sync.Map              // 保存监听的java进程
	JavaProcessCreateChan chan *process.Process // 进程扫描chan
}

func NewWatch(cfg *cfg.Config, env *environ.Environ) *Watch {
	w := &Watch{
		env:                   env,
		cfg:                   cfg,
		selfPid:               int32(os.Getpid()),
		LogReportTicker:       time.NewTicker(time.Hour * time.Duration(cfg.LogReportTicker)),
		scanTicker:            time.NewTicker(time.Second * time.Duration(cfg.ScanTicker)),
		PidExistsTicker:       time.NewTicker(time.Second * time.Duration(cfg.PidExistsTicker)),
		ProcessInjectTicker:   time.NewTicker(time.Second * time.Duration(cfg.ProcessInjectTicker)),
		HeartBeatReportTicker: time.NewTicker(time.Minute * time.Duration(cfg.HeartBeatReportTicker)),
		JavaProcessCreateChan: make(chan *process.Process, 500),
	}
	return w
}

// 全部上报
func (this *Watch) ScanProcess() {
	log.Infof(common.WATCH_DEFAULT, "scan java process start...", "scan period:%d(s)", this.cfg.ScanTicker)
	for {
		select {
		case _, ok := <-this.scanTicker.C:
			if !ok {
				return
			}
			pids, err := process.Pids()
			if err != nil {
				continue
			}
			// 检查是否是java进程
			this.checkIsJavaProcess(pids)
		case _, ok := <-this.PidExistsTicker.C:
			if !ok {
				return
			}
			// 移除已经退出的Java进程
			this.removeExitedJavaProcess()
		}
	}
}

func (this *Watch) DoAttach() {
	for {
		select {
		case p, ok := <-this.JavaProcessCreateChan:
			if !ok {
				log.Errorf(common.WATCH_DEFAULT, "JavaProcessCreateChan shutdown", "java process create chan error")
			}
			go this.getJavaProcessInfo(p)
		case _, ok := <-this.ProcessInjectTicker.C:
			if !ok {
				return
			}
			this.ProcessMap.Range(func(pid, p interface{}) bool {
				if this.checkExisted(pid) {
					return true // continue
				}
				javaProcess := (p).(*java_process.JavaProcess)
				if javaProcess.IsInject() {
					if !this.cfg.EnableHook {
						this.exitInject(javaProcess)
					}
					// 如果已经注入(成功注入/退出注入)并且是开启注入状态,继续保持注入
				} else {
					this.doInject(javaProcess)
				}
				return true // continue
			})
		}
	}
}

// 日志定时上报
func (this *Watch) LogReport() {
	for {
		select {
		case _, ok := <-this.LogReportTicker.C:
			if !ok {
				return
			}
			this.logJavaInfo()
		case _, ok := <-this.HeartBeatReportTicker.C:
			if !ok {
				return
			}
			// todo 上报以及注入的进程状态
			log.Infof(common.HEART_BEAT, "[heartBeat]", "pid=%d", this.selfPid)
		}
	}
}

func (this *Watch) logJavaInfo() {
	this.ProcessMap.Range(func(pid, p interface{}) bool {
		exists, err := process.PidExists(pid.(int32))
		if err != nil || !exists {
			// 出错或者不存在时，删除
			this.ProcessMap.Delete(pid)
			// todo 对应的run/pid目录确认删除
			log.Infof(common.WATCH_DEFAULT, "[ScanProcess]", "remove process[%d] watch, process has shutdown", pid)
		} else {
			processJava := (p).(*java_process.JavaProcess)
			log.Infof(common.WATCH_DEFAULT, "[LogReport]", utils.ToString(processJava))
		}
		return true
	})
}

// 进程资源、配置等检测
func (this *Watch) getJavaProcessInfo(procss *process.Process) {
	// 判断是否已经检查过了
	_, f := this.ProcessMap.Load(procss.Pid)
	if f {
		// todo 判断进程启动时间,防止进程退出后再次启动使用相同pid，10秒内重启的进程
		log.Debugf(common.WATCH_DEFAULT, "java process has been monitored", "javaPid:%d", procss.Pid)
		return
	}

	javaProcess := java_process.NewJavaProcess(procss, this.cfg, this.env)

	// cmdline 信息
	cmdLines, err := procss.CmdlineSlice()
	if err != nil {
		log.Warnf(common.WATCH_DEFAULT, "get process cmdLines error", `{"pid":%d,"err":%v}`, procss.Pid, err)
	}
	javaProcess.SetCmdLines(cmdLines)

	// todo jdk版本信息

	// 进程启动时间
	startTime, err := java_process.GetProcessStartTime(procss.Pid)
	// 进程启动时间
	if err != nil {
		log.Warnf(common.WATCH_DEFAULT, "get process startup time error", `{"pid":%d,"err":%v}`, procss.Pid, err)
	}
	timsStr := startTime.Format(common.DATE_FORMAT)
	javaProcess.StartTime = timsStr

	// 判断run/pid文件是否存在
	if javaProcess.CheckRunDir() {
		success := javaProcess.ReadTokenFile()
		if success {
			javaProcess.MarkSuccessInjected() // 已经注入过
		} else {
			javaProcess.MarkFailedExitInject() // 退出失败，文件异常
		}
	} else {
		javaProcess.MarkNotInjected() // 未注入过
	}
	log.Infof(common.WATCH_DEFAULT, "find a java process", utils.ToString(javaProcess))
	this.ProcessMap.Store(javaProcess.JavaPid, javaProcess)
}

func (this *Watch) removeExitedJavaProcess() {
	this.ProcessMap.Range(func(pid, v interface{}) bool {
		exists, err := process.PidExists(pid.(int32))
		if err != nil || !exists {
			// 出错或者不存在时，删除
			this.ProcessMap.Delete(pid)
			log.Infof(common.WATCH_DEFAULT, "[ScanProcess]", "remove process[%d] watch, process has shutdown", pid)
		}
		return true
	})
}

func (this *Watch) checkExisted(pid interface{}) bool {
	exists, err := process.PidExists(pid.(int32))
	if err != nil || !exists {
		// 出错或者不存在时，删除
		this.ProcessMap.Delete(pid)
		log.Infof(common.WATCH_DEFAULT, "[ScanProcess]", "remove process[%d] watch, process has shutdown", pid)
		return true // continue
	}
	return false
}

func (this *Watch) doInject(javaProcess *java_process.JavaProcess) {
	// 没有注入并且是开启注入状态和允许attach
	if this.cfg.EnableHook && this.cfg.EnableAttach {
		// 检查注入时间是否满足设定条件
		if javaProcess.CheckAttachTime() {
			err := javaProcess.Attach()
			if err != nil {
				// java_process 执行失败
				log.Errorf(common.WATCH_DEFAULT, "[Fix it]attach to java failed", "taget jvm[%d],err:%v", javaProcess.JavaPid, err)
				javaProcess.MarkFailedInjected()
			} else {
				// load agent 之后，标记为[注入状态]，防止 agent 错误再次发生，人工介入排查
				javaProcess.MarkSuccessInjected()
			}
		}
	}
}

func (this *Watch) exitInject(javaProcess *java_process.JavaProcess) {
	// 关闭注入
	success := javaProcess.ShutDownAgent()
	if success {
		// 标记为成功退出状态
		javaProcess.MarkExitInject()
		log.Infof(common.WATCH_DEFAULT, "java agent exit", "java pid:%d,status:%t", javaProcess.JavaPid, success)
	} else {
		// 标记为异常退出状态
		javaProcess.MarkFailedExitInject()
		log.Errorf(common.WATCH_DEFAULT, "[Fix it]java agent exit failed", "java pid:%d,status:%t", javaProcess.JavaPid, success)
	}
}

func (this *Watch) checkIsJavaProcess(pids []int32) {
	for _, pid := range pids {
		if pid != this.selfPid && pid != 1 {
			p, err := process.NewProcess(pid)
			if err != nil {
				continue
			}
			// 获取可执行文件
			exe, err := p.Exe()
			if err != nil {
				continue
			}
			// 检查是否是Java进程
			if !IsJavaExe(exe) {
				continue
			}
			// Java 进程加入
			this.JavaProcessCreateChan <- p
		}
	}
}

func IsJavaExe(exe string) bool {
	i := strings.LastIndex(exe, "bin/java")
	if i < 0 {
		return false
	}
	return true
}
