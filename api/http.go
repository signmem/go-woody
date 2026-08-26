package api

import (
        "net/http"
        "encoding/json"
        "github.com/signmem/go-woody/g"
)

type Dto struct {
        Msg     string          `json:"msg"`
        Data    interface{}     `json:"data"`
}


type Response struct {
	Code    int         `json:"code"`    // 业务码：0=成功，非0=失败
	Message string      `json:"message"` // 提示信息
	Data    interface{} `json:"data"`    // 数据体
}


type NotFoundResponse struct {
        Code    int             `json:"code"`
        Message interface{}     `json:"message"`
        Status  string  `json:"status"`
}

func init() {
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
                Code: http.StatusNotFound,
                Message: msg,
                Status: "Not Found",
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

func RenderFailJson(w http.ResponseWriter, code int, msg string) {
	RenderJson(w, Response{
		Code:    code,
		Message: msg,
		Data:    nil,
	})
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

func Start() error {

	address := g.Config().Http.Address
	port := g.Config().Http.Port
	listenAddr := address + ":" + port

	mux := http.NewServeMux()
	registerRoutes(mux)

	s := &http.Server{
		Addr:           listenAddr,
		Handler:        mux,
		MaxHeaderBytes: 1 << 30,
	}

	g.Logger.Infof("api server listening on: %s", listenAddr)
	return s.ListenAndServe()
}


func registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/_health_check",   healthCheckHandler)
	mux.HandleFunc("/api/hosts",       handleHosts)
	mux.HandleFunc("/api/hosts/",      handleHosts)
	mux.HandleFunc("/api/v2/domains",  handleDomains)
	mux.HandleFunc("/api/v2/domains/", handleDomains)
	mux.HandleFunc("/api/v2/hosts",    handleHostNames)
	mux.HandleFunc("/api/v2/hosts/",   handleHostNames)
	mux.HandleFunc("/api/v2/slave",    handleSlaveDomain)
	mux.HandleFunc("/api/v2/slave/",   handleSlaveDomain)
	mux.HandleFunc("/api/v2/soa",      handleSOA)
	mux.HandleFunc("/api/v2/soa/",     handleSOA)
}
