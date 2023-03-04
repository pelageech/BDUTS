package cache

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

// GetCacheIfExists Обращается к диску для нахождения ответа на запрос.
// Если таковой имеется - он возвращается, в противном случае выдаётся ошибка
func GetCacheIfExists(db *bolt.DB, req *http.Request) (*Item, error) {
	keyString := constructKeyFromRequest(req)
	requestHash := hash([]byte(keyString))

	info, err := getPageInfo(db, requestHash)
	if err != nil {
		return nil, err
	}

	if time.Now().After(info.DateOfDeath) {
		// delete
		return nil, errors.New("not fresh")
	}

	if info.IsPrivate && req.RemoteAddr != info.RemoteAddr {
		return nil, errors.New("private page: addresses are not equal")
	}

	bytes, err := readPageFromDisk(requestHash)
	if err != nil {
		return nil, err
	}

	var item Item
	err = json.Unmarshal(bytes, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

// Обращается к базе данных для получения мета-информации о кэше.
func getPageInfo(db *bolt.DB, requestHash []byte) (*Info, error) {
	var result []byte = nil

	err := db.View(func(tx *bolt.Tx) error {
		treeBucket, err := getBucket(tx, requestHash)
		if err != nil {
			return err
		}

		result = treeBucket.Get(requestHash[:])
		if result == nil {
			return errors.New("no record in cache")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var info Info
	err = json.Unmarshal(result, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// Производит чтение страницы с диска
// В случае успеха возвращает
func readPageFromDisk(requestHash []byte) ([]byte, error) {
	subhashLength := hashLength / subHashCount

	var subHashes [][]byte
	for i := 0; i < subHashCount; i++ {
		subHashes = append(subHashes, requestHash[i*subhashLength:(i+1)*subhashLength])
	}

	path := root
	for _, v := range subHashes {
		path += string(v) + "/"
	}
	path += string(requestHash[:])

	bytes, err := os.ReadFile(path)
	return bytes, err
}

// Универсальная функция получения бакета
func getBucket(tx *bolt.Tx, key []byte) (*bolt.Bucket, error) {
	bucket := tx.Bucket(key)
	if bucket != nil {
		return bucket, nil
	}

	return nil, errors.New("miss cache")
}
