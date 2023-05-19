// Package db implements a service for interacting with the database.
package db

import (
	"fmt"
	"os"

	"github.com/boltdb/bolt"
	"github.com/charmbracelet/log"
)

const (
	hashKey  = "hash"
	saltKey  = "salt"
	emailKey = "email"
)

// Service is a service that interacts with the database.
type Service struct {
	db     *bolt.DB
	logger *log.Logger
}

// Connect connects to the database.
func (s *Service) Connect(dbName string, mode os.FileMode, options *bolt.Options) (err error) {
	s.db, err = bolt.Open(dbName, mode, options)
	return
}

// Close closes the database connection.
func (s *Service) Close() (err error) {
	err = s.db.Close()
	return
}

// SetLogger sets the logger.
func (s *Service) SetLogger(logger *log.Logger) {
	s.logger = logger
}

// InsertUser inserts a new user into the database.
func (s *Service) InsertUser(username, salt, hash, email string) (err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(username))
		if err != nil {
			return fmt.Errorf("create bucket \"%s\": %w", username, err)
		}
		err = b.Put([]byte(saltKey), []byte(salt))
		if err != nil {
			return fmt.Errorf("put salt: %w", err)
		}
		err = b.Put([]byte(hashKey), []byte(hash))
		if err != nil {
			return fmt.Errorf("put hash: %w", err)
		}
		err = b.Put([]byte(emailKey), []byte(email))
		if err != nil {
			return fmt.Errorf("put email: %w", err)
		}
		return nil
	})
	return
}

// GetSaltAndHash gets the salt and hash for a given username.
func (s *Service) GetSaltAndHash(username string) (salt, hash string, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(username))
		if b == nil {
			return fmt.Errorf("bucket \"%s\" does not exist", username)
		}
		salt = string(b.Get([]byte(saltKey)))
		hash = string(b.Get([]byte(hashKey)))
		return nil
	})
	return
}

// ChangePassword changes the password for a given username.
func (s *Service) ChangePassword(username, salt, hash string) (err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(username))
		if b == nil {
			return fmt.Errorf("bucket \"%s\" does not exist", username)
		}
		err = b.Put([]byte(saltKey), []byte(salt))
		if err != nil {
			return fmt.Errorf("put salt: %w", err)
		}
		err = b.Put([]byte(hashKey), []byte(hash))
		if err != nil {
			return fmt.Errorf("put hash: %w", err)
		}
		return nil
	})
	return
}

// GetEmail gets the email address for a given username.
func (s *Service) GetEmail(username string) (email string, err error) {
	err = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(username))
		if b == nil {
			return fmt.Errorf("bucket \"%s\" does not exist", username)
		}
		email = string(b.Get([]byte(emailKey)))
		return nil
	})
	return
}

func (s *Service) DeleteUser(username string) (err error) {
	err = s.db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(username))
		if err != nil {
			return fmt.Errorf("delete bucket \"%s\": %w", username, err)
		}
		return nil
	})
	return
}
