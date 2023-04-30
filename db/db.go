package db

import (
	"database/sql"
	"fmt"
	"log"
)

type Service struct {
	db *sql.DB
}

func (s *Service) Connect(user, password, host, port, dbName string) (err error) {
	dataSource := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", user, password, host, port, dbName)

	s.db, err = sql.Open("postgres", dataSource)
	if err != nil {
		log.Printf("failed to open database: %v\n", err)
		return
	}

	err = s.db.Ping()
	if err != nil {
		log.Printf("failed to ping database: %v\n", err)
		return
	}

	log.Println("Database connection established")
	return
}

func (s *Service) Close() (err error) {
	err = s.db.Close()
	if err != nil {
		log.Printf("failed to close database: %v\n", err)
		return
	}

	log.Println("Database connection closed")
	return
}

func (s *Service) InsertUser(username, salt, hash, email string) (errs []error) {
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

func (s *Service) GetSaltAndHash(username string) (salt, hash string, err error) {
	err = s.db.QueryRow("SELECT salt, hash FROM users_credentials WHERE username = $1", username).Scan(&salt, &hash)
	if err != nil {
		log.Printf("Error getting salt and hash: %s\n", err)
		return
	}

	return
}

func (s *Service) ChangePassword(username, salt, hash string) (err error) {
	_, err = s.db.Exec("UPDATE users_credentials SET salt = $1, hash = $2 WHERE username = $3", salt, hash, username)
	return
}

func (s *Service) GetEmail(username string) (email string, err error) {
	err = s.db.QueryRow("SELECT email FROM users_info WHERE username = $1", username).Scan(&email)
	if err != nil {
		log.Printf("Error getting email: %s\n", err)
		return
	}
	return
}
