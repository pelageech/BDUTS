package db

import (
	"database/sql"
	"fmt"
	"log"
)

func Connect(user, password, host, port, dbName string) (db *sql.DB, err error) {
	dataSource := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", user, password, host, port, dbName)

	db, err = sql.Open("postgres", dataSource)
	if err != nil {
		log.Printf("failed to open database: %v\n", err)
		return
	}

	err = db.Ping()
	if err != nil {
		log.Printf("failed to ping database: %v\n", err)
		return
	}

	log.Println("Database connection established")
	return
}
