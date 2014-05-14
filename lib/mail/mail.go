package mail

import (
	"fmt"
	"log"
	"net/smtp"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

type Mailer struct {
	config gp.EmailConfig
}

type Header struct {
	Headers []byte
}

func (m *Mailer) NewHeader() *Header {
	h := Header{}
	h.Headers = []byte("From: " + m.config.FromHeader + "\r\n")
	h.Headers = append(h.Headers, []byte("Date: "+time.Now().Truncate(time.Second).UTC().String()+"\r\n"))
	return &h
}

func New(config gp.EmailConfig) *Mailer {
	return &Mailer{config: config}
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

func (m *Mailer) Send(to string, subject string, body string) (err error) {
	header := m.NewHeader()
	header.To(to)
	header.Subject(subject)
	auth := smtp.PlainAuth("", m.config.User, m.config.Pass, m.config.Server)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", m.config.Server, m.config.Port), auth, m.config.From, []string{to}, append(header.Headers, []byte(body)...))
	log.Println(err)
	return
}

func (m *Mailer) SendHTML(to string, subject string, body string) (err error) {
	header := m.NewHeader()
	header.HTML()
	header.To(to)
	header.Subject(subject)
	auth := smtp.PlainAuth("", m.config.User, m.config.Pass, m.config.Server)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", m.config.Server, m.config.Port), auth, m.config.From, []string{to}, append(header.Headers, []byte(body)...))
	log.Println(err)
	return

}
