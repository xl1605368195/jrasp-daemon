package environ

import (
	"fmt"
	"jrasp-daemon/utils"
	"os"
	"path/filepath"
	"runtime"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

type Environ struct {
	RaspHome string `json:"raspHome"` // rasp 安装目录

	// 是否为线上环境
	IsProd bool `json:"isProd"`

	// 服务与机器
	HostName    string `json:"hostName"`    // 主机/容器名称
	ServiceName string `json:"serviceName"` // 服务名称
	CompanyName string `json:"companyName"` // 公司名称

	// cpu、core、system、os、osVersion
	// sysType := runtime.GOO
	OsType        string `json:"osType"` // 操作系统类型
	KernelVersion string `json:"kernelVersion"`

	// cpu 信息
	PhysicalCnt int `json:"physicalCnt"`
	LogicalCnt  int `json:"logicalCnt"`

	// 内存
	TotalMemory uint64 `json:"totalMemory"` // 单位GB

	// 代码编译信息
	CodeVersion string `json:"codeVersion"`
	BuildTime   string `json:"buildTime"`

	// 磁盘可执行文件的md5值
	ExecDiskFileHash string `json:"execDiskFileHash"`
}

func NewEnviron() (*Environ, error) {
	// 可执行文件路径
	execPath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return nil, err
	}

	// cpu cnt
	physicalCnt, err := cpu.Counts(false)
	if err != nil {
		fmt.Printf("get physical cpu cnt error:%v", err)
	}
	logicalCnt, err := cpu.Counts(true)
	if err != nil {
		fmt.Printf("get logicalCnt cpu cnt error:%v", err)
	}

	// 内核
	kernelVersion, err := host.KernelVersion()
	if err != nil {
		fmt.Printf("get kernelVersion info  error:%v", err)
	}

	// 总内存
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		fmt.Printf("get mem info error:%v", err)
	}
	totalMem := virtualMemory.Total

	// md5 值
	md5Str, err := utils.CalcFileHash(execPath)
	if err != nil {
		return nil, err
	}

	env := &Environ{
		HostName:         utils.GetHostname(),
		RaspHome:         filepath.Dir(filepath.Dir(execPath)),
		OsType:           runtime.GOOS,
		PhysicalCnt:      physicalCnt,
		LogicalCnt:       logicalCnt,
		KernelVersion:    kernelVersion,
		TotalMemory:      totalMem / 1024 / 1024 / 1024,
		ExecDiskFileHash: md5Str,
	}
	return env, nil
}

func (this *Environ) CheckConfig() error {
	return nil
}
