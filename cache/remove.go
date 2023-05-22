package cache

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/pelageech/BDUTS/metrics"
)

// RemovePageFromCache removes the page from disk if it exists
// and its metadata from the database.
func (p *CachingProperties) RemovePageFromCache(key []byte) (*PageMetadata, error) {
	meta, err := p.removePageMetadata(key)
	if err != nil {
		return nil, errors.New("Error while deleting record from db: " + err.Error())
	}

	err = removePageFromDisk(key)
	if err != nil {
		return nil, errors.New("Error while deleting page from disk: " + err.Error())
	}

	return meta, nil
}

// removePageMetadata deletes cache.PageMetadata from the database.
func (p *CachingProperties) removePageMetadata(key []byte) (*PageMetadata, error) {
	var (
		m    []byte
		meta *PageMetadata
	)

	err := p.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(key)
		if b == nil {
			return errors.New("there's no page to delete")
		}

		m = b.Get([]byte(pageMetadataKey))

		return tx.DeleteBucket(key)
	})
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(m, &meta); err != nil {
		return nil, err
	}
	p.IncrementSize(-meta.Size)
	metrics.UpdateCachePagesCount(-1)

	return meta, nil
}

func removePageFromDisk(key []byte) error {
	subhashLength := hashLength / subHashCount

	var subHashes [][]byte
	for i := 0; i < subHashCount; i++ {
		subHashes = append(subHashes, key[i*subhashLength:(i+1)*subhashLength])
	}

	path := PagesPath
	for _, v := range subHashes {
		path += "/" + string(v)
	}

	if err := os.Remove(path + "/" + string(key)); err != nil {
		return err
	}

	for path != PagesPath {
		if err := os.Remove(path); err != nil {
			return err
		}
		path = path[:strings.LastIndexByte(path, '/')]
	}
	return nil
}
