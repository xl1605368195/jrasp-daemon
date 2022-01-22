package oss

import (
	"context"
	"io/fs"
	"io/ioutil"
	"jrasp-daemon/defs"
	"jrasp-daemon/environ"
	"jrasp-daemon/userconfig"
	"jrasp-daemon/utils"
	"jrasp-daemon/zlog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// TxOss 腾讯云对象存储
type TxOss struct {
	Client *cos.Client
	cfg    *userconfig.Config
	env    *environ.Environ
}

func NewTxOssClient(cfg *userconfig.Config, env *environ.Environ) *TxOss {
	u, _ := url.Parse(cfg.BucketURLStr)
	b := &cos.BaseURL{BucketURL: u}
	return &TxOss{
		cfg: cfg,
		env: env,
		Client: cos.NewClient(b, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:  cfg.SecretID,
				SecretKey: cfg.SecretKey,
			},
		}),
	}
}

// DownLoad ossName，在对象的访问域名 `examplebucket-1250000000.cos.COS_REGION.myqcloud.com/test/objectPut.go` 中，
// 对象键为 test/objectPut.go，ossName=test/objectPut.go
// filePath，下载的绝对路径,例如 /opt/jrasp-agent/bin/jrasp-daemon_new
func (oss *TxOss) DownLoad(ossName, filePath string) error {
	_, err := oss.Client.Object.GetToFile(context.Background(), ossName, filePath, nil)
	if err != nil {
		return err
	}
	return nil
}

// UpLoad ossName，在对象的访问域名 `examplebucket-1250000000.cos.COS_REGION.myqcloud.com/test/objectPut.go` 中，对象键为 test/objectPut.go，ossName=test/objectPut.go
// filePath，需要上传文件的绝对路径
func (oss *TxOss) UpLoad(ossName, filePath string) (bool, error) {
	_, err := oss.Client.Object.PutFromFile(context.Background(), ossName, filePath, nil)
	if err != nil {
		zlog.Errorf(defs.OSS_UPLOAD, "upload file failed", "ossName:%s,filePath:%s,err:%v", ossName, filePath, err)
		return false, err
	}
	return true, nil
}

func (oss *TxOss) UpdateDaemonFile() {
	// 配置中可执行文件hash不为空，并且与env中可执行文件hash不相同
	if oss.cfg.ExeOssFileHash != "" && oss.cfg.ExeOssFileHash != oss.env.ExeFileHash {
		newFilePath := filepath.Join(oss.env.InstallDir, "bin/jrasp-daemon_new")
		_ = oss.DownLoad(oss.cfg.ExeOssFileName, newFilePath)
		newHash, err := utils.GetFileHash(newFilePath)
		if err != nil {
			zlog.Errorf(defs.OSS_DOWNLOAD, "download  jrasp-daemon exec file", "err:%v", err)
		} else {
			oss.checkHashAndReStart(newHash, newFilePath)
		}
	} else {
		zlog.Infof(defs.OSS_DOWNLOAD, "no need to update jrasp-daemon", "userconfig.ExecOssFileHash:%s,env.ExecDiskFileHash:%s", oss.cfg.ExeOssFileHash, oss.env.ExeFileHash)
	}
}

func (oss *TxOss) checkHashAndReStart(newFileHash string, newFilePath string) {
	// 校验下载文件的hash
	if newFileHash == oss.cfg.ExeOssFileHash {
		oss.replace(newFileHash, newFilePath)
	} else {
		zlog.Errorf(defs.OSS_DOWNLOAD, "[BUG]check file hash err", "newFileHash:%s,configHash:%s", newFileHash, oss.cfg.ExeOssFileHash)
		err := os.Remove(newFilePath)
		if err != nil {
			zlog.Errorf(defs.OSS_DOWNLOAD, "[BUG]delete broken file err", "newFileHash:%s,fileHash:%s", newFilePath, newFileHash)
		}
	}
}

// replace
func (oss *TxOss) replace(newFileHash string, newFilePath string) {
	zlog.Infof(defs.OSS_DOWNLOAD, "check hash success", "hash:%s", newFileHash)
	// 增加可执行权限
	err := os.Chmod(newFilePath, 0700)
	if err != nil {
		zlog.Infof(defs.OSS_DOWNLOAD, "chmod +x jrasp-demon_new", "err:%v", err)
	}
	oldFilePath := filepath.Join(oss.env.InstallDir, "bin/jrasp-daemon")
	err = os.Rename(oldFilePath, newFilePath)
	if err == nil {
		zlog.Infof(defs.OSS_DOWNLOAD, "update jrasp-daemon file success", "jrasp-daemon process will exit...")
		os.Exit(0) // 进程退出
	} else {
		zlog.Errorf(defs.OSS_DOWNLOAD, "[BUG]rename jrasp-daemon file error", "jrasp-daemon file will delete")
		_ = os.Remove(newFilePath)
	}
}

// DownLoadModuleFiles 模块升级
func (oss *TxOss) DownLoadModuleFiles() {
	files, err := ioutil.ReadDir(filepath.Join(oss.env.InstallDir, "required-module"))
	if err != nil {
		zlog.Errorf(defs.OSS_DOWNLOAD, "list disk module file failed", "err:%v", err)
		return
	}
	// 1.先检测磁盘上的全部插件的名称、hash
	fileHashMap := oss.listModuleFile(files)
	// 2.下载
	oss.downLoad(fileHashMap)
}

func (oss *TxOss) downLoad(fileHashMap map[string]string) {
	for _, m := range oss.cfg.ModuleList {
		hash, ok := fileHashMap[m.ModuleName]
		if !ok || hash != m.Md5 {
			// 下载
			tmpFileName := filepath.Join(oss.env.InstallDir, "required-module", m.ModuleName+".tmp")
			err := oss.DownLoad(m.DownLoadURL, tmpFileName) // module.jar.tmp
			if err != nil {
				zlog.Errorf(defs.OSS_DOWNLOAD, "[BUG]download file failed", "tmpFileName:%s,err:%v", tmpFileName, err)
				continue
			}
			// hash 校验
			diskFileHash, err := utils.GetFileHash(tmpFileName)
			if err != nil {
				zlog.Errorf(defs.OSS_DOWNLOAD, "[BUG]cal file hash failed", "filePath:%s,err:%v", tmpFileName, err)
				_ = os.Remove(tmpFileName)
				continue
			}
			// 校验成功，修改名称
			if diskFileHash == m.Md5 {
				zlog.Infof(defs.OSS_DOWNLOAD, "check file hash success", "filePath:%s,hash:%v", tmpFileName, diskFileHash)
				newFilePath := filepath.Join(oss.env.InstallDir, "required-module", m.ModuleName)
				_ = os.Rename(tmpFileName, newFilePath)
			}
		}
	}
}

func (oss *TxOss) listModuleFile(files []fs.FileInfo) map[string]string {
	var fileHashMap = make(map[string]string)
	for _, file := range files {
		name := file.Name()
		if !file.IsDir() && strings.HasSuffix(name, ".jar") {
			hash, err := utils.GetFileHash(filepath.Join(oss.env.InstallDir, "required-module", name))
			if err != nil {
				zlog.Errorf(defs.OSS_DOWNLOAD, "[Fix it] calc file hash error", "file:%s,err:%v", name, err)
			} else {
				fileHashMap[name] = hash
			}
		}
	}
	return fileHashMap
}
