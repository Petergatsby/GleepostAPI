//package GleepostAPI is a simple REST API for gleepost.com
package main

import (
	"log"
	"net/http"
	"time"

	"runtime"

	"github.com/draaglom/GleepostAPI/lib/conf"
	"github.com/gorilla/mux"

	_ "net/http/pprof"
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

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	log.Println("Starting HTTP server")
	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: r,
	}
	go cleanupUploads()
	server.ListenAndServe()
}
