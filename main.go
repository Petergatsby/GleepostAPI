//package GleepostAPI is a simple REST API for gleepost.com
package main

import (
	"net/http"
	_ "net/http/pprof"
	"time"

	"runtime"

	"github.com/draaglom/GleepostAPI/lib/conf"
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
	config := conf.GetConfig()
	api.Start()
	go api.FeedbackDaemon(60)
	if !config.DevelopmentMode {
		api.PeriodicSummary(time.Date(2014, time.April, 9, 8, 0, 0, 0, time.UTC), time.Duration(24*time.Hour))
	}
	var futures []conf.PostFuture
	for _, f := range config.Futures {
		futures = append(futures, f.ParseDuration())
	}
	go api.KeepPostsInFuture(30*time.Minute, futures)

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}
	server.ListenAndServe()
}
