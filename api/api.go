package api

import (
	"fmt"
	"net/http"
)

func healthCheck() {
	http.HandleFunc("/_health_check",
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})
}


func apiControll() {

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


