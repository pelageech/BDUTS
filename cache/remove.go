package cache

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/boltdb/bolt"
)

// RemovePageFromCache removes the page from disk if it exists
// and its metadata from the database
func (p *CachingProperties) RemovePageFromCache(key []byte) error {
	keyCopy := make([]byte, len(key), len(key)) // todo: ПОЧЕМУ ГРЕБАНЫЙ КЛЮЧ key МЕНЯЕТСЯ????!!! разобраться
	copy(keyCopy, key)

	_, err := p.removePageMetadata(keyCopy)
	if err != nil {
		return errors.New("Error while deleting record from db: " + err.Error())
	}
	if err := removePageFromDisk(keyCopy); err != nil {
		return errors.New("Error while deleting page from disk: " + err.Error())
	}

	return nil
}

// removePageMetadata удаляет cache.PageMetadata запись из базы данных
func (p *CachingProperties) removePageMetadata(key []byte) (*PageMetadata, error) {
	var m []byte
	var meta *PageMetadata
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
