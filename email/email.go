package email

import (
	"fmt"
	"log"
	"net/smtp"
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
		log.Printf("Error sending email: %s\n", err)
	}
	return
}
