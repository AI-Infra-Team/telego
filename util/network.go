package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// 请求配置结构体
type RequestConfig struct {
	URL     string
	Timeout time.Duration
}

// 默认配置常量
const defaultTimeout = 5 * time.Second

// CheckURLAccessibilityBuilder 模式实现
type CheckURLAccessibilityBuilder struct {
	config *RequestConfig
}

// 创建新的 Builder 实例
func NewCheckURLAccessibilityBuilder() *CheckURLAccessibilityBuilder {
	return &CheckURLAccessibilityBuilder{
		config: &RequestConfig{
			Timeout: defaultTimeout, // 默认超时值
		},
	}
}

// 设置 URL
func (b *CheckURLAccessibilityBuilder) SetURL(url string) *CheckURLAccessibilityBuilder {
	b.config.URL = url
	return b
}

// 设置 Timeout
func (b *CheckURLAccessibilityBuilder) SetTimeout(timeout time.Duration) *CheckURLAccessibilityBuilder {
	b.config.Timeout = timeout
	return b
}

// 直接调用 checkURLAccessibility
func (b *CheckURLAccessibilityBuilder) CheckAccessibility() error {
	client := http.Client{
		Timeout: b.config.Timeout,
	}

	// 发送 HEAD 请求
	resp, err := client.Head(b.config.URL)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil // 成功访问
	}

	return fmt.Errorf("HTTP 状态码错误: %d", resp.StatusCode)
}

// 获取文件总大小
func getFileSize(url string) (int64, error) {
	resp, err := http.Head(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch file size: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	size, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse content length: %v", err)
	}

	return size, nil
}

func ReadHttpSmallFile(url string) (string, error) {
	// 发送 HTTP GET 请求
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// 下载文件
func DownloadFile(url, filename string) error {
	fileDir := path.Dir(filename)
	err := os.MkdirAll(fileDir, 0755)
	if err != nil {
		fmt.Printf("downloadFile Error: %v, url: %s\n", err, url)
		return err
	}

	// 获取文件大小
	fileSize, err := getFileSize(url)
	if err != nil {
		fmt.Printf("downloadFile Error: %v, url: %s\n", err, url)
		return err
	}
	fmt.Printf("DownloadFile %s to %s with size: %d bytes\n", url, filename, fileSize)

	// 创建文件
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("downloadFile Error: %v, url: %s\n", err, url)
		return err
	}
	defer file.Close()

	// 初始化进度条
	bar := progressbar.DefaultBytes(
		fileSize,
		"Downloading",
	)

	// 自定义 HTTP 客户端，设置 5 秒超时
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
		},
		Timeout: 180 * time.Second, // 取消默认的请求超时，保持默认的行为
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 写入文件并更新进度条
	_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	fmt.Println(color.GreenString("Downloaded %s to %s", url, filename))
	return nil
}

// func UploadMultipleFilesInOneConnection(files []string, multipartApi string) (string, error) {
// 	if len(files) == 0 {
// 		return "", fmt.Errorf("no files to upload")
// 	}

// 	// Create a pipe to stream data
// 	pr, pw := io.Pipe()
// 	writer := multipart.NewWriter(pw)

// 	// Goroutine to write multipart data to the pipe
// 	go func() {
// 		defer pw.Close()
// 		defer writer.Close()

// 		for _, file := range files {
// 			// Open the file
// 			f, err := os.Open(file)
// 			if err != nil {
// 				pw.CloseWithError(fmt.Errorf("failed to open file %s: %w", file, err))
// 				return
// 			}
// 			defer f.Close()

// 			// Add file to the multipart writer
// 			part, err := writer.CreateFormFile("files", filepath.Base(file))
// 			if err != nil {
// 				pw.CloseWithError(fmt.Errorf("failed to create form file for %s: %w", file, err))
// 				return
// 			}
// 			if _, err := io.Copy(part, f); err != nil {
// 				pw.CloseWithError(fmt.Errorf("failed to copy file content for %s: %w", file, err))
// 				return
// 			}
// 		}
// 	}()

// 	// Create the HTTP request with the pipe reader
// 	req, err := http.NewRequest("POST", multipartApi, pr)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to create HTTP request: %w, url: %s", err, multipartApi)
// 	}
// 	req.Header.Set("Content-Type", writer.FormDataContentType())

// 	// Send the request
// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to upload files: %w, url: %s", err, multipartApi)
// 	}
// 	defer resp.Body.Close()

// 	// Check the response
// 	respBody, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to read response body: %w, url: %s", err, multipartApi)
// 	}

// 	if resp.StatusCode != http.StatusOK {
// 		return "", fmt.Errorf("failed to upload files, server returned: %s, url: %s", string(respBody), multipartApi)
// 	}

// 	return string(respBody), nil
// }

func HttpAuth(url string, user string, pw string) ([]byte, error) {
	// 创建基本认证
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// fmt.Printf("Error creating request: %v\n", err)
		return []byte{}, fmt.Errorf("failed to create request: %v, url: %s", err, url)
	}

	// 设置基本认证
	req.SetBasicAuth(user, pw)

	// 发送 GET 请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// fmt.Printf("Request failed: %v\n", err)
		return []byte{}, fmt.Errorf("failed to send request: %v, url: %s", err, url)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		// fmt.Printf("Request failed with status: %v\n", resp.Status)
		return []byte{}, fmt.Errorf("request failed with status: %v, url: %s", resp.Status, url)
	}

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// fmt.Printf("Error reading response body: %v\n", err)
		return []byte{}, fmt.Errorf("failed to read response body: %v, url: %s", err, url)
	}

	return body, nil

	// // 假设响应体是 JSON 格式并且包含 "token" 字段
	// var result map[string]interface{}
	// if err := json.Unmarshal(body, &result); err != nil {
	// 	fmt.Printf("Error unmarshalling response: %v\n", err)
	// 	return
	// }

	// // 获取 token
	// token, ok := result["token"].(string)
	// if !ok {
	// 	fmt.Println("No token found in response")
	// 	return
	// }
}

// HttpOneshot sends a POST request with the given URL and JSON object.
// It returns the response body as a string and an error if any.
func HttpOneshot(url string, jsonObj interface{}) ([]byte, error) {
	jsonData := []byte{}
	if jsonObj != nil {
		// Marshal the JSON object into a JSON byte array
		jsonData_, err := json.Marshal(jsonObj)
		if err != nil {
			return []byte{}, fmt.Errorf("failed to marshal JSON: %v", err)
		}
		jsonData = jsonData_
	}

	// Create a new HTTP POST request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return []byte{}, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read response body: %v", err)
	}

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return []byte{}, fmt.Errorf("received non-2xx response: %d, body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func UploadMultipleFilesInOneConnection(files []string, multipartApi string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files to upload")
	}

	// 创建管道，用于流式上传
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	// 创建 goroutine 写入文件数据到管道
	go func() {
		defer pw.Close()
		defer writer.Close()

		for _, file := range files {
			// 打开文件
			f, err := os.Open(file)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to open file %s: %w", file, err))
				return
			}
			defer f.Close()

			// 获取文件大小用于显示进度
			fileInfo, err := f.Stat()
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to get file info for %s: %w", file, err))
				return
			}
			fileSize := fileInfo.Size()

			// 创建进度条
			bar := progressbar.DefaultBytes(
				fileSize,
				fmt.Sprintf("Uploading %s", filepath.Base(file)),
			)

			// 添加文件到 multipart writer
			part, err := writer.CreateFormFile("files", filepath.Base(file))
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to create form file for %s: %w", file, err))
				return
			}

			// 使用 io.MultiWriter 同时写入进度条和目标 writer
			_, err = io.Copy(io.MultiWriter(part, bar), f)
			if err != nil {
				pw.CloseWithError(fmt.Errorf("failed to copy file content for %s: %w", file, err))
				return
			}
		}
	}()

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", multipartApi, pr)
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w, url: %s", err, multipartApi)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 自定义 HTTP 客户端，设置超时
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second, // 设置连接超时时间
			}).DialContext,
			ForceAttemptHTTP2: true,
			MaxIdleConns:      100,
		},
		Timeout: 10000 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to upload files: %w, url: %s", err, multipartApi)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to upload files, server returned: %s, url: %s", string(respBody), multipartApi)
	}

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w, url: %s", err, multipartApi)
	}

	fmt.Println(color.GreenString("Successfully uploaded all files to %s", multipartApi))
	return string(respBody), nil
}
