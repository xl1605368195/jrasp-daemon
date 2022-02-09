package java_process

import (
	"errors"
	"fmt"
	"io/ioutil"
	"jrasp-daemon/defs"
	"jrasp-daemon/environ"
	"jrasp-daemon/userconfig"
	"jrasp-daemon/utils"
	"jrasp-daemon/zlog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/process"
)

const (
	serverIp   = "0.0.0.0"
	serverPort = 0
)

// 注入状态
type InjectType string

const (
	NOT_INJECT InjectType = "not inject" // 未注入

	SUCCESS_INJECT InjectType = "success inject" // 注入正常
	FAILED_INJECT  InjectType = "failed inject"  // 注入时失败

	SUCCESS_EXIT InjectType = "success uninstall agent" // agent卸载成功
	FAILED_EXIT  InjectType = "failed uninstall agent"  // agent卸载失败

	FAILED_DEGRADE  InjectType = "failed degrade"  // 降级失败时后失败
	SUCCESS_DEGRADE InjectType = "success degrade" // 降级正常
)

type JavaProcess struct {
	JavaPid    int32                `json:"javaPid"`   // 进程信息
	StartTime  string               `json:"startTime"` // 启动时间
	CmdLines   []string             `json:"cmdLines"`  // 命令行信息
	AgentMode  userconfig.AgentMode `json:"agentMode"` // agent 运行模式
	ServerIp   string               `json:"serverIp"`  // 内置jetty开启的IP:端口
	ServerPort string               `json:"serverPort"`

	env     *environ.Environ   // 环境变量
	cfg     *userconfig.Config // 配置
	process *process.Process   // process 对象

	httpClient *http.Client

	InjectedStatus InjectType `json:"injectedStatus"`

	// 加载的模块信息
	ModuleInfos []ModuleInfo `json:"moduleInfo"`
}

// module信息
type ModuleInfo struct {
	Name        string `json:"name"`
	IsLoaded    bool   `json:"isLoaded"`
	IsActivated bool   `json:"isActivated"`
	ClassCnt    int    `json:"classCnt"`
	MethodCnt   int    `json:"methodCnt"`
	Version     string `json:"version"`
	Author      string `json:"author"`
}

func NewJavaProcess(p *process.Process, cfg *userconfig.Config, env *environ.Environ) *JavaProcess {
	javaProcess := &JavaProcess{
		JavaPid:    p.Pid,
		process:    p,
		env:        env,
		cfg:        cfg,
		AgentMode:  cfg.AgentMode,
		httpClient: &http.Client{},
	}
	return javaProcess
}

// 执行attach
func (jp *JavaProcess) Attach() error {
	// 执行attach并检查java_pid文件
	err := jp.execCmd()
	if err != nil {
		return err
	}

	// read token file
	ok := jp.ReadTokenFile()
	if !ok {
		zlog.Errorf(defs.ATTACH_DEFAULT, "[Attach]", "read token file error:%v", err)
		return errors.New("read token file,error")
	}
	zlog.Infof(defs.ATTACH_DEFAULT, "[Attach]", "attach to jvm[%d] success", jp.JavaPid)

	// login
	token, err := jp.getToken()
	if err != nil {
		zlog.Errorf(defs.ATTACH_DEFAULT, "[Attach]", "ylog failed,error:%v", err)
		return errors.New("login failed")
	}
	if token.Code != 200 {
		zlog.Errorf(defs.ATTACH_DEFAULT, "[Attach]", "login response bad,message=%s", token.Message)
	}

	// soft flush
	jp.SoftFlush()

	return nil
}

func (jp *JavaProcess) execCmd() error {
	zlog.Infof(defs.ATTACH_DEFAULT, "[Attach]", "attach to jvm[%d] start...", jp.JavaPid)
	// 通过attach 传递给目标jvm的参数
	agentArgs := fmt.Sprintf("raspHome=%s;serverIp=%s;serverPort=%d;namespace=%s;enableAuth=%t;username=%s;password=%s",
		jp.env.InstallDir, serverIp, serverPort, jp.cfg.Namespace, jp.cfg.EnableAuth, jp.cfg.Username, jp.cfg.Password)

	// jattach pid load instrument false jrasp-launcher.jar
	cmd := exec.Command(
		filepath.Join(jp.env.InstallDir, "bin", "jattach"),
		fmt.Sprintf("%d", jp.JavaPid),
		"load", "instrument", "false",
		fmt.Sprintf("%s=%s", filepath.Join(jp.env.InstallDir, "lib", "jrasp-launcher.jar"), agentArgs),
	)

	zlog.Debugf(defs.ATTACH_DEFAULT, "[Attach]", "cmdArgs:%s", cmd.Args)
	// 权限切换在 jattach 里面做了，直接命令执行就行

	if err := cmd.Start(); err != nil {
		zlog.Warnf(defs.ATTACH_DEFAULT, "[Attach]", "cmd.Start error:%v", err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		zlog.Warnf(defs.ATTACH_DEFAULT, "[Attach]", "cmd.Wait error:%v", err)
		return err
	}

	//判断socket文件是否存在
	sockfile := filepath.Join(os.TempDir(), fmt.Sprintf(".java_pid%d", jp.GetPid()))
	if exist(sockfile) {
		zlog.Infof(defs.ATTACH_DEFAULT, "[Attach]", "target jvm[%d] create sockfile success:%s", jp.JavaPid, sockfile)
	} else {
		zlog.Warnf(defs.ATTACH_DEFAULT, "[Attach]", "target jvm[%d] create socket file failed", jp.JavaPid)
		return fmt.Errorf("target jvm[%d] create socket file failed", jp.JavaPid)
	}

	err := cmd.Process.Release()
	if err != nil {
		zlog.Warnf(defs.ATTACH_DEFAULT, "[Attach]", "cmd.Process.Release error:%v", err)
		return err
	}
	return nil
}

// CheckRunDir run/pid目录
func (jp *JavaProcess) CheckRunDir() bool {
	runPidFilePath := filepath.Join(jp.env.InstallDir, "run", fmt.Sprintf("%d", jp.JavaPid))
	exist, err := utils.PathExists(runPidFilePath)
	if err != nil || !exist {
		return false
	}
	return true
}

func (jp *JavaProcess) ReadTokenFile() bool {
	// 文件不存在
	tokenFilePath := filepath.Join(jp.env.InstallDir, "run", fmt.Sprintf("%d", jp.JavaPid), ".jrasp.token")
	exist, err := utils.PathExists(tokenFilePath)
	if err != nil {
		zlog.Errorf(defs.ATTACH_READ_TOKEN, "[token file]", "check token file[%s],error:%v", tokenFilePath, err)
		return false
	}

	// 文件存在
	if exist {
		fileContent, err := ioutil.ReadFile(tokenFilePath)
		if err != nil {
			zlog.Errorf(defs.ATTACH_READ_TOKEN, "[token file]", "read attach token file[%s],error:%v", tokenFilePath, err)
			return false
		}
		fileContentStr := string(fileContent)                         // jrasp;admin;123456;0.0.0.0;61535
		fileContentStr = strings.Replace(fileContentStr, " ", "", -1) // 字符串去掉"\n"和"空格"
		fileContentStr = strings.Replace(fileContentStr, "\n", "", -1)
		tokenArray := strings.Split(fileContentStr, ";")
		zlog.Debugf(defs.ATTACH_READ_TOKEN, "[token file]", "token file content:%s", fileContentStr)
		if len(tokenArray) == 5 {
			jp.ServerIp = tokenArray[3]
			jp.ServerPort = tokenArray[4]
			return true
		} else {
			zlog.Errorf(defs.ATTACH_READ_TOKEN, "[Attach]", "[Fix it] token file content bad,tokenFilePath:%s,fileContentStr:%s", tokenFilePath, fileContentStr)
			return false
		}
	} else {
		zlog.Infof(defs.ATTACH_READ_TOKEN, "[token file]", "attach token file[%s] not exist", tokenFilePath)
		return false
	}
}

func (jp *JavaProcess) IsInject() bool {
	return jp.InjectedStatus == SUCCESS_INJECT || jp.InjectedStatus == FAILED_INJECT
}

func (jp *JavaProcess) MarkExitInject() {
	jp.InjectedStatus = SUCCESS_EXIT
}

func (jp *JavaProcess) MarkFailedExitInject() {
	jp.InjectedStatus = FAILED_EXIT
}

func (jp *JavaProcess) MarkSuccessInjected() {
	jp.InjectedStatus = SUCCESS_INJECT
}

func (jp *JavaProcess) MarkFailedInjected() {
	jp.InjectedStatus = FAILED_INJECT
}

func (jp *JavaProcess) MarkNotInjected() {
	jp.InjectedStatus = NOT_INJECT
}

func (jp *JavaProcess) SetPid(pid int32) {
	jp.JavaPid = pid
}

func (jp *JavaProcess) SetCmdLines() {
	cmdLines, err := jp.process.CmdlineSlice()
	if err != nil {
		zlog.Warnf(defs.WATCH_DEFAULT, "get process cmdLines error", `{"pid":%d,"err":%v}`, jp.JavaPid, err)
	}
	jp.CmdLines = cmdLines
}

func (jp *JavaProcess) GetPid() int32 {
	return jp.JavaPid
}

func (jp *JavaProcess) SetStartTime() {
	startTime, err := jp.process.CreateTime()
	if err != nil {
		zlog.Warnf(defs.WATCH_DEFAULT, "get process startup time error", `{"pid":%d,"err":%v}`, jp.JavaPid, err)
	}
	time := time.Unix(startTime/1000, 0)
	timsStr := time.Format(defs.DATE_FORMAT)
	jp.StartTime = timsStr
}

func (jp *JavaProcess) SetInjectStatus() {
	if jp.CheckRunDir() {
		success := jp.ReadTokenFile()
		if success {
			jp.MarkSuccessInjected() // 已经注入过
		} else {
			jp.MarkFailedExitInject() // 退出失败，文件异常
		}
	} else {
		jp.MarkNotInjected() // 未注入过
	}
}

func exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}
