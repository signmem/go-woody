package api

import (
        "net/http"
        "encoding/json"
        "github.com/signmem/go-woody/g"
)

func init() {
        healthCheck()
        apiControll()
}

type Dto struct {
        Msg     string          `json:"msg"`
        Data    interface{}     `json:"data"`
}


type NotFoundResponse struct {
        Code    int             `json:"code"`
        Message interface{}     `json:"message"`
        Status  string  `json:"status"`
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

func Start() {

        address := g.Config().Http.Address
        port := g.Config().Http.Port
        listen := address + ":" + port

        s := &http.Server{
                Addr:           listen,
                MaxHeaderBytes: 1 << 30,
        }

        g.Logger.Infof("listening %s", listen)
        g.Logger.Fatalln(s.ListenAndServe())
}

