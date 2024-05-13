package main

import (
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	http.HandleFunc("/proxy", func(w http.ResponseWriter, r *http.Request) {
		// 提取target参数
		target := r.URL.Query().Get("target")
		if target == "" {
			http.Error(w, "Illegal Parameters", http.StatusBadRequest)
			return
		}
		log.Println("target:", target)

		// 解析target参数以支持带查询的URL
		u, err := url.Parse(target)
		if err != nil {
			http.Error(w, "Invalid target URL", http.StatusBadRequest)
			return
		}

		// 设置CORS响应头
		// w.Header().Set("Access-Control-Allow-Origin", "*")
		// w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		// w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 创建代理请求
		proxyReq, err := http.NewRequest(r.Method, u.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 复制原始请求的头到代理请求，除了Host和Authorization
		for key, values := range r.Header {
			if key != "Host" && key != "Authorization" {
				for _, value := range values {
					proxyReq.Header.Add(key, value)
				}
			}
		}

		// 发送代理请求
		resp, err := http.DefaultClient.Do(proxyReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// 复制响应头到原始响应中，除了Set-Cookie（避免冲突）
		for key, values := range resp.Header {
			if key != "Set-Cookie" {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
		}

		// 设置原始响应的状态码
		w.WriteHeader(resp.StatusCode)

		// 写入响应体
		io.Copy(w, resp.Body)
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// 启动服务器
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
	log.Println("Server started on port " + port)
}
