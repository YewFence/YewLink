package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxRetries = 3

// httpClient 带超时的 HTTP 客户端，避免默认客户端无限等待
var httpClient = &http.Client{Timeout: 10 * time.Second}

// drainAndClose 读取并关闭 response body，确保底层连接可复用
func drainAndClose(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}

// httpDoWithRetry 对 HTTP 请求进行重试，网络层错误和 5xx 响应均会重试，4xx 不重试
func httpDoWithRetry(do func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	for i := range maxRetries {
		resp, err := do()
		if err != nil {
			// net/http 约定 err != nil 时 resp 可能非 nil，需关闭以防泄漏
			drainAndClose(resp)
			lastErr = err
		} else if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("服务端错误 (HTTP %d)", resp.StatusCode)
			drainAndClose(resp)
		} else {
			return resp, nil
		}

		if i < maxRetries-1 {
			delay := time.Duration(2<<i) * time.Second
			log.Printf("请求失败 (%d/%d)，%v 后重试: %v", i+1, maxRetries, delay, lastErr)
			time.Sleep(delay)
		}
	}
	return nil, lastErr
}

// 读取凭据文件内容（trim 换行符和空格）
func readCredentialFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("读取凭据文件 %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

// 调用 Universal Auth 登录接口，返回 accessToken
func fetchToken(host, clientID, clientSecret string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"clientId":     clientID,
		"clientSecret": clientSecret,
	})
	resp, err := httpDoWithRetry(func() (*http.Response, error) {
		return httpClient.Post(
			host+"/api/v1/auth/universal-auth/login",
			"application/json",
			bytes.NewReader(body),
		)
	})
	if err != nil {
		return "", fmt.Errorf("认证请求失败（已重试 %d 次）: %w", maxRetries, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("认证失败 (HTTP %d): %s", resp.StatusCode, string(raw))
	}

	var result struct {
		AccessToken string `json:"accessToken"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析认证响应失败: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("认证响应中 accessToken 为空")
	}
	return result.AccessToken, nil
}

// 调用 Infisical API 列出指定路径下的文件夹，返回文件夹名称列表
func discoverFolders(host, projectID, environment, path, token string) ([]string, error) {
	req, err := http.NewRequest("GET", host+"/api/v2/folders", nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	q := req.URL.Query()
	q.Set("projectId", projectID)
	q.Set("environment", environment)
	q.Set("path", path)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpDoWithRetry(func() (*http.Response, error) {
		return httpClient.Do(req)
	})
	if err != nil {
		return nil, fmt.Errorf("列举文件夹请求失败（已重试 %d 次）: %w", maxRetries, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("列举文件夹失败 (HTTP %d): %s", resp.StatusCode, string(raw))
	}

	var result struct {
		Folders []struct {
			Name string `json:"name"`
		} `json:"folders"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析文件夹列表失败: %w", err)
	}

	names := make([]string, 0, len(result.Folders))
	for _, f := range result.Folders {
		if f.Name != "" {
			names = append(names, f.Name)
		}
	}
	return names, nil
}
