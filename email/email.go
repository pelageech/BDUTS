package email

import (
	"fmt"
	"net/smtp"

	"github.com/charmbracelet/log"
)

const (
	subjectNewUser = "Your credentials for BDUTS load balancer"
	msgNewUser     = "Your username: %s\n" +
		"Your password: %s\n\n" +
		"Please log in and change your password.\n" +
		"By changing your temporary password, you're helping to ensure that your account is secure " +
		"and that only you have access to it. It's also an opportunity to choose a password " +
		"that's easy for you to remember, but difficult for others to guess."

	subjectPasswordChanged = "Your password has been changed"
	msgPasswordChanged     = "Your password has been changed.\nIf" +
		" you did not change your password, please contact the administrator."
)

type Sender struct {
	auth   smtp.Auth
	from   string
	host   string
	port   string
	logger *log.Logger
}

func New(username, password, host, port string, logger *log.Logger) *Sender {
	return &Sender{
		auth: smtp.PlainAuth(
			"",
			username,
			password,
			host,
		),
		from:   username,
		host:   host,
		port:   port,
		logger: logger,
	}
}

func (s *Sender) send(to, subject, body string) (err error) {
	msg := fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	if err = smtp.SendMail(addr, s.auth, s.from, []string{to}, []byte(msg)); err != nil {
		s.logger.Error("Failed to send email", "err", err)
	}
	return
}

func (s *Sender) SendSignUpEmail(to, username, password string) (err error) {
	msg := fmt.Sprintf(msgNewUser, username, password)
	return s.send(to, subjectNewUser, msg)
}

func (s *Sender) SendChangedPasswordEmail(to string) (err error) {
	return s.send(to, subjectPasswordChanged, msgPasswordChanged)
}
