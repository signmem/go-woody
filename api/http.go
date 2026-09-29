package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/signmem/go-woody/g"
)

type Dto struct {
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type Response struct {
	Code    int         `json:"code"`    // 业务码: 与 HTTP 状态码一致, 0=成功场景不使用
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 数据体
}

type NotFoundResponse struct {
	Code    int         `json:"code"`
	Message interface{} `json:"message"`
	Status  string      `json:"status"`
}

func RenderJson(w http.ResponseWriter, v interface{}) {
	bs, err := json.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(bs)
}

func RenderErrorJson(w http.ResponseWriter, msg interface{}) {

	reponse := NotFoundResponse{
		Code:    http.StatusNotFound,
		Message: msg,
		Status:  "Not Found",
	}

	bs, err := json.Marshal(reponse)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write(bs)
}

// RenderFailJson 输出失败响应。
// 修复: 原实现只把 code 写进 JSON body, HTTP 状态码恒为 200,
// 客户端与监控无法用 HTTP 语义判断成败; 现在同步设置真实状态码。
func RenderFailJson(w http.ResponseWriter, code int, msg string) {
	bs, err := json.Marshal(Response{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(code)
	w.Write(bs)
}

func RenderDataJson(w http.ResponseWriter, data interface{}) {
	RenderJson(w, Dto{Msg: "success", Data: data})
}

func RenderMsgJson(w http.ResponseWriter, msg string) {
	RenderJson(w, map[string]string{"msg": msg})
}

func AutoRender(w http.ResponseWriter, data interface{}, err error) {
	if err != nil {
		RenderMsgJson(w, err.Error())
		return
	}

	RenderDataJson(w, data)
}

// AuthMiddleware 简单 token 鉴权; 未配置 auth_token 时不启用 (兼容存量部署)。
// 支持两种携带方式:
//
//	X-Auth-Token: ***
//	Authorization: Bearer <token>
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := g.Config().AuthToken
		if token == "" || r.URL.Path == "/_health_check" {
			next.ServeHTTP(w, r)
			return
		}

		provided := r.Header.Get("X-Auth-Token")
		if provided == "" {
			if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
				provided = strings.TrimPrefix(h, "Bearer ")
			}
		}

		// 常数时间比较, 防时序侧信道
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			RenderFailJson(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// NewServer 构建带合理超时的 HTTP server。
// 修复:
//  1. 移除 MaxHeaderBytes: 1<<30 (1GB header 上限是 DoS 放大器), 使用默认 1MB;
//  2. 补齐 ReadHeaderTimeout/ReadTimeout/WriteTimeout/IdleTimeout, 防 Slowloris;
//  3. 返回 *http.Server, 由 main 负责 ListenAndServe 与 Shutdown (优雅退出)。
func NewServer() *http.Server {
	address := g.Config().Http.Address
	port := g.Config().Http.Port

	mux := http.NewServeMux()
	registerRoutes(mux)

	var handler http.Handler = mux
	handler = AuthMiddleware(handler)
	handler = CorsMiddleware(handler)

	return &http.Server{
		Addr:              address + ":" + port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/_health_check", healthCheckHandler)
	mux.HandleFunc("/api/hosts", handleHosts)
	mux.HandleFunc("/api/hosts/", handleHosts)
	mux.HandleFunc("/api/v2/domains", handleDomains)
	mux.HandleFunc("/api/v2/domains/", handleDomains)
	mux.HandleFunc("/api/v2/hosts", handleHostNames)
	mux.HandleFunc("/api/v2/hosts/", handleHostNames)
	mux.HandleFunc("/api/v2/slave", handleSlaveDomain)
	mux.HandleFunc("/api/v2/slave/", handleSlaveDomain)
	mux.HandleFunc("/api/v2/soa", handleSOA)
	mux.HandleFunc("/api/v2/soa/", handleSOA)
}
