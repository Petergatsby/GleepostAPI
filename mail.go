package main

import (
	"net/smtp"
	"log"
)

func send(addrs []string) (err error) {
	conf := GetConfig()
	auth := smtp.PlainAuth("", conf.Email.User, conf.Email.Pass, conf.Email.Server)
	err = smtp.SendMail(conf.Email.Server, auth, conf.Email.From, addrs, []byte("sup"))
	log.Println(err)
	return
}
