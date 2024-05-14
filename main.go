package main

import (
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"time"
)

func logMiddleware(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		log.Printf("<-- [%s] %s", r.Method, r.URL)
		// 记录响应信息
		rr := httptest.NewRecorder()

		// 添加跨域
		headers := map[string]string{
			"Access-Control-Allow-Origin":      r.Header.Get("Origin"),
			"Access-Control-Allow-Methods":     "GET, POST, PUT, DELETE, PATCH, OPTIONS",
			"Access-Control-Allow-Credentials": "true",
		}
		for key, value := range headers {
			if w.Header().Get(key) == "" {
				w.Header().Set(key, value)
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
		} else {
			next.ServeHTTP(w, r)
		}

		end := time.Now()
		elapsed := end.Sub(begin)
		log.Printf("--> [%s] %d %s +%s", r.Method, rr.Code, r.URL, elapsed)
	})
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
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
}

func main() {
	http.HandleFunc("/proxy", logMiddleware(http.HandlerFunc(proxyHandler)))
	http.HandleFunc("/proxy/", logMiddleware(http.HandlerFunc(proxyHandler)))
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	log.Println("Server listening on port " + port)
	// 启动服务器
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
