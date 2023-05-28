package email

import (
	"errors"
	"os"
	"testing"

	"github.com/charmbracelet/log"
)

func TestNew(t *testing.T) {
	username := "username"
	password := "password"
	host := "host"
	port := "port"
	logger := log.New(os.Stdout)

	tests := []struct {
		name     string
		username string
		password string
		host     string
		port     string
		logger   *log.Logger
		err      error
	}{
		{
			name:     "empty username",
			username: "",
			password: password,
			host:     host,
			port:     port,
			logger:   logger,
			err:      ErrEmptyUsername,
		},
		{
			name:     "empty password",
			username: username,
			password: "",
			host:     host,
			port:     port,
			logger:   logger,
			err:      ErrEmptyPassword,
		},
		{
			name:     "empty host",
			username: username,
			password: password,
			host:     "",
			port:     port,
			logger:   logger,
			err:      ErrEmptyHost,
		},
		{
			name:     "empty port",
			username: username,
			password: password,
			host:     host,
			port:     "",
			logger:   logger,
			err:      ErrEmptyPort,
		},
		{
			name:     "nil logger",
			username: username,
			password: password,
			host:     host,
			port:     port,
			logger:   nil,
			err:      ErrEmptyLogger,
		},
		{
			name:     "all good",
			username: username,
			password: password,
			host:     host,
			port:     port,
			logger:   logger,
			err:      nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(test.username, test.password, test.host, test.port, test.logger)
			if errors.Is(err, test.err) == false {
				t.Errorf("expected %v, got %v", test.err, err)
			}
		})
	}
}
