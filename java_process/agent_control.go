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
	BASE_URL     = "http://%s:%s"
	shutdownUrl  = "http://%s:%s/jrasp/control/shutdown"
	loginUrl     = "http://%s:%s/jrasp/user/login"
	softFlushUrl = "http://%s:%s/jrasp/module/flush?force=false"
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
		// 退出后消息立即上报
		zlog.Infof(defs.AGENT_SUCCESS_EXIT, "java agent exit", `{"pid":%d,"status":"%s","startTime":"%s"}`, jp.JavaPid, jp.InjectedStatus, jp.StartTime)
	} else {
		// 标记为异常退出状态
		jp.MarkFailedExitInject()
		zlog.Errorf(defs.WATCH_DEFAULT, "[BUG] java agent exit failed", "java pid:%d,status:%t", jp.JavaPid, success)
	}
	return success
}

// ShutDownAgent 关闭注入
func (jp *JavaProcess) ShutDownAgent() bool {
	token, err := jp.getToken()
	if err != nil {
		zlog.Errorf(defs.HTTP_TOKEN, "shutdown java agent", "get http token err:%v", err)
		return false
	}
	resp, err := HttpGet(jp.httpClient, fmt.Sprintf(shutdownUrl, jp.ServerIp, jp.ServerPort), "", token.Data)
	if err != nil {
		zlog.Errorf(defs.HTTP_TOKEN, "shutdown java agent", "send shutdown request error:%v", err)
		return false
	}
	if resp.Code != 200 {
		zlog.Errorf(defs.HTTP_TOKEN, "shutdown java agent", "send shutdown request error,resp.Code=%d", resp.Code)
		return false
	}
	return true
}

// SoftFlush 软刷新
func (jp *JavaProcess) SoftFlush() bool {
	token, _ := jp.getToken()
	resp, err := HttpGet(jp.httpClient, fmt.Sprintf(softFlushUrl, jp.ServerIp, jp.ServerPort), "", token.Data)
	if err != nil {
		zlog.Errorf(defs.HTTP_TOKEN, "[BUG]soft flush module", "send flush request error:%v", err)
		return false
	}
	if resp.Code != 200 {
		zlog.Errorf(defs.HTTP_TOKEN, "[BUG]soft flush module", "error,resp.Code=%d", resp.Code)
		return false
	}
	zlog.Infof(defs.HTTP_TOKEN, "soft flush module", "success")
	return true
}

// 获取token
func (jp *JavaProcess) getToken() (*Response, error) {
	var params = fmt.Sprintf(`username=%s&password=%s`, jp.cfg.Username, jp.cfg.Password)
	return HttpPost(jp.httpClient, fmt.Sprintf(loginUrl, jp.ServerIp, jp.ServerPort), params, "")
}

// HttpGet GET 请求
func HttpGet(httpClient *http.Client, url string, params string, token string) (*Response, error) {
	return HttpUtil(httpClient, url, params, token, "GET")
}

// HttpPost POST请求
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
