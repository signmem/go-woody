package api

import (
	_ "fmt"
	"net/http"
	"strings"
	"github.com/signmem/go-woody/g"
)


func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func handleHosts(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
		handleHostsPost(w, r)
	case http.MethodGet:
		handleHostsGet(w, r)
	case http.MethodPut:
		handleHostsPut(w, r)
	case http.MethodDelete:
		handleHostsDelete(w, r)
	default:
		// 不支持的方法
		RenderFailJson(w, http.StatusMethodNotAllowed, "method not allowed")
	}

}

// handleHostsPost POST /api/hosts 添加域名
func handleHostsPost(w http.ResponseWriter, r *http.Request) {
	data, err := dnsAdd(r)
	if err != nil {
		g.Logger.Errorf("dnsAdd failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}

// handleHostsGet GET /api/hosts 查询域名
func handleHostsGet(w http.ResponseWriter, r *http.Request) {
	dnsInfo, err := dnsGet(r)
	if err != nil {
		g.Logger.Errorf("dnsGet failed: %v", err)
		RenderFailJson(w, http.StatusInternalServerError, err.Error())
		return
	}
	RenderJson(w, dnsInfo)
}

func handleHostsPut(w http.ResponseWriter, r *http.Request) {
	data, err := dnsModify(r)
	if err != nil {
		g.Logger.Errorf("dnsModify failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}

// handleHostsDelete DELETE /api/hosts 删除域名
func handleHostsDelete(w http.ResponseWriter, r *http.Request) {
	data, err := dnsDelete(r)
	if err != nil {
		g.Logger.Errorf("dnsDelete failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}


// isContentTypeJson 校验请求是否为 JSON
func isContentTypeJson(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return strings.Contains(ct, "application/json")
}



func handleDomains(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
		handleHDomainPost(w, r)
	case http.MethodDelete:
		handleDomainDelete(w, r)
	case http.MethodGet:
		handleDomainGet(w, r)
	case http.MethodOptions:
		// CORS header
		w.WriteHeader(http.StatusNoContent)
	default:
		// 不支持的方法
		RenderFailJson(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}


// handleDomainGet GET /api/hosts 查询域名
func handleDomainGet(w http.ResponseWriter, r *http.Request) {

	domainInfo, err := domainGet(r)
	if err != nil {
		g.Logger.Errorf("domainGet failed: %v", err)
		RenderFailJson(w, http.StatusInternalServerError, err.Error())
		return
	}
	RenderJson(w, domainInfo)
}

// handleHDomainPost POST /api/domains 添加域名
func handleHDomainPost(w http.ResponseWriter, r *http.Request) {

	// max 8MB BODY limit
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024*1024)

	data, err := domainAdd(r)
	if err != nil {
		g.Logger.Errorf("domainAdd failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}


// handleDomainDelete DELETE /api/domains 删除域名
func handleDomainDelete(w http.ResponseWriter, r *http.Request) {

	data, err := domainDelete(r)
	if err != nil {
		g.Logger.Errorf("domainDelete failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}