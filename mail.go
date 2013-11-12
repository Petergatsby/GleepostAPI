package main

import (
	"net/smtp"
	"log"
	"fmt"
)


func send(addr string) (err error) {
	conf := GetConfig()
	headers := []byte("From: Gleepost <contact@gleepost.com>\r\n")
	headers = append(headers, []byte("To: "+addr+"\r\n")...)
	headers = append(headers, []byte("Subject: Sup\r\n")...)
	auth := smtp.PlainAuth("", conf.Email.User, conf.Email.Pass, conf.Email.Server)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", conf.Email.Server, conf.Email.Port), auth, conf.Email.From, []string{addr}, append(headers, []byte("sup")...))
	log.Println(err)
	return
}
