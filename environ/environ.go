package environ

import (
	"errors"
	"jrasp-daemon/utils"
	"net"
	"os"
	"path/filepath"
	"runtime"
)

type Environ struct {
	InstallDir  string `json:"installDir"`  // 安装目录
	HostName    string `json:"hostName"`    // 主机/容器名称
	Ip          string `json:"ip"`          // ipAddress
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
	// ip
	ipAddress, err := getExternalIP()
	if err != nil {
		return nil, err
	}
	env := &Environ{
		HostName:    getHostname(),
		Ip:          ipAddress,
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

func getExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}
