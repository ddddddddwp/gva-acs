package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
)

// StartTR069Server 启动TR-069服务器，监听7547端口
func StartTR069Server() {
	// 创建HTTP服务器
	server := &http.Server{
		Addr:         ":7547",
		Handler:      http.HandlerFunc(handleTR069Request),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// 启动服务器
	global.GVA_LOG.Info("TR-069服务器启动，监听端口: 7547")
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		global.GVA_LOG.Error(fmt.Sprintf("TR-069服务器启动失败: %v", err))
	}
}

// handleTR069Request 处理TR-069请求
func handleTR069Request(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	
	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("读取请求体失败: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	
	// 记录请求信息
	global.GVA_LOG.Info(fmt.Sprintf("收到TR-069请求: %s", string(body)))
	
	// 简单响应
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<SOAP-ENV:Envelope xmlns:SOAP-ENV=\"http://schemas.xmlsoap.org/soap/envelope/\" xmlns:SOAP-ENC=\"http://schemas.xmlsoap.org/soap/encoding/\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:xsd=\"http://www.w3.org/2001/XMLSchema\" xmlns:cwmp=\"urn:dslforum-org:cwmp-1-0\"><SOAP-ENV:Header></SOAP-ENV:Header><SOAP-ENV:Body></SOAP-ENV:Body></SOAP-ENV:Envelope>"))
}