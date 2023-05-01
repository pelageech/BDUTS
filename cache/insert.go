package cache

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/pelageech/BDUTS/metrics"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

var (
	infinityTime = time.Unix(0, 0).AddDate(7999, 12, 31)
	nullTime     = time.Time{}
)

// InsertPageInCache stores a new page in cache or rewrites the current page.
// First, it adds PageMetadata in DB and then the function starts a process of
// transactional writing the page on a disk.
// Page transforms to json-file.
func (p *CachingProperties) InsertPageInCache(key []byte, req *http.Request, resp *http.Response, page *Page) error {
	var err error

	size := int64(len(page.Body))
	if p.Size+size > p.Cleaner().maxFileSize {
		return errors.New("maximum size cache exceeded")
	}

	requestDirectives := loadRequestDirectives(req.Header)
	responseDirectives := loadResponseDirectives(resp.Header)

	if requestDirectives.NoStore || responseDirectives.NoStore {
		return errors.New("can't be stored in cache")
	}

	meta := createCacheInfo(resp, int64(len(page.Body)))

	if err = p.insertPageMetadataToDB(key, meta); err != nil {
		return err
	}

	if err = writePageToDisk(key, page); err != nil {
		_, _ = p.removePageMetadata(key)
		return err
	}

	log.Println("Successfully saved, page's size = ", meta.Size)
	return nil
}

func (p *CachingProperties) insertPageMetadataToDB(key []byte, meta *PageMetadata) error {
	value, err := json.Marshal(*meta)
	if err != nil {
		return err
	}

	err = p.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket(key)
		if err == bolt.ErrBucketExists {
			b = tx.Bucket(key)
		}
		if err == nil || err == bolt.ErrBucketExists {
			_ = b.Put([]byte(pageMetadataKey), value)
			bs := make([]byte, 4)
			_ = b.Put([]byte(usesKey), bs)
		}

		return err
	})

	if err == nil {
		p.IncrementSize(meta.Size)
		metrics.UpdateCachePagesCount(1)
	}
	if err == bolt.ErrBucketExists {
		return nil
	}
	return err
}

func writePageToDisk(key []byte, page *Page) error {
	value, err := json.Marshal(*page)
	if err != nil {
		return err
	}

	path := makePath(key, subHashCount)
	if err := os.MkdirAll(path, 0770); err != nil {
		return err
	}

	file, err := os.Create(path + "/" + string(key[:]))
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriterSize(file, bufferSize)
	_, err = w.Write(value)
	if err != nil {
		return err
	}
	return w.Flush()
}

// Создаёт экземпляр структуры cache.PageMetadata, в которой хранится
// информация о странице, помещаемой в кэш.
func createCacheInfo(resp *http.Response, size int64) *PageMetadata {
	meta := &PageMetadata{
		Size:               size,
		ResponseDirectives: *loadResponseDirectives(resp.Header),
	}

	return meta
}
