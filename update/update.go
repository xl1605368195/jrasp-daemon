package update

import (
	"io/ioutil"
	"jrasp-daemon/defs"
	"jrasp-daemon/environ"
	"jrasp-daemon/userconfig"
	"jrasp-daemon/utils"
	"jrasp-daemon/zlog"
	"os"
	"path/filepath"
	"strings"
)

// TxOss 腾讯云对象存储
type Update struct {
	cfg *userconfig.Config
	env *environ.Environ
}

func NewUpdateClient(cfg *userconfig.Config, env *environ.Environ) *Update {
	return &Update{
		cfg: cfg,
		env: env,
	}
}

// DownLoad
// url 文件下载链接
// filePath 下载的绝对路径
func (this *Update) DownLoad(url, filePath string) error {
	err := utils.DownLoadFile(url, filePath)
	if err != nil {
		return err
	}
	return nil
}

// UpdateDaemonFile 更新守护进程
func (this *Update) UpdateDaemonFile() {
	// 配置中可执行文件hash不为空，并且与env中可执行文件hash不相同
	if this.cfg.ExeOssFileHash != "" && this.cfg.ExeOssFileHash != this.env.ExeFileHash {
		newFilePath := filepath.Join(this.env.InstallDir, "bin", "jrasp-daemon.tmp")
		_ = this.DownLoad(this.cfg.ExeOssFileName, newFilePath)
		newHash, err := utils.GetFileHash(newFilePath)
		if err != nil {
			zlog.Errorf(defs.DOWNLOAD, "download  jrasp-daemon file", "err:%v", err)
		} else {
			// 校验下载文件的hash
			if newHash == this.cfg.ExeOssFileHash {
				this.replace()
			} else {
				zlog.Errorf(defs.DOWNLOAD, "[BUG]check new file hash err", "newFileHash:%s,configHash:%s", newHash, this.cfg.ExeOssFileHash)
				err := os.Remove(newFilePath)
				if err != nil {
					zlog.Errorf(defs.DOWNLOAD, "[BUG]delete broken file err", "newFileHash:%s", newHash)
				}
			}
		}
	} else {
		zlog.Infof(defs.DOWNLOAD, "no need to update jrasp-daemon", "userconfig.ExecOssFileHash:%s,env.ExecDiskFileHash:%s", this.cfg.ExeOssFileHash, this.env.ExeFileHash)
	}
}

// replace 文件rename
func (this *Update) replace() {
	// 增加可执行权限
	err := os.Chmod("jrasp-daemon.tmp", 0700)
	if err != nil {
		zlog.Infof(defs.DOWNLOAD, "chmod +x jrasp-demon.tmp", "err:%v", err)
	}
	err = os.Rename("jrasp-daemon.tmp", "jrasp-daemon")
	if err == nil {
		zlog.Infof(defs.DOWNLOAD, "update jrasp-daemon file success", "rename jrasp-daemon file success,daemon process will exit...")
		// 再次check
		success, _ := utils.PathExists("jrasp-daemon")
		if success {
			os.Exit(0) // 进程退出
		}
	} else {
		zlog.Errorf(defs.DOWNLOAD, "[BUG]rename jrasp-daemon file error", "jrasp-daemon file will delete")
		_ = os.Remove("jrasp-daemon.tmp")
	}
}

// DownLoadModuleFiles 模块升级
func (this *Update) DownLoadModuleFiles() {
	// 获取磁盘上的插件
	files, err := ioutil.ReadDir(filepath.Join(this.env.InstallDir, "required-module"))
	if err != nil {
		zlog.Errorf(defs.DOWNLOAD, "list disk module file failed", "err:%v", err)
		return
	}

	// 1.先检测磁盘上的全部插件的名称、hash
	var fileHashMap = make(map[string]string)
	for _, file := range files {
		name := file.Name()
		if !file.IsDir() && strings.HasSuffix(name, ".jar") {
			hash, err := utils.GetFileHash(filepath.Join(this.env.InstallDir, "required-module", name))
			if err != nil {
				zlog.Errorf(defs.DOWNLOAD, "[Fix it] calc file hash error", "file:%s,err:%v", name, err)
			} else {
				fileHashMap[name] = hash
			}
		}
	}

	// 2.下载
	for _, m := range this.cfg.ModuleConfigMap {
		hash, ok := fileHashMap[m.ModuleName]
		if !ok || hash != m.Md5 {
			// 下载
			tmpFileName := filepath.Join(this.env.InstallDir, "required-module", m.ModuleName+".tmp")
			err := utils.DownLoadFile(m.DownLoadURL, tmpFileName) // module.jar.tmp
			if err != nil {
				zlog.Errorf(defs.DOWNLOAD, "[BUG]download file failed", "tmpFileName:%s,err:%v", tmpFileName, err)
				continue
			}
			// hash 校验
			diskFileHash, err := utils.GetFileHash(tmpFileName)
			if err != nil {
				zlog.Errorf(defs.DOWNLOAD, "[BUG]cal file hash failed", "filePath:%s,err:%v", tmpFileName, err)
				_ = os.Remove(tmpFileName)
				continue
			}
			// 校验成功，修改名称
			if diskFileHash == m.Md5 {
				zlog.Infof(defs.DOWNLOAD, "check file hash success", "filePath:%s,hash:%v", tmpFileName, diskFileHash)
				newFilePath := filepath.Join(this.env.InstallDir, "required-module", m.ModuleName+".jar")
				err := os.Rename(tmpFileName, newFilePath)
				if err != nil {
					zlog.Errorf(defs.DOWNLOAD, "[BUG]rename file name failed", "tmpFileName:%s,newFilePath:%s,err:%v", tmpFileName, newFilePath, err)
					_ = os.Remove(tmpFileName)
					continue
				}
			}
		}
	}
}
