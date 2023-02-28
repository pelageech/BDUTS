package cacheController

import (
	"log"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

type cacheController struct {
	db          *bolt.DB
	dbFile      *os.File
	maxFileSize int64
	frequency   time.Ticker
}

func New(db *bolt.DB, dbFile *os.File, maxFileSize int64, frequency time.Ticker) *cacheController {
	return &cacheController{
		db:          db,
		dbFile:      dbFile,
		maxFileSize: maxFileSize,
		frequency:   frequency,
	}
}

func (c *cacheController) Observe() {
	for {
		<-c.frequency.C
		if c.isSizeExceeded() {
			c.deleteExpiredCache()
		}
	}
}

func (c *cacheController) isSizeExceeded() bool {
	fileInfo, err := c.dbFile.Stat()
	if err != nil {
		log.Printf("Error getting file info in cacheController: %v", err) //todo: how to handle this error properly?
	}

	return fileInfo.Size() > c.maxFileSize
}

func (c *cacheController) deleteExpiredCache() {

}
