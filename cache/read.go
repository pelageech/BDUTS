package cache

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
)

// GetPageFromCache Обращается к диску для нахождения ответа на запрос.
// Если таковой имеется - он возвращается, в противном случае выдаётся ошибка
func GetPageFromCache(db *bolt.DB, req *http.Request) (*Item, error) {
	var info *Info
	var item Item
	var err error

	requestDirectives := loadRequestDirectives(req.Header)
	// doesn't modify the request but adds a context key-value item
	req = req.WithContext(context.WithValue(req.Context(), OnlyIfCachedKey, requestDirectives.OnlyIfCached))

	keyString := constructKeyFromRequest(req)
	requestHash := hash([]byte(keyString))

	if info, err = getPageInfo(db, requestHash); err != nil {
		return nil, err
	}

	afterDeath := time.Duration(requestDirectives.MaxStale)

	maxAge := info.ResponseDirectives.MaxAge
	if !info.ResponseDirectives.SMaxAge.IsZero() {
		maxAge = info.ResponseDirectives.SMaxAge
	}

	freshTime := requestDirectives.MinFresh

	if info.ResponseDirectives.MustRevalidate {
		afterDeath = time.Duration(0)
	}
	if time.Now().After(maxAge.Add(afterDeath)) || !time.Now().After(freshTime) {
		return nil, errors.New("not fresh")
	}

	var byteItem []byte
	if byteItem, err = readPageFromDisk(requestHash); err != nil {
		return nil, err
	}

	if err = json.Unmarshal(byteItem, &item); err != nil {
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

		result = treeBucket.Get([]byte(pageInfo))
		if result == nil {
			return errors.New("no record in cache")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	var info Info
	if err = json.Unmarshal(result, &info); err != nil {
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

	path := PagesPath
	for _, v := range subHashes {
		path += "/" + string(v)
	}
	path += "/" + string(requestHash[:])

	bytes, err := os.ReadFile(path)
	return bytes, err
}

// Универсальная функция получения бакета
func getBucket(tx *bolt.Tx, key []byte) (*bolt.Bucket, error) {
	if bucket := tx.Bucket(key); bucket != nil {
		return bucket, nil
	}

	return nil, errors.New("miss cache")
}
