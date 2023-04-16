package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const (
	passwordLength = 25
	saltLength     = 20
	minUsername    = 4
	maxUsername    = 20
)

var db *sql.DB

func init() {
	postgresUser := os.Getenv("POSTGRES_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dataSource := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", postgresUser, dbPassword, dbHost, dbPort, dbName)

	var err error
	db, err = sql.Open("postgres", dataSource)
	if err != nil {
		log.Fatalf("failed to open database: %v\n", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("failed to ping database: %v\n", err)
	}

	log.Println("Database connection established")
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

func SignUp(w http.ResponseWriter, r *http.Request) {
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

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error starting the transaction: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stmt1, err := tx.Prepare("INSERT INTO users_credentials (username, salt, hash) VALUES ($1, $2, $3)")
	if err != nil {
		log.Printf("Error preparing query: %s\n", err)
		err := tx.Rollback()
		if err != nil {
			log.Printf("Error aborting the transaction: %s\n", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			log.Printf("Error closing statement: %s\n", err)
		}
	}(stmt1)

	_, err = stmt1.Exec(user.Username, salt, hashString)
	if err != nil {
		log.Printf("Error executing query: %s\n", err)
		err := tx.Rollback()
		if err != nil {
			log.Printf("Error aborting the transaction: %s\n", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	stmt2, err := tx.Prepare("INSERT INTO users_info (username, email) VALUES ($1, $2)")
	if err != nil {
		log.Printf("Error preparing query: %s\n", err)
		err := tx.Rollback()
		if err != nil {
			log.Printf("Error aborting the transaction: %s\n", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer func(stmt *sql.Stmt) {
		err := stmt.Close()
		if err != nil {
			log.Printf("Error closing statement: %s\n", err)
		}
	}(stmt2)

	_, err = stmt2.Exec(user.Username, user.Email)
	if err != nil {
		log.Printf("Error executing query: %s\n", err)
		err := tx.Rollback()
		if err != nil {
			log.Printf("Error aborting the transaction: %s\n", err)
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing the transaction: %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if !IsAuthorized(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func IsAuthenticated(r *http.Request) bool {
	bearerToken := r.Header.Get("Authorization")
	if bearerToken == "" {
		return false
	}

	return true
}

func IsAuthorized(r *http.Request) bool {
	return false
}
