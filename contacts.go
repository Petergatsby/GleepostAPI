package main

func init() {
	base.HandleFunc("/contacts", goneHandler)
	base.HandleFunc("/contacts/{id:[0-9]+}", goneHandler)
	base.HandleFunc("/contacts/{id:[0-9]+}/", goneHandler)
}
