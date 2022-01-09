package java_process

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"jrasp-daemon/common"
	"jrasp-daemon/log"
	"net/http"
	"strings"
)

const (
	shutdownUrl = "http://%s:%s/jrasp/control/shutdown"
	loginUrl    = "http://%s:%s/jrasp/user/login"
	degradeUrl  = "http://%s:%s/jrasp/user/login"
	listUrl     = "http://%s:%s/jrasp/module/list"
)

type Response struct {
	Code    int    `json:"code"`
	Data    string `json:"data"`
	Message string `json:"message"`
}

// 关闭注入
func (this *JavaProcess) ShutDownAgent() bool {
	token, err := this.getToken()
	if err != nil {
		log.Errorf(common.HTTP_TOKEN, "shut down agent", "get http token err:%v", err)
		return false
	}
	resp, err := HttpGet(this.httpClient, fmt.Sprintf(shutdownUrl, this.ServerIp, this.ServerPort), "", token.Data)
	if err != nil {
		log.Errorf(common.HTTP_TOKEN, "shut down agent", "send shutdown request error:%v", err)
		return false
	}
	if resp.Code != 200 {
		log.Errorf(common.HTTP_TOKEN, "shut down agent", "send shutdown request error,resp.Code=%d", resp.Code)
		return false
	}
	return true
}

// 降级冻结
func (this *JavaProcess) DegradeAgent() bool {
	token, err := this.getToken()
	// 查询所有模块  listUrl
	resp, err := HttpGet(this.httpClient, fmt.Sprintf(listUrl, this.ServerIp, this.ServerPort), "", token.Data)
	if err != nil {
		log.Errorf(common.HTTP_TOKEN, "degrade agent", "send list request error:%v", err)
		return false
	}
	var moduleInfoList []ModuleInfo
	err = json.Unmarshal([]byte(resp.Data), &moduleInfoList)
	if err != nil {
		log.Errorf(common.HTTP_TOKEN, "get module list", "error:%v", err)
		return false
	}
	var ids []string
	for _, v := range moduleInfoList {
		if v.IsActivated {
			ids = append(ids, v.Name)
		}
	}
	// 有激活的模块
	if len(ids) > 0 {
		var params = fmt.Sprintf(`ids=%s`, strings.Join(ids, ","))
		resp, err := HttpGet(this.httpClient, fmt.Sprintf(degradeUrl, this.ServerIp, this.ServerPort), params, token.Data)
		if err != nil {
			log.Errorf(common.HTTP_TOKEN, "degrade agent", "send degrade request error:%v", err)
			return false
		}
		if resp.Code != 200 {
			log.Errorf(common.HTTP_TOKEN, "degrade agent", "send degrade request error,resp.Code=%d", resp.Code)
			return false
		}
	}
	return true
}

// 获取token
func (this *JavaProcess) getToken() (*Response, error) {
	var params = fmt.Sprintf(`username=%s&password=%s`, this.cfg.Username, this.cfg.Password)
	return HttpPost(this.httpClient, fmt.Sprintf(loginUrl, this.ServerIp, this.ServerPort), params, "")
}

// GET 请求
func HttpGet(httpClient *http.Client, url string, params string, token string) (*Response, error) {
	return HttpUtil(httpClient, url, params, token, "GET")
}

// POST请求
func HttpPost(httpClient *http.Client, url string, params string, token string) (*Response, error) {
	return HttpUtil(httpClient, url, params, token, "POST")
}

func HttpUtil(httpClient *http.Client, url string, params string, token string, method string) (*Response, error) {
	var data = strings.NewReader(params)
	req, err := http.NewRequest(method, url, data)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authentication", token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var response Response
	err = json.Unmarshal(bodyText, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
