package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/factory"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

type ParseRequest struct {
	Message string `json:"message"`
}

type ParseResponse struct {
	Success    bool                  `json:"success"`
	Method     string               `json:"method,omitempty"`
	Parameters []interfaces.Parameter `json:"parameters,omitempty"`
	Error      string               `json:"error,omitempty"`
	Details    string               `json:"details,omitempty"`
}

func main() {
	r := mux.NewRouter()

	// API路由 - 必须在静态文件路由之前
	api := r.PathPrefix("/api").Subrouter()
	
	// 解析TR069消息的API端点
	api.HandleFunc("/parse", parseHandler).Methods("POST")
	api.HandleFunc("/health", healthHandler).Methods("GET")

	// 静态文件服务
	frontendDir := "../frontend"
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		frontendDir = "./frontend"
	}
	
	absPath, _ := filepath.Abs(frontendDir)
	log.Printf("Serving static files from: %s", absPath)
	
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(frontendDir)))

	// CORS处理
	handler := corsMiddleware(r)

	port := ":8080"
	log.Printf("TR069 Debug UI Server starting on http://localhost%s", port)
	log.Printf("Press Ctrl+C to stop the server")
	
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func parseHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := ParseResponse{
			Success: false,
			Error:   "Invalid JSON format",
			Details: err.Error(),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.Message == "" {
		response := ParseResponse{
			Success: false,
			Error:   "Message field is required",
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// 使用工厂创建解析器
	parserFactory := factory.NewParserFactory()
	parser := parserFactory.CreateParser()
	ctx := context.Background()
	
	// 解析消息
	message, err := parser.ParseMessage(ctx, []byte(req.Message))
	if err != nil {
		response := ParseResponse{
			Success: false,
			Error:   "Failed to parse TR069 message",
			Details: err.Error(),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// 返回解析结果
	response := ParseResponse{
		Success:    true,
		Method:     message.Method,
		Parameters: message.Parameters,
	}
	
	json.NewEncoder(w).Encode(response)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status": "healthy",
		"parser": "factory",
	}
	json.NewEncoder(w).Encode(response)
}