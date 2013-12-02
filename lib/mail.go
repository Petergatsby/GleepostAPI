package lib

import (
	"fmt"
	"github.com/draaglom/GleepostAPI/lib/gp"
	"log"
	"net/smtp"
)

type Header struct {
	Headers []byte
}

func NewHeader() Header {
	conf := gp.GetConfig()
	h := Header{}
	h.Headers = []byte("From: " + conf.Email.FromHeader + "\r\n")
	return h
}

func (h *Header) To(address string) {
	h.Headers = append(h.Headers, []byte("To: "+address+"\r\n")...)
	return
}

func (h *Header) HTML() {
	h.Headers = append(h.Headers, []byte("Content-Type: text/html; charset=\"UTF-8\"\r\n")...)
	return
}

func (h *Header) Subject(subject string) {
	h.Headers = append(h.Headers, []byte("Subject: "+subject+"\r\n")...)
	return
}

func send(to string, subject string, body string) (err error) {
	conf := gp.GetConfig()
	header := NewHeader()
	header.To(to)
	header.Subject(subject)
	auth := smtp.PlainAuth("", conf.Email.User, conf.Email.Pass, conf.Email.Server)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", conf.Email.Server, conf.Email.Port), auth, conf.Email.From, []string{to}, append(header.Headers, []byte(body)...))
	log.Println(err)
	return
}

func sendHTML(to string, subject string, body string) (err error) {
	conf := gp.GetConfig()
	header := NewHeader()
	header.HTML()
	header.To(to)
	header.Subject(subject)
	auth := smtp.PlainAuth("", conf.Email.User, conf.Email.Pass, conf.Email.Server)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", conf.Email.Server, conf.Email.Port), auth, conf.Email.From, []string{to}, append(header.Headers, []byte(body)...))
	log.Println(err)
	return

}
