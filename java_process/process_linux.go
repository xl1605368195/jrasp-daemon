package java_process

import (
	"fmt"
	"io/ioutil"
	"jrasp-daemon/common"
	"jrasp-daemon/log"
	"os"
	"strings"
	"time"
)

func GetProcessStartTime(pid int32) (time.Time, error) {
	procDir := fmt.Sprintf("/proc/%d/status", pid)
	stat, err := os.Lstat(procDir)
	if err != nil {
		return time.Time{}, err
	}
	return stat.ModTime(), nil
}

// 是否加载了jar文件
func IsLoaderJar(pid int32, jarName string) bool {
	path := fmt.Sprintf("/proc/%d/maps", pid)
	_, err := os.Stat(path)
	if err != nil {
		log.Warnf(common.LOAD_JAR, "[isLoaderJar]", fmt.Sprintf("进程%d的maps文件不存在,err:%v", pid, err))
		return false
	}
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warnf(common.LOAD_JAR, "[isLoaderJar]", fmt.Sprintf("打开进程%d的maps文件失败,err:%v", pid, err))
		return false
	}
	if strings.Contains(string(buf), jarName) {
		return true
	}
	return false
}
