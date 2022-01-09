package java_process

import (
	"jrasp-daemon/utils"
	"time"
)

func GetProcessStartTime(pid int32) (time.Time, error) {
	// not implement
	return time.Time{}, nil
}

func IsLoaderJar(pid int32, jarName string) bool {
	return utils.OpenFiles(pid, jarName)
}
