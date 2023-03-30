package cache

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

type CacheCleaner struct {
	dbFile      *os.File
	maxFileSize int64
	fillFactor  float64
	frequency   *time.Ticker
}

func NewCacheCleaner(dbFile *os.File, maxFileSize int64, fillFactor float64, frequency *time.Ticker) *CacheCleaner {
	return &CacheCleaner{
		dbFile:      dbFile,
		maxFileSize: maxFileSize,
		fillFactor:  fillFactor,
		frequency:   frequency,
	}
}

func (p *CachingProperties) Observe() {
	for {
		<-p.cleaner.frequency.C
		if p.isSizeExceeded() {
			p.deleteExpiredCache()
		}
	}
}

func (p *CachingProperties) isSizeExceeded() bool {
	return float64(p.Size) > float64(p.cleaner.maxFileSize)*p.cleaner.fillFactor
}

func (p *CachingProperties) deleteExpiredCache() {
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
	err := p.DB().View(func(tx *bolt.Tx) error {
		return tx.ForEach(addExpiredKeys)
	})
	if err != nil {
		log.Printf("Error while viewing cache in CacheCleaner: %v", err)
	}

	if sizeReleased > 0 {
		log.Printf("Anticipating to be released %d byte from disk...", sizeReleased)
	}
	// deleting expired data
	for _, key := range expiredKeys {
		if err = p.RemovePageFromCache(key); err != nil {
			log.Println(err)
		}
	}
}
