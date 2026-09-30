package api

import (
	"net/http"
	"github.com/signmem/go-woody/g"
	"mime"
)


func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}


func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 跨域响应头
		// 生产环境把 * 修改为你的前端域名，例如 "https://admin.xxx.com"
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// 预检OPTIONS请求，直接返回204，不继续执行后续handler
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 非OPTIONS请求，继续执行业务handler
		next.ServeHTTP(w, r)
	})
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
	// max 8MB BODY limit
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024*1024)
	data, err := dnsAdd(r)
	if err != nil {
		g.Logger.Errorf("[v1-add] dnsAdd failed: %v", err)
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
		g.Logger.Errorf("[v1-modify] dnsModify failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}

// handleHostsDelete DELETE /api/hosts 删除域名
func handleHostsDelete(w http.ResponseWriter, r *http.Request) {
	data, err := dnsDelete(r)
	if err != nil {
		g.Logger.Errorf("[v1-delete] dnsDelete failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}


// isContentTypeJson 校验请求是否为 JSON
func isContentTypeJson(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return mediaType == "application/json"
}



func handleDomains(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
		handleHDomainPost(w, r)
	case http.MethodDelete:
		handleDomainDelete(w, r)
	case http.MethodGet:
		handleDomainGet(w, r)
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
		g.Logger.Errorf("[v2-domain-add] domainAdd failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}


// handleDomainDelete DELETE /api/domains 删除域名
func handleDomainDelete(w http.ResponseWriter, r *http.Request) {

	data, err := domainDelete(r)
	if err != nil {
		g.Logger.Errorf("[v2-domain-delete] domainDelete failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}


func handleSOA(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
		handleSOAPOST(w, r)
	case http.MethodGet:
		handleSOAGet(w, r)
	default:
		// 不支持的方法
		RenderFailJson(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}


func handleSOAPOST(w http.ResponseWriter, r *http.Request) {
	data, err := domainSOAFlush(r)
	if err != nil {
		g.Logger.Errorf("[v2-domain-soa] domainSOAFlush failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}


// handleSOAGet GET /api/v2/soa 查询 SOA
func handleSOAGet(w http.ResponseWriter, r *http.Request) {
	dnsInfo, err := domainSOAGet(r)
	if err != nil {
		g.Logger.Errorf("handleSOAGet failed: %v", err)
		RenderFailJson(w, http.StatusInternalServerError, err.Error())
		return
	}
	RenderJson(w, dnsInfo)
}



func handleHostNames(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
		handleHostnamePost(w, r)
	case http.MethodGet:
		handleHostnameGet(w, r)
	case http.MethodDelete:
		handleHostnameDelete(w, r)
	case http.MethodPut:
		handleHostnamePut(w, r)
	default:
		RenderFailJson(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleHostnamePost POST /api/v2/hosts 添加域名
func handleHostnamePost(w http.ResponseWriter, r *http.Request) {
	// max 8MB BODY limit
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024*1024)
	data, err := hostAdd(r)
	if err != nil {
		g.Logger.Errorf("[v2-host-add] hostAdd failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}

// handleHostsGet GET /api/hosts 查询域名
func handleHostnameGet(w http.ResponseWriter, r *http.Request) {
	dnsInfo, err := hostGet(r)
	if err != nil {
		g.Logger.Errorf("hostGet failed: %v", err)
		RenderFailJson(w, http.StatusInternalServerError, err.Error())
		return
	}
	RenderJson(w, dnsInfo)
}


// handleHostnameDelete DELETE /api/v2/hosts 删除 host
func handleHostnameDelete(w http.ResponseWriter, r *http.Request) {
	data, err := hostDelete(r)
	if err != nil {
		g.Logger.Errorf("[v2-host-delete] hostDelete failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}

// handleHostnamePut PUT /api/v2/hosts modify ipaddr only
func handleHostnamePut(w http.ResponseWriter, r *http.Request) {
	data, err := hostModify(r)
	if err != nil {
		g.Logger.Errorf("[v2-host-modify] hostModify failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}


func handleSlaveDomain(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPost:
		handleDomainMetaPost(w, r)
	case http.MethodGet:
		handleDomainMetaGet(w, r)
	default:
		// 不支持的方法
		RenderFailJson(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}


func handleDomainMetaPost(w http.ResponseWriter, r *http.Request) {
	// max 8MB BODY limit
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024*1024)
	data, err := syncSlaveDomain(r)
	if err != nil {
		g.Logger.Errorf("[v2-domain-add] handleDomainMetaPost() failed: %v", err)
		RenderFailJson(w, http.StatusBadRequest, err.Error())
		return
	}
	RenderJson(w, data)
}


func handleDomainMetaGet(w http.ResponseWriter, r *http.Request) {
	dnsInfo, err := getSlaveDomain(r)
	if err != nil {
		g.Logger.Errorf("getSlaveDomain() failed: %v", err)
		RenderFailJson(w, http.StatusInternalServerError, err.Error())
		return
	}
	RenderJson(w, dnsInfo)
}
