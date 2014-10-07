//package GleepostAPI is a simple REST API for gleepost.com
package main

import (
	"net/http"
	_ "net/http/pprof"

	"runtime"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var (
	r    = mux.NewRouter()
	base = r.PathPrefix("/api/{version}").Subrouter()
)

func main() {
	ascii()
	runtime.GOMAXPROCS(runtime.NumCPU())
	conf := GetConfig()

	server := &http.Server{
		Addr:    ":" + conf.Port,
		Handler: r,
	}
	server.ListenAndServe()
}
