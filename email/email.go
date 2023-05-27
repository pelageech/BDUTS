// Package email implements a service for sending emails.
package email

import (
	"errors"
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

var (
	// ErrEmptyUsername is returned when the username is empty.
	ErrEmptyUsername = errors.New("username cannot be empty")
	// ErrEmptyPassword is returned when the password is empty.
	ErrEmptyPassword = errors.New("password cannot be empty")
	// ErrEmptyHost is returned when the host is empty.
	ErrEmptyHost = errors.New("host cannot be empty")
	// ErrEmptyPort is returned when the port is empty.
	ErrEmptyPort = errors.New("port cannot be empty")
	// ErrEmptyLogger is returned when the logger is nil.
	ErrEmptyLogger = errors.New("logger cannot be nil")
)

// Sender is a struct that contains all the configuration
// of the email sender.
type Sender struct {
	auth   smtp.Auth
	from   string
	host   string
	port   string
	logger *log.Logger
}

// New creates a new Sender.
func New(username, password, host, port string, logger *log.Logger) (*Sender, error) {
	if username == "" {
		return nil, ErrEmptyUsername
	}
	if password == "" {
		return nil, ErrEmptyPassword
	}
	if host == "" {
		return nil, ErrEmptyHost
	}
	if port == "" {
		return nil, ErrEmptyPort
	}
	if logger == nil {
		return nil, ErrEmptyLogger
	}

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
	}, nil
}

func (s *Sender) send(to, subject, body string) (err error) {
	msg := fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, body)
	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	if err = smtp.SendMail(addr, s.auth, s.from, []string{to}, []byte(msg)); err != nil {
		s.logger.Error("Failed to send email", "err", err)
	}
	return
}

// SendSignUpEmail sends an email with a temporary password to the user.
func (s *Sender) SendSignUpEmail(to, username, password string) (err error) {
	msg := fmt.Sprintf(msgNewUser, username, password)
	return s.send(to, subjectNewUser, msg)
}

// SendChangedPasswordEmail sends an email notifying the user
// that their password has been changed.
func (s *Sender) SendChangedPasswordEmail(to string) (err error) {
	return s.send(to, subjectPasswordChanged, msgPasswordChanged)
}
