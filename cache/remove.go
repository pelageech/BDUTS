package cache

import (
	"errors"
	"github.com/boltdb/bolt"
	"os"
	"strings"
)

// RemovePageFromCache removes the page from disk if it exists
// and its metadata from the database
func (props *CachingProperties) RemovePageFromCache(key []byte) error {
	if err := removePageMetadata(props.DB(), key); err != nil {
		return errors.New("Error while deleting record from db: " + err.Error())
	}

	if err := removePageFromDisk(key); err != nil {
		return errors.New("Error while deleting page from disk: " + err.Error())
	}

	return nil
}

// removePageMetadata удаляет cache.PageMetadata запись из базы данных
func removePageMetadata(db *bolt.DB, key []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(key)
	})
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
