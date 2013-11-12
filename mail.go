package main

import (
	"net/smtp"
	"log"
	"fmt"
)


func send(addrs []string) (err error) {
	conf := GetConfig()
	var headers []byte("From: Gleepost <contact@gleepost.com>\r\n")
	headers += []byte("To: "+addrs+"\r\n")
	headers += []byte("Subject: Sup\r\n")
	auth := smtp.PlainAuth("", conf.Email.User, conf.Email.Pass, conf.Email.Server)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", conf.Email.Server, conf.Email.Port), auth, conf.Email.From, addrs, []byte("sup"))
	log.Println(err)
	return
}
