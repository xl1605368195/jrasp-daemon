package java_process

import (
	"encoding/json"
	"fmt"
	"jrasp-daemon/defs"
	"jrasp-daemon/zlog"
)

const DEPENDENCY_URL = "http://%s:%s/jrasp/dependency/get"

type Dependency struct {
	//Pid          int32             `json:"pid"`       // 进程pid信息
	Product string `json:"product"` // jar包的artifactId
	Version string `json:"version"` // jar包的版本version
	Vendor  string `json:"vendor"`  // jar包的groupId
	Path    string `json:"path"`    // jar包的路径
	Source  string `json:"source"`  // jar包的引入方式
}

func NewDependency(product, version, vendor, path, source string) *Dependency {
	return &Dependency{
		Product: product,
		Version: version,
		Vendor:  vendor,
		Path:    path,
		Source:  source,
	}
}

// 获取依赖信息
func (jp *JavaProcess) GetDependency() ([]Dependency, bool) {
	list := make([]Dependency, 0)
	token, _ := jp.getToken()
	// 查询依赖信息
	resp, err := HttpGet(jp.httpClient, fmt.Sprintf(DEPENDENCY_URL, jp.ServerIp, jp.ServerPort), "", token.Data)
	if err != nil {
		zlog.Errorf(defs.HTTP_TOKEN, "[BUG]get dependency error", "send dependency request error:%v", err)
		return list, false
	}
	if resp.Code != 200 {
		zlog.Errorf(defs.HTTP_TOKEN, "[BUG]get dependency error", "error,resp.Code=%d", resp.Code)
		return list, false
	}
	if err := json.Unmarshal([]byte(resp.Data), &list); err == nil {
		return list, true
	}
	return list, false
}
