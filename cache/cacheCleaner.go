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
func NewCacheController(db *bolt.DB, dbFile *os.File, maxFileSize int64, fillFactor float64, frequency *time.Ticker) *cacheController {
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
	sizeReleased := int64(0)
	expiredKeys := make([][]byte, 0)

	addExpiredKeys := func(name []byte, b *bolt.Bucket) error {
		v := b.Get([]byte(pageInfo))

		var info PageMetadata
		if err := json.Unmarshal(v, &info); err != nil {
			return err
		}

		if isExpired(&info, time.Duration(0)) {
			expiredKeys = append(expiredKeys, name)
			sizeReleased += info.Size
		}
		return nil
	}

	// iterating over all buckets and all keys in each bucket
	// and collecting expired keys of expired data
	err := c.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(addExpiredKeys)
	})
	if err != nil {
		log.Printf("Error while viewing cache in cacheController: %v", err)
	}

	if sizeReleased > 0 {
		log.Printf("Anticipating to be released %d byte from disk...", sizeReleased)
	}
	// deleting expired data
	for _, key := range expiredKeys {
		if err = RemovePageFromCache(c.db, key); err != nil {
			log.Println(err)
		}
	}
}
