package mail

import (
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
	"time"
)

//mailer is able to send email.
type mailer struct {
	fromHeader string
	from       string
	smtpUser   string
	smtpPass   string
	smtpServer string
	smtpPort   int
}

//Mailer is able to send email.
type Mailer interface {
	SendPlaintext(to, subject, body string) error
	SendHTML(to, subject, body string) error
}

//NewHeader generates a Header with from and date pre-populated.
func (m *mailer) newHeader() mail.Header {
	h := mail.Header{}
	h["From"] = []string{m.fromHeader}
	h["Date"] = []string{time.Now().Format(time.RFC1123Z)}
	return h
}

//New creates a Mailer.
func New(fromHeader, from, smtpUser, smtpPass, smtpServer string, smtpPort int) Mailer {
	return &mailer{fromHeader: fromHeader, from: from, smtpUser: smtpUser, smtpPass: smtpPass, smtpServer: smtpServer, smtpPort: smtpPort}
}

func toBytes(h mail.Header) []byte {
	var bytes []byte
	for k, v := range h {
		//Assuming that there will only ever be a single value for each header...
		bytes = append(bytes, []byte(k+": "+v[0]+"\r\n")...)
	}
	bytes = append(bytes, []byte("\r\n")...)
	return bytes
}

//Send an old fashioned (ascii) email to "to"
func (m *mailer) SendPlaintext(to string, subject string, body string) (err error) {
	header := m.newHeader()
	header["To"] = []string{to}
	header["Subject"] = []string{subject}
	auth := smtp.PlainAuth("", m.smtpUser, m.smtpPass, m.smtpServer)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", m.smtpServer, m.smtpPort), auth, m.from, []string{to}, append(toBytes(header), []byte(body)...))
	log.Println(err)
	return
}

//SendHTML - Send, but with HTML
func (m *mailer) SendHTML(to string, subject string, body string) (err error) {
	header := m.newHeader()
	header["Content-Type"] = []string{"text/html; charset=\"UTF-8\""}
	header["To"] = []string{to}
	header["Subject"] = []string{subject}
	auth := smtp.PlainAuth("", m.smtpUser, m.smtpPass, m.smtpServer)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", m.smtpServer, m.smtpPort), auth, m.from, []string{to}, append(toBytes(header), []byte(body)...))
	log.Println(err)
	return
}

//NewMock returns a Mailer which won't actually send mail.
func NewMock() Mailer {
	return &stubMailer{}
}

type stubMailer struct {
}

func (s *stubMailer) SendPlaintext(to, subject, body string) error {
	log.Println("Sending plaintext:", to, subject, body)
	return nil
}

func (s *stubMailer) SendHTML(to, subject, body string) error {
	log.Println("Sending html email:", to, subject, body)
	return nil
}
