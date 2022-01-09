package java_process

import (
	"errors"
	"fmt"
	"io/ioutil"
	"jrasp-daemon/cfg"
	"jrasp-daemon/common"
	"jrasp-daemon/environ"
	"jrasp-daemon/log"
	"jrasp-daemon/utils"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/process"
)

const (
	serverIp   = "0.0.0.0"
	serverPort = 0
)

type JavaProcess struct {
	JavaPid      int32    `json:"javaPid"`      // 进程信息
	StartTime    string   `json:"startTime"`    // 启动时间
	CmdLines     []string `json:"cmdLines"`     // 命令行信息
	EnableHook   bool     `json:"enableHook"`   // 是否开启注入(字节码转换)
	EnableAttach bool     `json:"enableAttach"` // 是否允许attach
	ServerIp     string   `json:"serverIp"`     // 内置jetty开启的IP:端口
	ServerPort   string   `json:"serverPort"`
	lock         sync.RWMutex
	env          *environ.Environ // 环境变量
	cfg          *cfg.Config      // 配置
	httpClient   *http.Client

	InjectedStatus InjectType `json:"injectedStatus"`

	// 是否需要更新参数
	needUpdateParameters bool
	hitCmdlineBlackList  bool

	// jdk信息
	// JdkVersion string `json:"jdkVersion"` // jdk 版本

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

func NewJavaProcess(p *process.Process, cfg *cfg.Config, env *environ.Environ) *JavaProcess {
	javaProcess := &JavaProcess{
		JavaPid:      p.Pid,
		env:          env,
		cfg:          cfg,
		EnableHook:   cfg.EnableHook,
		EnableAttach: cfg.EnableAttach,
		httpClient:   &http.Client{},
	}
	return javaProcess
}

// 执行attach
func (this *JavaProcess) Attach() error {
	// 执行attach并检查java_pid文件
	err := this.execCmd()
	if err != nil {
		return err
	}

	// read token file
	ok := this.ReadTokenFile()
	if !ok {
		log.Errorf(common.ATTACH_DEFAULT, "[Attach]", "read token file error:%v", err)
		return errors.New("read token file,error")
	}
	log.Infof(common.ATTACH_DEFAULT, "[Attach]", "attach to jvm[%d] success", this.JavaPid)

	// login
	token, err := this.getToken()
	if err != nil {
		log.Errorf(common.ATTACH_DEFAULT, "[Attach]", "log failed,error:%v", err)
		return errors.New("login failed")
	}
	if token.Code != 200 {
		log.Errorf(common.ATTACH_DEFAULT, "[Attach]", "login response bad,message=%s", token.Message)
	}
	return nil
}

func (this *JavaProcess) execCmd() error {
	log.Infof(common.ATTACH_DEFAULT, "[Attach]", "attach to jvm[%d] start...", this.JavaPid)
	// 通过attach 传递给目标jvm的参数
	agentArgs := fmt.Sprintf("raspHome=%s;serverIp=%s;serverPort=%d;namespace=%s;enableAuth=%t;username=%s;password=%s",
		this.env.RaspHome, serverIp, serverPort, this.cfg.Namespace, this.cfg.EnableAuth, this.cfg.Username, this.cfg.Password)

	// jattach pid load instrument false jrasp-launcher.jar
	cmd := exec.Command(
		filepath.Join(this.env.RaspHome, "bin", "jattach"),
		fmt.Sprintf("%d", this.JavaPid),
		"load", "instrument", "false",
		fmt.Sprintf("%s=%s", filepath.Join(this.env.RaspHome, "lib", "jrasp-launcher.jar"), agentArgs),
	)

	log.Debugf(common.ATTACH_DEFAULT, "[Attach]", "cmdArgs:%s", cmd.Args)
	// 权限切换在 jattach 里面做了，直接命令执行就行

	if err := cmd.Start(); err != nil {
		log.Warnf(common.ATTACH_DEFAULT, "[Attach]", "cmd.Start error:%v", err)
		return err
	}

	if err := cmd.Wait(); err != nil {
		log.Warnf(common.ATTACH_DEFAULT, "[Attach]", "cmd.Wait error:%v", err)
		return err
	}

	//判断socket文件是否存在
	sockfile := filepath.Join(os.TempDir(), fmt.Sprintf(".java_pid%d", this.GetPid()))
	if exist(sockfile) {
		log.Infof(common.ATTACH_DEFAULT, "[Attach]", "target jvm[%d] create sockfile success:%s", this.JavaPid, sockfile)
	} else {
		log.Warnf(common.ATTACH_DEFAULT, "[Attach]", "target jvm[%d] create socket file failed", this.JavaPid)
		return errors.New(fmt.Sprintf("target jvm[%d] create socket file failed", this.JavaPid))
	}

	err := cmd.Process.Release()
	if err != nil {
		log.Warnf(common.ATTACH_DEFAULT, "[Attach]", "cmd.Process.Release error:%v", err)
		return err
	}
	return nil
}

func (this *JavaProcess) IsHitCmdlineBlackList() (isHit bool) {
	if len(this.cfg.CmdLineBlackList) == 0 {
		return false
	}
	for _, cmdline := range this.CmdLines {
		for _, substr := range this.cfg.CmdLineBlackList {
			if strings.Contains(cmdline, substr) {
				this.hitCmdlineBlackList = true
				return true
			}
		}
	}
	return false
}

// 检测注入条件是否满足
func (this *JavaProcess) checkInjectCondition() bool {
	// 是否开启全局注入开关
	// 是否命中黑名单
	return this.cfg.EnableHook && !this.hitCmdlineBlackList
}

// CheckRunDir run/pid目录
func (this *JavaProcess) CheckRunDir() bool {
	runPidFilePath := filepath.Join(this.env.RaspHome, "run", fmt.Sprintf("%d", this.JavaPid))
	exist, err := utils.PathExists(runPidFilePath)
	if err != nil || !exist {
		return false
	}
	return true
}
func (this *JavaProcess) ReadTokenFile() bool {
	// 文件不存在
	tokenFilePath := filepath.Join(this.env.RaspHome, "run", fmt.Sprintf("%d", this.JavaPid), ".jrasp.token")
	exist, err := utils.PathExists(tokenFilePath)
	if err != nil {
		log.Errorf(common.ATTACH_READ_TOKEN, "[token file]", "check token file[%s],error:%v", tokenFilePath, err)
		return false
	}

	// 文件存在
	if exist {
		fileContent, err := ioutil.ReadFile(tokenFilePath)
		if err != nil {
			log.Errorf(common.ATTACH_READ_TOKEN, "[token file]", "read attach token file[%s],error:%v", tokenFilePath, err)
			return false
		}
		fileContentStr := string(fileContent)                         // jrasp;admin;123456;0.0.0.0;61535
		fileContentStr = strings.Replace(fileContentStr, " ", "", -1) // 字符串去掉"\n"和"空格"
		fileContentStr = strings.Replace(fileContentStr, "\n", "", -1)
		tokenArray := strings.Split(fileContentStr, ";")
		log.Debugf(common.ATTACH_READ_TOKEN, "[token file]", "token file content:%s", fileContentStr)
		if len(tokenArray) == 5 {
			this.ServerIp = tokenArray[3]
			this.ServerPort = tokenArray[4]
			return true
		} else {
			log.Errorf(common.ATTACH_READ_TOKEN, "[Attach]", "[Fix it] token file content bad,tokenFilePath:%s,fileContentStr:%s", tokenFilePath, fileContentStr)
			return false
		}
	} else {
		log.Infof(common.ATTACH_READ_TOKEN, "[token file]", "attach token file[%s] not exist", tokenFilePath)
		return false
	}
}

// 注入状态
type InjectType string

const (
	NOT_INJECT InjectType = "not inject" // 未注入

	SUCCESS_INJECT InjectType = "success inject" // 注入正常
	FAILED_INJECT  InjectType = "failed inject"  // 注入时失败

	SUCCESS_EXIT InjectType = "success exit inject" // 注入后正常退出状态
	FAILED_EXIT  InjectType = "failed exit"         // 退出时后失败

	FAILED_DEGRADE  InjectType = "failed degrade"  // 降级失败时后失败
	SUCCESS_DEGRADE InjectType = "success degrade" // 降级正常
)

func (this *JavaProcess) IsInject() bool {
	return this.InjectedStatus == SUCCESS_INJECT || this.InjectedStatus == FAILED_INJECT
}

func (this *JavaProcess) MarkExitInject() {
	this.InjectedStatus = SUCCESS_EXIT
}

func (this *JavaProcess) MarkFailedExitInject() {
	this.InjectedStatus = FAILED_EXIT
}

func (this *JavaProcess) MarkSuccessInjected() {
	this.InjectedStatus = SUCCESS_INJECT
}

func (this *JavaProcess) MarkFailedInjected() {
	this.InjectedStatus = FAILED_INJECT
}

func (this *JavaProcess) MarkNotInjected() {
	this.InjectedStatus = NOT_INJECT
}

func (this *JavaProcess) SetPid(pid int32) {
	this.JavaPid = pid
}

func (this *JavaProcess) SetCmdLines(cmds []string) {
	this.CmdLines = cmds
}

func (this *JavaProcess) GetPid() int32 {
	return this.JavaPid
}

func (this *JavaProcess) CheckAttachTime() bool {
	if this.cfg.AttachTime < 0 || time.Now().Hour() == this.cfg.AttachTime {
		return true
	}
	return false
}

func exist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

// 修改密码
func (this *JavaProcess) updatePassword() {

}
