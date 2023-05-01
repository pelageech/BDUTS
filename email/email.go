package email

import (
	"fmt"
	"net/smtp"

	"github.com/charmbracelet/log"
)

type Sender struct {
	auth smtp.Auth
	from string
	host string
	port string
}

func New(username, password, host, port string) *Sender {
	return &Sender{
		auth: smtp.PlainAuth(
			"",
			username,
			password,
			host,
		),
		from: username,
		host: host,
		port: port,
	}
}

func (s *Sender) Send(to, subject, body string) (err error) {
	msg := fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	if err = smtp.SendMail(addr, s.auth, s.from, []string{to}, []byte(msg)); err != nil {
		log.Error("Failed to send email", "err", err)
	}
	return
}
