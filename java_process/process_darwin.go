package java_process

import "jrasp-daemon/utils"

func IsLoaderJar(pid int32, jarName string) bool {
	return utils.OpenFiles(pid, jarName)
}
