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

// ParseMessageRequest 表示解析消息请求
type ParseMessageRequest struct {
	SOAPMessage string `json:"soapMessage"`
}

// ParseMessageResponse 表示解析消息响应
type ParseMessageResponse struct {
	MessageType string                 `json:"messageType"`
	Content     map[string]interface{} `json:"content"`
}

// BuildMessageRequest 表示构建消息请求
type BuildMessageRequest struct {
	MessageType string                 `json:"messageType"`
	Content     map[string]interface{} `json:"content"`
}

// BuildMessageResponse 表示构建消息响应
type BuildMessageResponse struct {
	SOAPMessage string `json:"soapMessage"`
}

// BuildFaultRequest 表示构建故障消息请求
type BuildFaultRequest struct {
	FaultCode   int    `json:"faultCode"`
	FaultString string `json:"faultString"`
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

// ParseMessage 解析TR069 SOAP消息为JSON格式
func (c *Client) ParseMessage(soapMessage string) (*ParseMessageResponse, error) {
	url := fmt.Sprintf("%s/messages/parse", c.baseURL)

	reqBody := ParseMessageRequest{
		SOAPMessage: soapMessage,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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

	// 解析嵌套的消息数据
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return nil, err
	}

	var parseResp ParseMessageResponse
	if err := json.Unmarshal(dataBytes, &parseResp); err != nil {
		return nil, err
	}

	return &parseResp, nil
}

// BuildMessage 将JSON格式转换为TR069 SOAP消息
func (c *Client) BuildMessage(messageType string, content map[string]interface{}) (string, error) {
	url := fmt.Sprintf("%s/messages/build", c.baseURL)

	reqBody := BuildMessageRequest{
		MessageType: messageType,
		Content:     content,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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

	// 解析嵌套的消息数据
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return "", err
	}

	var buildResp BuildMessageResponse
	if err := json.Unmarshal(dataBytes, &buildResp); err != nil {
		return "", err
	}

	return buildResp.SOAPMessage, nil
}

// BuildFault 构建TR069故障响应消息
func (c *Client) BuildFault(faultCode int, faultString string) (string, error) {
	url := fmt.Sprintf("%s/messages/fault", c.baseURL)

	reqBody := BuildFaultRequest{
		FaultCode:   faultCode,
		FaultString: faultString,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
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

	// 解析嵌套的消息数据
	dataBytes, err := json.Marshal(apiResp.Data)
	if err != nil {
		return "", err
	}

	var buildResp BuildMessageResponse
	if err := json.Unmarshal(dataBytes, &buildResp); err != nil {
		return "", err
	}

	return buildResp.SOAPMessage, nil
}

func main() {
	// 创建API客户端
	client := NewClient(baseURL, apiKey)

	// 示例1: 解析TR069 SOAP消息
	fmt.Println("解析TR069 SOAP消息:")
	soapMessage := `<SOAP-ENV:Envelope xmlns:SOAP-ENV="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <SOAP-ENV:Header>
    <cwmp:ID SOAP-ENV:mustUnderstand="1">1234</cwmp:ID>
  </SOAP-ENV:Header>
  <SOAP-ENV:Body>
    <cwmp:Inform>
      <DeviceId>
        <Manufacturer>Vendor</Manufacturer>
        <OUI>00AABB</OUI>
        <ProductClass>Gateway</ProductClass>
        <SerialNumber>ABC123456</SerialNumber>
      </DeviceId>
      <Event>
        <EventCode>2 PERIODIC</EventCode>
      </Event>
      <ParameterList>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.HardwareVersion</Name>
          <Value>v2.1</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.SoftwareVersion</Name>
          <Value>1.0.3</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:Inform>
  </SOAP-ENV:Body>
</SOAP-ENV:Envelope>`

	parsedMessage, err := client.ParseMessage(soapMessage)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("消息类型: %s\n", parsedMessage.MessageType)
	fmt.Printf("消息内容:\n")
	prettyJSON, _ := json.MarshalIndent(parsedMessage.Content, "", "  ")
	fmt.Println(string(prettyJSON))
	fmt.Println()

	// 示例2: 构建TR069 SOAP消息
	fmt.Println("构建TR069 SOAP消息:")
	content := map[string]interface{}{
		"parameterNames": []string{
			"Device.DeviceInfo.ModelName",
			"Device.DeviceInfo.SerialNumber",
		},
	}

	builtMessage, err := client.BuildMessage("GetParameterValues", content)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("SOAP消息:\n%s\n", builtMessage)
	fmt.Println()

	// 示例3: 构建故障消息
	fmt.Println("构建故障消息:")
	faultMessage, err := client.BuildFault(9005, "Invalid Parameter Name")
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("故障SOAP消息:\n%s\n", faultMessage)
}