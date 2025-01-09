package push

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/silenceper/log"
)

func getToken(authHeader, username, password string) (string, error) {
	// 解析 Www-Authenticate 头
	realm, service, scope, err := parseAuthHeader(authHeader)
	if err != nil {
		log.Errorf("Failed to parse Www-Authenticate header: %v", err)
		return "", err
	}

	log.Infof("Parsed Www-Authenticate header - realm: %s, service: %s, scope: %s", realm, service, scope)

	// 构造请求参数
	data := url.Values{}
	data.Set("username", username)
	data.Set("password", password)
	data.Set("service", service)
	data.Set("scope", scope)

	log.Infof("Constructed token request data: %v", data)

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", realm, strings.NewReader(data.Encode()))
	if err != nil {
		log.Errorf("Failed to create token request: %v", err)
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	log.Infof("Sending token request to %s", realm)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Failed to send token request: %v", err)
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	log.Infof("Received token response with status code: %d", resp.StatusCode)

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		log.Errorf("Token request failed with status code: %d", resp.StatusCode)
		return "", fmt.Errorf("authentication failed: status code %d", resp.StatusCode)
	}

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Failed to read token response body: %v", err)
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// 解析 JSON 响应
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		log.Errorf("Failed to parse token response JSON: %v", err)
		return "", fmt.Errorf("failed to parse token response JSON: %v", err)
	}

	// 提取 token
	token, ok := result["token"].(string)
	if !ok || token == "" {
		log.Errorf("Token is empty or invalid in response")
		return "", fmt.Errorf("token is empty or invalid in response")
	}

	return token, nil
}

func parseAuthHeader(authHeader string) (string, string, string, error) {
	// 示例：Bearer realm="https://dockerauth.cn-hangzhou.cs.com/auth",service="registry.cs.com:cn-hangzhou:26842",scope="repository:404name/alist:pull"
	parts := strings.Split(authHeader, ",")
	if len(parts) < 3 {
		log.Errorf("Invalid Www-Authenticate header: %s", authHeader)
		return "", "", "", fmt.Errorf("invalid Www-Authenticate header: %s", authHeader)
	}

	// 提取 realm
	realm := strings.TrimPrefix(parts[0], "Bearer realm=")
	realm = strings.Trim(realm, `"`)

	// 提取 service
	service := strings.TrimPrefix(parts[1], "service=")
	service = strings.Trim(service, `"`)

	// 提取 scope
	scope := strings.TrimPrefix(parts[2], "scope=")
	scope = strings.Trim(scope, `"`)
	// TODO 需要判断下 默认添加下推送权限，不然通过header默认拿到的都是pull，导致推送的时候权限不够
	scope += ",push"

	log.Infof("Parsed Www-Authenticate header - realm: %s, service: %s, scope: %s", realm, service, scope)
	return realm, service, scope, nil
}
