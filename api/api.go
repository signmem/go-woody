package api

import (
	_ "fmt"
	"net/http"
	"strings"
	"github.com/signmem/go-woody/g"
)

/*
func healthCheck() {
	http.HandleFunc("/_health_check",
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})
}

func apiControll() {
	ttp.HandleFunc("/api/hosts", handleHosts)
}


*/



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


/*

	var hostsHandler =	func(w http.ResponseWriter, r *http.Request) {

			if r.Method == http.MethodPost {

				// api post 请求 添加域名

				data, err := dnsAdd(r)
				if err != nil {
					RenderErrorJson(w, data.Msg)
					return
				}

				RenderJson(w, data)
				return
			}

			if r.Method == http.MethodGet {

				// api get 请求 域名查询

				dnsInfo, err := dnsGet(r)

				if err != nil {
					msg := fmt.Sprintf("%s", err)
					RenderErrorJson(w, msg)
					return
				}

				RenderJson(w, dnsInfo)
				return
			}

			if r.Method == http.MethodDelete {

				data, err := dnsDelete(r)

				if err != nil {
					msg := fmt.Sprintf("%s", err)
					RenderErrorJson(w, msg)
					return
				}

				RenderJson(w, data)
				return
			}

			if r.Method == http.MethodPut {

				// api put 请求 修改域名

				data, err := dnsModify(r)

				if err != nil {
					msg := fmt.Sprintf("%s", err)
					RenderErrorJson(w, msg)
					return
				}

				RenderJson(w, data)
				return
			}
		}

	http.HandleFunc("/api/hosts", hostsHandler)
	http.HandleFunc("/api/hosts/", hostsHandler)
}

*/
