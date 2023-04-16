package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	passwordLength = 25
	saltLength     = 20
	minUsername    = 4
	maxUsername    = 20
)

type Service struct {
	db *sql.DB
}

func New(db *sql.DB) *Service {
	return &Service{
		db: db,
	}
}

type User struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func generateRandomPassword() (password string, err error) {
	passwordBytes := make([]byte, passwordLength)
	_, err = rand.Read(passwordBytes)
	if err != nil {
		log.Printf("Error generating random password: %s", err)
		return
	}
	password = base64.URLEncoding.EncodeToString(passwordBytes)
	return
}

func generateSalt() (salt string, err error) {
	saltBytes := make([]byte, saltLength)
	_, err = rand.Read(saltBytes)
	if err != nil {
		log.Printf("Error generating salt: %s", err)
		return
	}
	salt = base64.URLEncoding.EncodeToString(saltBytes)
	return
}

func (s *Service) insertUser(username, salt, hash, email string) (errs []error) {
	tx, err := s.db.Begin()
	if err != nil {
		log.Printf("Error starting the transaction: %s\n", err)
		errs = append(errs, err)
		return
	}

	stmt1, err := tx.Prepare("INSERT INTO users_credentials (username, salt, hash) VALUES ($1, $2, $3)")
	if err != nil {
		log.Printf("Error preparing query: %s\n", err)
		errs = append(errs, err)
		err = tx.Rollback()
		if err != nil {
			log.Printf("Error aborting the transaction: %s\n", err)
			errs = append(errs, err)
		}
		return
	}
	defer func(stmt *sql.Stmt) {
		err = stmt.Close()
		if err != nil {
			log.Printf("Error closing statement: %s\n", err)
			errs = append(errs, err)
		}
	}(stmt1)

	_, err = stmt1.Exec(username, salt, hash)
	if err != nil {
		log.Printf("Error executing query: %s\n", err)
		errs = append(errs, err)
		err = tx.Rollback()
		if err != nil {
			log.Printf("Error aborting the transaction: %s\n", err)
			errs = append(errs, err)
		}
		return
	}

	stmt2, err := tx.Prepare("INSERT INTO users_info (username, email) VALUES ($1, $2)")
	if err != nil {
		log.Printf("Error preparing query: %s\n", err)
		errs = append(errs, err)
		err = tx.Rollback()
		if err != nil {
			log.Printf("Error aborting the transaction: %s\n", err)
			errs = append(errs, err)
		}
		return
	}
	defer func(stmt *sql.Stmt) {
		err = stmt.Close()
		if err != nil {
			log.Printf("Error closing statement: %s\n", err)
			errs = append(errs, err)
		}
	}(stmt2)

	_, err = stmt2.Exec(username, email)
	if err != nil {
		log.Printf("Error executing query: %s\n", err)
		errs = append(errs, err)
		err = tx.Rollback()
		if err != nil {
			log.Printf("Error aborting the transaction: %s\n", err)
			errs = append(errs, err)
		}
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing the transaction: %s\n", err)
		errs = append(errs, err)
		return
	}

	return
}

func (s *Service) SignUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(user.Username) < minUsername || len(user.Username) > maxUsername {
		msg := fmt.Sprintf("Username must be between %d and %d characters", minUsername, maxUsername)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	password, err := generateRandomPassword()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	salt, err := generateSalt()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(password+salt), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Convert the hashed password to a string
	hashString := string(hash)

	errs := s.insertUser(user.Username, salt, hashString, user.Email)
	if len(errs) != 0 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
