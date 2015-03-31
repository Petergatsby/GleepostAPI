//package GleepostAPI is a simple REST API for gleepost.com
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"time"

	"runtime"

	"github.com/draaglom/GleepostAPI/lib/conf"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

var (
	r    = mux.NewRouter().StrictSlash(true)
	base = r.PathPrefix("/api/{version}").Subrouter()
)

func main() {
	ascii()
	runtime.GOMAXPROCS(runtime.NumCPU())
	log.Println("Getting config")
	config := conf.GetConfig()
	log.Println("Starting API")
	api.Start()
	log.Println("Starting APNS feedback daemons")
	go api.FeedbackDaemon(60)
	if !config.DevelopmentMode {
		log.Println("Starting stats summary email daemon")
		api.PeriodicSummary(time.Date(2014, time.April, 9, 8, 0, 0, 0, time.UTC), time.Duration(24*time.Hour))
	}

	go api.KeepPostsInFuture(30 * time.Minute)

	log.Println("Starting HTTP server")
	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}
	server.ListenAndServe()
}
