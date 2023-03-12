package cache

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

// GetPageFromCache Обращается к диску для нахождения ответа на запрос.
// Если таковой имеется - он возвращается, в противном случае выдаётся ошибка
func GetPageFromCache(db *bolt.DB, req *http.Request) (*Item, error) {
	var info *Info
	var item Item
	var err error

	keyString := constructKeyFromRequest(req)
	requestHash := hash([]byte(keyString))

	if info, err = getPageInfo(db, requestHash); err != nil {
		return nil, err
	}

	afterDeath := time.Duration(0)
	if !info.MustRevalidate {
		afterDeath = getAfterDeath(req)
	}
	if time.Now().After(info.DateOfDeath.Add(afterDeath)) {
		return nil, errors.New("not fresh")
	}

	//if info.IsPrivate && req.RemoteAddr != info.RemoteAddr {
	//	return nil, errors.New("private page: addresses are not equal")
	//}

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

	path := CachePath
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

func getAfterDeath(req *http.Request) time.Duration {
	cacheControlString := req.Header.Get("cache-control")

	cacheControl := strings.Split(cacheControlString, ";")
	for _, v := range cacheControl {
		if strings.Contains(v, "max-stale") {
			_, t, _ := strings.Cut(v, "=")
			age, _ := strconv.Atoi(t)
			return time.Duration(age)
		}
	}

	return time.Duration(0)
}
