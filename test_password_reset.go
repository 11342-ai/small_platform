package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// 测试忘记密码流程
func main1() {
	baseURL := "http://localhost:8080/api"

	// 1. 先注册一个测试用户
	fmt.Println("=== 1. 注册测试用户 ===")
	registerData := map[string]string{
		"username": "testuser",
		"password": "oldpassword123",
		"email":    "test@example.com",
	}

	registerResp := sendRequest("POST", baseURL+"/register", registerData)
	fmt.Printf("注册响应: %s\n\n", registerResp)

	// 等待一下
	time.Sleep(1 * time.Second)

	// 2. 发送验证码
	fmt.Println("=== 2. 发送验证码 ===")
	sendCodeData := map[string]string{
		"username":  "testuser",
		"code_type": "password_reset",
	}

	sendCodeResp := sendRequest("POST", baseURL+"/auth/send-code", sendCodeData)
	fmt.Printf("发送验证码响应: %s\n\n", sendCodeResp)

	// 注意：在实际测试中，你需要从控制台获取验证码
	// 这里我们假设验证码是 "123456"
	fmt.Println("=== 注意：请查看服务器控制台获取验证码 ===")
	fmt.Println("=== 假设验证码是 '123456' ===")

	// 3. 验证验证码
	fmt.Println("=== 3. 验证验证码 ===")
	verifyCodeData := map[string]string{
		"username":  "testuser",
		"code":      "123456",
		"code_type": "password_reset",
	}

	verifyCodeResp := sendRequest("POST", baseURL+"/auth/verify-code", verifyCodeData)
	fmt.Printf("验证验证码响应: %s\n\n", verifyCodeResp)

	// 4. 重置密码
	fmt.Println("=== 4. 重置密码 ===")
	resetPasswordData := map[string]string{
		"username":     "testuser",
		"code":         "123456",
		"new_password": "newpassword123",
	}

	resetPasswordResp := sendRequest("POST", baseURL+"/auth/reset-password", resetPasswordData)
	fmt.Printf("重置密码响应: %s\n\n", resetPasswordResp)

	// 5. 测试新密码登录
	fmt.Println("=== 5. 测试新密码登录 ===")
	loginData := map[string]string{
		"username": "testuser",
		"password": "newpassword123",
	}

	loginResp := sendRequest("POST", baseURL+"/login", loginData)
	fmt.Printf("新密码登录响应: %s\n\n", loginResp)

	fmt.Println("=== 测试完成 ===")
}

func sendRequest(method, url string, data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("JSON编码错误: %v", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Sprintf("创建请求错误: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("请求错误: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("读取响应错误: %v", err)
	}

	return fmt.Sprintf("状态码: %d, 响应: %s", resp.StatusCode, string(body))
}
