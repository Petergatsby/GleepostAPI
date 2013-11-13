package main

import (
	"fmt"
	"log"
	"net/smtp"
)

type Header struct {
	Headers []byte
}

func NewHeader() Header {
	conf := GetConfig()
	h := Header{}
	h.Headers = []byte("From: " + conf.Email.FromHeader + "\n")
	return h
}

func (h Header) To(address string) {
	h.Headers = append(h.Headers, []byte("To: "+address+"\n")...)
	return
}

func (h Header) Subject(subject string) {
	h.Headers = append(h.Headers, []byte("Subject: "+subject+"\n")...)
	return
}

func send(to string, subject string, body string) (err error) {
	conf := GetConfig()
	header := NewHeader()
	header.To(to)
	header.Subject(subject)
	auth := smtp.PlainAuth("", conf.Email.User, conf.Email.Pass, conf.Email.Server)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", conf.Email.Server, conf.Email.Port), auth, conf.Email.From, []string{to}, append(header.Headers, []byte(body)...))
	log.Println(err)
	return
}
