package java_process

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"jrasp-daemon/defs"
	"jrasp-daemon/zlog"
	"net/http"
	"strings"
)

const (
	shutdownUrl  = "http://%s:%s/jrasp/control/shutdown"
	loginUrl     = "http://%s:%s/jrasp/user/login"
	degradeUrl   = "http://%s:%s/jrasp/user/login"
	listUrl      = "http://%s:%s/jrasp/module/list"
	softFlushUrl = "http://%s:%s/jrasp/module/flush&force=false"
)

type Response struct {
	Code    int    `json:"code"`
	Data    string `json:"data"`
	Message string `json:"message"`
}

func (jp *JavaProcess) ExitInjectImmediately() bool {
	// 关闭注入
	success := jp.ShutDownAgent()
	if success {
		// 标记为成功退出状态
		jp.MarkExitInject()
		zlog.Infof(defs.WATCH_DEFAULT, "java agent exit", "java pid:%d,status:%t", jp.JavaPid, success)
	} else {
		// 标记为异常退出状态
		jp.MarkFailedExitInject()
		zlog.Errorf(defs.WATCH_DEFAULT, "[BUG] java agent exit failed", "java pid:%d,status:%t", jp.JavaPid, success)
	}
	return success
}

// 关闭注入
func (jp *JavaProcess) ShutDownAgent() bool {
	token, err := jp.getToken()
	if err != nil {
		zlog.Errorf(defs.HTTP_TOKEN, "shut down agent", "get http token err:%v", err)
		return false
	}
	resp, err := HttpGet(jp.httpClient, fmt.Sprintf(shutdownUrl, jp.ServerIp, jp.ServerPort), "", token.Data)
	if err != nil {
		zlog.Errorf(defs.HTTP_TOKEN, "shut down agent", "send shutdown request error:%v", err)
		return false
	}
	if resp.Code != 200 {
		zlog.Errorf(defs.HTTP_TOKEN, "shut down agent", "send shutdown request error,resp.Code=%d", resp.Code)
		return false
	}
	return true
}

// 降级冻结
func (jp *JavaProcess) DegradeAgent() bool {
	token, _ := jp.getToken()
	// 查询所有模块  listUrl
	resp, err := HttpGet(jp.httpClient, fmt.Sprintf(listUrl, jp.ServerIp, jp.ServerPort), "", token.Data)
	if err != nil {
		zlog.Errorf(defs.HTTP_TOKEN, "degrade agent", "send list request error:%v", err)
		return false
	}
	var moduleInfoList []ModuleInfo
	err = json.Unmarshal([]byte(resp.Data), &moduleInfoList)
	if err != nil {
		zlog.Errorf(defs.HTTP_TOKEN, "get module list", "error:%v", err)
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
		resp, err := HttpGet(jp.httpClient, fmt.Sprintf(degradeUrl, jp.ServerIp, jp.ServerPort), params, token.Data)
		if err != nil {
			zlog.Errorf(defs.HTTP_TOKEN, "degrade agent", "send degrade request error:%v", err)
			return false
		}
		if resp.Code != 200 {
			zlog.Errorf(defs.HTTP_TOKEN, "degrade agent", "send degrade request error,resp.Code=%d", resp.Code)
			return false
		}
	}
	return true
}

// 软刷新
func (jp *JavaProcess) SoftFlush() bool {
	token, _ := jp.getToken()
	// 查询所有模块  listUrl
	resp, err := HttpGet(jp.httpClient, fmt.Sprintf(softFlushUrl, jp.ServerIp, jp.ServerPort), "", token.Data)
	if err != nil {
		zlog.Errorf(defs.HTTP_TOKEN, "[BUG]soft flush module", "send flush request error:%v", err)
		return false
	}
	if resp.Code != 200 {
		zlog.Errorf(defs.HTTP_TOKEN, "[BUG]soft flush module", "error,resp.Code=%d", resp.Code)
		return false
	}
	return true
}

// 获取token
func (jp *JavaProcess) getToken() (*Response, error) {
	var params = fmt.Sprintf(`username=%s&password=%s`, jp.cfg.Username, jp.cfg.Password)
	return HttpPost(jp.httpClient, fmt.Sprintf(loginUrl, jp.ServerIp, jp.ServerPort), params, "")
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
