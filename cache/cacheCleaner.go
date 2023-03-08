package cache

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

type cacheController struct {
	db          *bolt.DB
	dbFile      *os.File
	maxFileSize int64
	fillFactor  float64
	frequency   *time.Ticker
}

//goland:noinspection GoExportedFuncWithUnexportedType
func New(db *bolt.DB, dbFile *os.File, maxFileSize int64, fillFactor float64, frequency *time.Ticker) *cacheController {
	return &cacheController{
		db:          db,
		dbFile:      dbFile,
		maxFileSize: maxFileSize,
		fillFactor:  fillFactor,
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

	return float64(fileInfo.Size()) > float64(c.maxFileSize)*c.fillFactor
}

func (c *cacheController) deleteExpiredCache() {
	expiredKeys := make([][]byte, 0)

	addExpiredKeys := func(k, v []byte) error {
		var info Info
		err := json.Unmarshal(v, &info)

		if isExpired(&info) {
			expiredKeys = append(expiredKeys, k)
		}
		return err
	}

	// iterating over all buckets and all keys in each buckets
	// and collecting expired keys of expired data
	err := c.db.View(func(tx *bolt.Tx) error {
		err := tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			err := b.ForEach(addExpiredKeys)
			return err
		})
		return err
	})
	if err != nil {
		log.Printf("Error while viewing cache in cacheController: %v", err)
	}

	// deleting expired data
	for _, key := range expiredKeys {
		err = DeleteRecord(c.db, key)
		if err != nil {
			log.Printf("Error while deleting expired info about page in cacheController: %v", err)
		}
		err = RemovePageFromDisk(key)
		if err != nil {
			log.Printf("Error while removing expired page from disk in cacheController: %v", err)
		}
	}
}
