package environ

import (
	"jrasp-daemon/utils"
	"os"
	"path/filepath"
	"runtime"
)

type Environ struct {
	InstallDir  string `json:"installDir"`  // 安装目录
	HostName    string `json:"hostName"`    // 主机/容器名称
	OsType      string `json:"osType"`      // 操作系统类型
	ExeFileHash string `json:"exeFileHash"` // 磁盘可执行文件的md5值
}

func NewEnviron() (*Environ, error) {
	// 可执行文件路径
	execPath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return nil, err
	}
	// md5 值
	md5Str, err := utils.GetFileHash(execPath)
	if err != nil {
		return nil, err
	}
	env := &Environ{
		HostName:    getHostname(),
		InstallDir:  filepath.Dir(filepath.Dir(execPath)),
		OsType:      runtime.GOOS,
		ExeFileHash: md5Str,
	}
	return env, nil
}

func getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}
