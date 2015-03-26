package main

import "net/http"

func init() {
	base.Handle("/contacts", timeHandler(api, http.HandlerFunc(goneHandler)))
	base.Handle("/contacts/{id:[0-9]+}", timeHandler(api, http.HandlerFunc(goneHandler)))
}
