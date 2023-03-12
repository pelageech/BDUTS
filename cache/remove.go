package cache

import (
	"errors"
	"github.com/boltdb/bolt"
	"os"
	"strings"
)

// deleteRecord удаляет cache.Info запись из базы данных
func deleteRecord(db *bolt.DB, requestHash []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket(requestHash)
	})
}

func removePageFromDisk(requestHash []byte) error {
	subhashLength := hashLength / subHashCount

	var subHashes [][]byte
	for i := 0; i < subHashCount; i++ {
		subHashes = append(subHashes, requestHash[i*subhashLength:(i+1)*subhashLength])
	}

	path := CachePath
	for _, v := range subHashes {
		path += "/" + string(v)
	}

	err := os.Remove(path + "/" + string(requestHash))
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

func RemovePageFromCache(db *bolt.DB, key []byte) error {
	requestHash := hash(key)
	if err := deleteRecord(db, requestHash); err != nil {
		return errors.New("Error while deleting record from db: " + err.Error())
	}

	if err := removePageFromDisk(requestHash); err != nil {
		return errors.New("Error while deleting page from disk: " + err.Error())
	}

	return nil
}
