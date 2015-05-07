package engine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/nicholaskh/golib/server"
	log "github.com/nicholaskh/log4go"
	"labix.org/v2/mgo/bson"
)

type PushdLongPollingServer struct {
	*server.HttpServer
	sessTimeout time.Duration
}

func NewPushdLongPollingServer(name string) *PushdLongPollingServer {
	this := new(PushdLongPollingServer)
	this.HttpServer = server.NewHttpServer(name)

	return this
}

func (this *PushdLongPollingServer) Launch(listenAddr string, sessTimeout time.Duration) {
	this.sessTimeout = sessTimeout
	this.HttpServer.RegisterHandler("/sub/{channel}/{ts}", this.ServeSubscribe)
	this.HttpServer.RegisterHandler("/pub/{channel}/{msg}", this.ServePublish)
	this.HttpServer.RegisterHandler("/history/{channel}/{ts}", this.ServeHistory)
	this.HttpServer.Launch(listenAddr)
}

func (this *PushdLongPollingServer) ServeSubscribe(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	channel := vars["channel"]
	tsStr := vars["ts"]
	ts, err := strconv.Atoi(tsStr)
	if err != nil {
		io.WriteString(w, "invalid timestamp")
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	conn, _, err := hj.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
	conn.Write([]byte("Access-Control-Allow-Origin: *\r\n"))
	conn.Write([]byte("\r\n"))

	//fetch history first
	if ts != 0 {
		hisRet, err := history(channel, int64(ts)+1)
		for i, hisRetEle := range hisRet {
			retEleBson, _ := hisRetEle.(bson.M)
			retEleBson["ts"] = strconv.Itoa(int(retEleBson["ts"].(int64)))
			hisRet[i] = retEleBson
		}
		if err != nil {
			log.Error(err)
		}
		if len(hisRet) > 0 {
			var retBytes []byte
			retBytes, err = json.Marshal(hisRet)

			conn.Write(retBytes)
			conn.Close()
			return
		}
	}

	c := server.NewClient(conn, time.Now(), this.sessTimeout, server.CONN_TYPE_LONG_POLLING)
	client := NewClient()
	client.Client = c
	client.OnClose = client.Close

	subscribe(client, channel)

	if this.sessTimeout.Nanoseconds() > int64(0) {
		go client.Client.CheckTimeout()
	}

	for {
		_, err := client.Conn.Read(make([]byte, 1460))

		if err != nil {
			if err == io.EOF {
				log.Info("Client end polling: %s", client.Conn.RemoteAddr())
				client.Close()
				client.Client.Close()
				return
			} else {
				log.Info("Client polling read error: %s", client.Conn.RemoteAddr())
				return
			}
		}
	}
}

func (this *PushdLongPollingServer) ServeHistory(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	vars := mux.Vars(req)
	channel := vars["channel"]
	ts := vars["ts"]
	tsInt, err := strconv.Atoi(ts)
	if err != nil {
		io.WriteString(w, "invalid timestamp")
		return
	}

	hisRet, err := history(channel, int64(tsInt))
	if err != nil {
		io.WriteString(w, fmt.Sprintf("fetch history of channel[%s] from[%d] error", channel, tsInt))
		return
	}
	var retBytes []byte
	retBytes, err = json.Marshal(hisRet)

	ret := string(retBytes)
	io.WriteString(w, ret)
}

func (this *PushdLongPollingServer) ServePublish(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	vars := mux.Vars(req)
	channel := vars["channel"]
	msg := vars["msg"]

	ret := publish(channel, msg, false)
	io.WriteString(w, ret)
}
