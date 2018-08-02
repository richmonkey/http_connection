package main

import "net/http"
import "encoding/json"
import "os"
import "time"
import "strings"
import "runtime"
import "runtime/pprof"
import log "github.com/golang/glog"

type ServerSummary struct {
	nconnections      int64
	nclients          int64
	nerrors           int64
}

func NewServerSummary() *ServerSummary {
	s := new(ServerSummary)
	return s
}

//用户长连接
func UserConnection(rw http.ResponseWriter, req *http.Request) {
	cn, ok := rw.(http.CloseNotifier)
	if !ok {
		rw.WriteHeader(400)
		return
	}	

	auth := req.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		WriteHttpError(400, "no auth", rw)
		return
	}
	token := strings.TrimPrefix(auth, "Bearer ")
	appid, uid, _, _, err := LoadUserAccessToken(token)
	if err != nil {
		WriteHttpError(400, "auth error", rw)
		return
	}
	if appid == 0 || uid == 0 {
		WriteHttpError(400, "auth error", rw)
		return
	}

	user_manager.AddUser(appid, uid)

	deliver := GetUserStateDeliver(uid)
	deliver.DeliverUserState(appid, uid)
	
	ch := cn.CloseNotify()	
	select {
	case <- ch:
		log.Infof("http connection closed")
	case <- time.After(60*time.Second):
		log.Infof("http connection not closed")
	}

	user_manager.RemoveUser(appid, uid)
	deliver.DeliverUserState(appid, uid)

	resp := make(map[string]interface{})
	resp["success"] = "true"
	WriteHttpObj(resp, rw)
}


func Summary(rw http.ResponseWriter, req *http.Request) {
	obj := make(map[string]interface{})
	obj["goroutine_count"] = runtime.NumGoroutine()
	obj["connection_count"] = server_summary.nconnections
	obj["client_count"] = server_summary.nclients
	obj["error_count"] = server_summary.nerrors

	res, err := json.Marshal(obj)
	if err != nil {
		log.Info("json marshal:", err)
		return
	}

	rw.Header().Add("Content-Type", "application/json")
	_, err = rw.Write(res)
	if err != nil {
		log.Info("write err:", err)
	}
	return
}

func Stack(rw http.ResponseWriter, req *http.Request) {
	pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
	rw.WriteHeader(200)
}

func WriteHttpError(status int, err string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	obj := make(map[string]interface{})
	meta := make(map[string]interface{})
	meta["code"] = status
	meta["message"] = err
	obj["meta"] = meta
	b, _ := json.Marshal(obj)
	w.WriteHeader(status)
	w.Write(b)
}

func WriteHttpObj(data map[string]interface{}, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	obj := make(map[string]interface{})
	obj["data"] = data
	b, _ := json.Marshal(obj)
	w.Write(b)
}
