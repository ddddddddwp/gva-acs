package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	baseURL = "http://localhost:8080/api/v1"
	apiKey  = "your-api-key-here"
)

// APIResponse 表示API响应的通用结构
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError 表示API错误响应
type APIError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Device 表示TR069设备
type Device struct {
	ID             string            `json:"id"`
	SerialNumber   string            `json:"serialNumber"`
	Manufacturer   string            `json:"manufacturer"`
	Model          string            `json:"model"`
	Status         string            `json:"status"`
	LastInform     string            `json:"lastInform"`
	IPAddress      string            `json:"ipAddress"`
	Parameters     map[string]string `json:"parameters,omitempty"`
	HardwareVersion string           `json:"hardwareVersion,omitempty"`
	SoftwareVersion string           `json:"softwareVersion,omitempty"`
}

// DeviceListResponse 表示设备列表响应
type DeviceListResponse struct {
	Devices    []Device   `json:"devices"`
	Pagination Pagination `json:"pagination"`
}

// Pagination 表示分页信息
type Pagination struct {
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Pages int `json:"pages"`
}

// Parameter 表示设备参数
type Parameter struct {
	Name     string      `json:"name"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type"`
	Writable bool        `json:"writable,omitempty"`
}

// ParameterSetRequest 表示设置参数请求
type ParameterSetRequest struct {
	Parameters []Parameter `json:"parameters"`
}

// Client 是TR069 REST API的客户端
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient 创建一个新的API客户端
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// ListDevices 获取设备列表
func (c *Client) ListDevices(page, limit int, status string) ([]Device, *Pagination, error) {
	url := fmt.Sprintf("%s/devices?page=%d&limit=%d", c.baseURL, page, limit)
	if status != "" {
		url += "&status=" + status
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, nil, fmt.Errorf("API error: %d", resp.StatusCode)
		}
		return nil, nil, fmt.Errorf("API error: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, nil, err
	}

	// 解析嵌套的设备列表数据
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, nil, err
	}

	var deviceListResp DeviceListResponse
	if err := json.Unmarshal(dataBytes, &deviceListResp); err != nil {
		return nil, nil, err
	}

	return deviceListResp.Devices, &deviceListResp.Pagination, nil
}

// GetDevice 获取设备详情
func (c *Client) GetDevice(deviceID string) (*Device, error) {
	url := fmt.Sprintf("%s/devices/%s", c.baseURL, deviceID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, fmt.Errorf("API error: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("API error: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	// 解析嵌套的设备数据
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, err
	}

	var device Device
	if err := json.Unmarshal(dataBytes, &device); err != nil {
		return nil, err
	}

	return &device, nil
}

// SetDeviceParameters 设置设备参数
func (c *Client) SetDeviceParameters(deviceID string, parameters []Parameter) (string, error) {
	url := fmt.Sprintf("%s/devices/%s/parameters", c.baseURL, deviceID)

	reqBody := ParameterSetRequest{
		Parameters: parameters,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return "", fmt.Errorf("API error: %d", resp.StatusCode)
		}
		return "", fmt.Errorf("API error: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return "", err
	}

	// 解析任务ID
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return "", err
	}

	var taskResp struct {
		Status string `json:"status"`
		TaskID string `json:"taskId"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(dataBytes, &taskResp); err != nil {
		return "", err
	}

	return taskResp.TaskID, nil
}

// GetDeviceParameters 获取设备参数
func (c *Client) GetDeviceParameters(deviceID string, names []string, path string) ([]Parameter, error) {
	url := fmt.Sprintf("%s/devices/%s/parameters", c.baseURL, deviceID)
	
	// 添加查询参数
	if len(names) > 0 {
		url += "?names=" + names[0]
		for i := 1; i < len(names); i++ {
			url += "," + names[i]
		}
	}
	
	if path != "" {
		if len(names) > 0 {
			url += "&path=" + path
		} else {
			url += "?path=" + path
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var apiResp APIResponse
		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, fmt.Errorf("API error: %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("API error: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	// 解析参数列表
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, err
	}

	var paramResp struct {
		Parameters []Parameter `json:"parameters"`
	}
	if err := json.Unmarshal(dataBytes, &paramResp); err != nil {
		return nil, err
	}

	return paramResp.Parameters, nil
}

func main() {
	// 创建API客户端
	client := NewClient(baseURL, apiKey)

	// 示例1: 列出所有在线设备
	fmt.Println("列出所有在线设备:")
	devices, pagination, err := client.ListDevices(1, 10, "online")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("找到 %d 个设备 (总共 %d 个)\n", len(devices), pagination.Total)
	for _, device := range devices {
		fmt.Printf("- %s (%s): %s\n", device.SerialNumber, device.Model, device.Status)
	}
	fmt.Println()

	// 如果有设备，获取第一个设备的详情
	if len(devices) > 0 {
		deviceID := devices[0].ID
		
		// 示例2: 获取设备详情
		fmt.Printf("获取设备 %s 的详情:\n", deviceID)
		device, err := client.GetDevice(deviceID)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Printf("设备信息:\n")
			fmt.Printf("- 序列号: %s\n", device.SerialNumber)
			fmt.Printf("- 制造商: %s\n", device.Manufacturer)
			fmt.Printf("- 型号: %s\n", device.Model)
			fmt.Printf("- 硬件版本: %s\n", device.HardwareVersion)
			fmt.Printf("- 软件版本: %s\n", device.SoftwareVersion)
			fmt.Printf("- 状态: %s\n", device.Status)
			fmt.Printf("- 最后通知时间: %s\n", device.LastInform)
			fmt.Printf("- IP地址: %s\n", device.IPAddress)
		}
		fmt.Println()

		// 示例3: 获取设备参数
		fmt.Printf("获取设备 %s 的WiFi参数:\n", deviceID)
		params, err := client.GetDeviceParameters(deviceID, nil, "Device.WiFi")
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Printf("WiFi参数:\n")
			for _, param := range params {
				fmt.Printf("- %s = %v (%s)\n", param.Name, param.Value, param.Type)
			}
		}
		fmt.Println()

		// 示例4: 设置设备参数
		fmt.Printf("设置设备 %s 的WiFi参数:\n", deviceID)
		parameters := []Parameter{
			{
				Name:  "Device.WiFi.SSID.1.SSID",
				Value: "MyNetwork",
				Type:  "string",
			},
			{
				Name:  "Device.WiFi.SSID.1.Enable",
				Value: true,
				Type:  "boolean",
			},
		}
		
		taskID, err := client.SetDeviceParameters(deviceID, parameters)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
		} else {
			fmt.Printf("参数设置请求已提交，任务ID: %s\n", taskID)
		}
	}
}