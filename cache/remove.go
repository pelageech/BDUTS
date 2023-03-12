package cache

import (
	"errors"
	"github.com/boltdb/bolt"
	"os"
	"strings"
)

func RemovePageFromCache(db *bolt.DB, key []byte) error {
	if err := removePageInfo(db, key); err != nil {
		return errors.New("Error while deleting record from db: " + err.Error())
	}

	if err := removePageFromDisk(key); err != nil {
		return errors.New("Error while deleting page from disk: " + err.Error())
	}

	return nil
}

// removePageInfo удаляет cache.Info запись из базы данных
func removePageInfo(db *bolt.DB, key []byte) error {
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

	path := CachePath
	for _, v := range subHashes {
		path += "/" + string(v)
	}

	err := os.Remove(path + "/" + string(key))
	if err != nil {
		return err
	}

	for path != CachePath {
		err := os.Remove(path)
		if err != nil {
			return err
		}
		path = path[:strings.LastIndexByte(path, '/')]
	}
	return nil
}
