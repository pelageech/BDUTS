package cache

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
	"sort"
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
			if size := p.deleteExpiredCache(); size > 0 {
				log.Printf("Removed %d bytes of expired pages from cache\n", size)
			}
			if size := p.deletePagesLRU(); size > 0 {
				log.Printf("Removed %d bytes of the least recently used pages from cache\n", size)
			}
		}
	}
}

func (p *CachingProperties) isSizeExceeded() bool {
	return float64(p.Size) > float64(p.cleaner.maxFileSize)*p.cleaner.fillFactor
}

func (p *CachingProperties) deleteExpiredCache() int64 {
	type expiredItem struct {
		key  []byte
		size int64
	}

	size := int64(0)
	expiredKeys := make([]expiredItem, 0)

	addExpiredKeys := func(name []byte, b *bolt.Bucket) error {
		v := b.Get([]byte(pageMetadataKey))

		var info PageMetadata
		if err := json.Unmarshal(v, &info); err != nil {
			return err
		}

		if isExpired(&info, time.Duration(0)) {
			expiredKeys = append(expiredKeys, expiredItem{name, info.Size})
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

	// deleting expired data
	for _, item := range expiredKeys {
		meta, err := p.RemovePageFromCache(item.key)
		if err != nil {
			log.Println(err)
		}
		size += meta.Size
	}

	return size
}

func (p *CachingProperties) deletePagesLRU() int64 {
	type lruItem struct {
		key  []byte
		uses uint32
	}

	var lruItems []lruItem
	_ = p.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			bytes := b.Get([]byte(usesKey))

			newLruItems := lruItem{
				key:  make([]byte, len(name)),
				uses: binary.LittleEndian.Uint32(bytes),
			}

			copy(newLruItems.key, name)
			lruItems = append(lruItems, newLruItems)
			return nil
		})
	})

	sort.Slice(lruItems, func(i, j int) bool {
		return lruItems[i].uses < lruItems[j].uses
	})

	var size int64
	for i := 0; p.isSizeExceeded() && i < len(lruItems); i++ {
		meta, err := p.RemovePageFromCache(lruItems[i].key)
		if err != nil {
			log.Println(err)
		}
		size += meta.Size
	}

	return size
}
