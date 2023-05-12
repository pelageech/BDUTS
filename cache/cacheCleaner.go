package cache

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"time"

	"github.com/boltdb/bolt"
	"github.com/charmbracelet/log"
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
			func() {
				size, err := p.deleteExpiredCache()
				if err != nil {
					logger.Warnf("Expired cache: %v", err)
					return
				}
				log.Printf("Removed %d bytes of expired pages from cache\n", size)
			}()
			func() {
				size, err := p.deletePagesLRU()
				if err != nil {
					log.Warnf("LRU: %v", err)
					return
				}
				log.Printf("Removed %d bytes of the least recently used pages from cache\n", size)
			}()
		}
	}
}

func (p *CachingProperties) isSizeExceeded() bool {
	return float64(p.Size) > float64(p.cleaner.maxFileSize)*p.cleaner.fillFactor
}

// deletes cache if its max-age is expired. For page's metadata the page is
// considered expired iff time.Now > PageMetadata.ResponseDirectives.MaxAge
// returns error if boltdb view throws an error or there's no page to delete, in
// other words return size is zero.
func (p *CachingProperties) deleteExpiredCache() (int64, error) {
	type expiredItem struct {
		key  []byte
		size int64
	}

	size := int64(0)
	expiredKeys := make([]expiredItem, 0, 1024)

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
		return -1, err
	}

	// deleting expired data
	for _, item := range expiredKeys {
		meta, err := p.RemovePageFromCache(item.key)
		if err != nil {
			logger.Errorf("Failed to remove page: %v", err)
			continue
		}
		size += meta.Size
	}

	if size == 0 {
		return -1, errors.New("null size")
	}

	return size, nil
}

func (p *CachingProperties) deletePagesLRU() (int64, error) {
	type lruItem struct {
		key  []byte
		uses uint32
	}

	var lruItems []lruItem
	err := p.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			bytes := b.Get([]byte(usesKey))
			if bytes == nil {
				return errors.New("cannot access to uses")
			}

			newLruItems := lruItem{
				key:  make([]byte, len(name)),
				uses: binary.LittleEndian.Uint32(bytes),
			}

			copy(newLruItems.key, name)
			lruItems = append(lruItems, newLruItems)
			return nil
		})
	})
	if err != nil {
		return -1, err
	}

	sort.Slice(lruItems, func(i, j int) bool {
		return lruItems[i].uses < lruItems[j].uses
	})

	var size int64
	for i := 0; p.isSizeExceeded() && i < len(lruItems); i++ {
		meta, err := p.RemovePageFromCache(lruItems[i].key)
		if err != nil {
			logger.Errorf("Failed to remove page: %v", err)
			continue
		}
		size += meta.Size
	}

	if size == 0 {
		return -1, errors.New("null size")
	}

	return size, nil
}
