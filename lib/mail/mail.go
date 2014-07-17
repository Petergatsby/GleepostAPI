package mail

import (
	"fmt"
	"log"
	"net/mail"
	"net/smtp"
	"time"

	"github.com/draaglom/GleepostAPI/lib/gp"
)

//Mailer is able to send email.
type Mailer struct {
	config gp.EmailConfig
}

//NewHeader generates a Header with from and date pre-populated.
func (m *Mailer) NewHeader() mail.Header {
	h := mail.Header{}
	h["From"] = []string{m.config.FromHeader}
	h["Date"] = []string{time.Now().Truncate(time.Second).UTC().String()}
	return h
}

//New creates a Mailer.
func New(config gp.EmailConfig) *Mailer {
	return &Mailer{config: config}
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
func (m *Mailer) Send(to string, subject string, body string) (err error) {
	header := m.NewHeader()
	header["To"] = []string{to}
	header["Subject"] = []string{subject}
	auth := smtp.PlainAuth("", m.config.User, m.config.Pass, m.config.Server)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", m.config.Server, m.config.Port), auth, m.config.From, []string{to}, append(toBytes(header), []byte(body)...))
	log.Println(err)
	return
}

//SendHTML - Send, but with HTML
func (m *Mailer) SendHTML(to string, subject string, body string) (err error) {
	header := m.NewHeader()
	header["Content-Type"] = []string{"text/html; charset=\"UTF-8\""}
	header["To"] = []string{to}
	header["Subject"] = []string{subject}
	auth := smtp.PlainAuth("", m.config.User, m.config.Pass, m.config.Server)
	err = smtp.SendMail(fmt.Sprintf("%s:%d", m.config.Server, m.config.Port), auth, m.config.From, []string{to}, append(toBytes(header), []byte(body)...))
	log.Println(err)
	return

}
