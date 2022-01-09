package oss

import (
	"context"
	"io/fs"
	"io/ioutil"
	"jrasp-daemon/cfg"
	"jrasp-daemon/common"
	"jrasp-daemon/environ"
	"jrasp-daemon/log"
	"jrasp-daemon/utils"
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
	cfg    *cfg.Config
	env    *environ.Environ
}

func NewTxOssClient(cfg *cfg.Config, env *environ.Environ) *TxOss {
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
func (this *TxOss) DownLoad(ossName, filePath string) error {
	_, err := this.Client.Object.GetToFile(context.Background(), ossName, filePath, nil)
	if err != nil {
		return err
	}
	return nil
}

// UpLoad ossName，在对象的访问域名 `examplebucket-1250000000.cos.COS_REGION.myqcloud.com/test/objectPut.go` 中，对象键为 test/objectPut.go，ossName=test/objectPut.go
// filePath，需要上传文件的绝对路径
func (this *TxOss) UpLoad(ossName, filePath string) (bool, error) {
	_, err := this.Client.Object.PutFromFile(context.Background(), ossName, filePath, nil)
	if err != nil {
		log.Errorf(common.OSS_UPLOAD, "upload file failed", "ossName:%s,filePath:%s,err:%v", ossName, filePath, err)
		return false, err
	}
	return true, nil
}

func (this *TxOss) UpdateDaemonFile() {
	// 配置中可执行文件hash不为空，并且与env中可执行文件hash不相同
	if this.cfg.ExecOssFileHash != "" && this.cfg.ExecOssFileHash != this.env.ExecDiskFileHash {
		newFilePath := filepath.Join(this.env.RaspHome, "bin/jrasp-daemon_new")
		_ = this.DownLoad(this.cfg.ExecOssFileName, newFilePath)
		newHash, err := utils.CalcFileHash(newFilePath)
		if err != nil {
			log.Errorf(common.OSS_DOWNLOAD, "download  jrasp-daemon exec file", "err:%v", err)
		} else {
			this.checkHashAndReStart(newHash, newFilePath)
		}
	} else {
		log.Infof(common.OSS_DOWNLOAD, "no need to update jrasp-daemon", "cfg.ExecOssFileHash:%s,env.ExecDiskFileHash:%s", this.cfg.ExecOssFileHash, this.env.ExecDiskFileHash)
	}
}

func (this *TxOss) checkHashAndReStart(newFileHash string, newFilePath string) {
	// 校验下载文件的hash
	if newFileHash == this.cfg.ExecOssFileHash {
		this.replace(newFileHash, newFilePath)
	} else {
		log.Errorf(common.OSS_DOWNLOAD, "[Fix it]check file hash err", "newFileHash:%s,configHash:%s", newFileHash, this.cfg.ExecOssFileHash)
		err := os.Remove(newFilePath)
		if err != nil {
			log.Errorf(common.OSS_DOWNLOAD, "[Fix it]delete broken file err", "newFileHash:%s,fileHash:%s", newFilePath, newFileHash)
		}
	}
}

// replace
func (this *TxOss) replace(newFileHash string, newFilePath string) {
	log.Infof(common.OSS_DOWNLOAD, "check hash success", "hash:%s", newFileHash)
	// 增加可执行权限
	err := os.Chmod(newFilePath, 0700)
	if err != nil {
		log.Infof(common.OSS_DOWNLOAD, "chmod +x jrasp-demon_new", "err:%v", err)
	}
	oldFilePath := filepath.Join(this.env.RaspHome, "bin/jrasp-daemon")
	err = os.Rename(oldFilePath, newFilePath)
	if err == nil {
		log.Infof(common.OSS_DOWNLOAD, "update jrasp-daemon file success", "jrasp-daemon process will exit...")
		os.Exit(0) // 进程退出
	} else {
		log.Errorf(common.OSS_DOWNLOAD, "[Fix it]rename jrasp-daemon file error", "jrasp-daemon file will delete")
		_ = os.Remove(newFilePath)
	}
}

// DownLoadModuleFiles 模块升级
func (this *TxOss) DownLoadModuleFiles() {
	files, err := ioutil.ReadDir(filepath.Join(this.env.RaspHome, "required-module"))
	if err != nil {
		log.Errorf(common.OSS_DOWNLOAD, "list disk module file failed", "err:%v", err)
		return
	}
	// 1.先检测磁盘上的全部插件的名称、hash
	fileHashMap := this.listModuleFile(files)
	// 2.下载
	this.downLoad(fileHashMap)
}

func (this *TxOss) downLoad(fileHashMap map[string]string) {
	for _, m := range this.cfg.ModuleList {
		hash, ok := fileHashMap[m.ModuleName]
		if !ok || hash != m.Md5 {
			// 下载
			tmpFileName := filepath.Join(this.env.RaspHome, "required-module", m.ModuleName+".tmp")
			err := this.DownLoad(m.DownLoadURL, tmpFileName) // module.jar.tmp
			if err != nil {
				log.Errorf(common.OSS_DOWNLOAD, "[Fixt it]download file failed", "tmpFileName:%s,err:%v", tmpFileName, err)
				continue
			}
			// hash 校验
			diskFileHash, err := utils.CalcFileHash(tmpFileName)
			if err != nil {
				log.Errorf(common.OSS_DOWNLOAD, "[Fixt it]cal file hash failed", "filePath:%s,err:%v", tmpFileName, err)
				_ = os.Remove(tmpFileName)
				continue
			}
			// 校验成功，修改名称
			if diskFileHash == m.Md5 {
				log.Infof(common.OSS_DOWNLOAD, "check file hash success", "filePath:%s,hash:%v", tmpFileName, diskFileHash)
				newFilePath := filepath.Join(this.env.RaspHome, "required-module", m.ModuleName)
				_ = os.Rename(tmpFileName, newFilePath)
			}
		}
	}
}

func (this *TxOss) listModuleFile(files []fs.FileInfo) map[string]string {
	var fileHashMap = make(map[string]string)
	for _, file := range files {
		name := file.Name()
		if !file.IsDir() && strings.HasSuffix(name, ".jar") {
			hash, err := utils.CalcFileHash(filepath.Join(this.env.RaspHome, "required-module", name))
			if err != nil {
				log.Errorf(common.OSS_DOWNLOAD, "[Fix it] calc file hash error", "file:%s,err:%v", name, err)
			} else {
				fileHashMap[name] = hash
			}
		}
	}
	return fileHashMap
}
