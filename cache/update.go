package cache

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
)

// PutRecordInCache Помещает новую запись в кэш.
// Считает хэш аттрибутов запроса, по нему проходит вниз по дереву
// и записывает как лист новую запись.
func PutRecordInCache(db *bolt.DB, req *http.Request, item *Item) error {
	if !isStorable(req) {
		return errors.New("can't be stored in cache:(")
	}

	info := createCacheInfo(req, item.Header)

	valueInfo, err := json.Marshal(*info)
	if err != nil {
		return err
	}

	page, err := json.Marshal(*item)
	if err != nil {
		return err
	}

	keyString := constructKeyFromRequest(req)

	requestHash := hash([]byte(keyString))
	err = putPageInfoIntoDB(db, requestHash, valueInfo)
	if err != nil {
		return err
	}

	err = writePageToDisk(requestHash, page)
	return err
}

// Добавляет новую запись в кэш.
func putPageInfoIntoDB(db *bolt.DB, requestHash []byte, value []byte) error {
	err := db.Update(func(tx *bolt.Tx) error {
		treeBucket, err := tx.CreateBucketIfNotExists(requestHash)
		if err != nil {
			return err
		}

		err = treeBucket.Put(requestHash[:], value)
		return err
	})

	return err
}

func writePageToDisk(requestHash []byte, value []byte) error {
	subhashLength := hashLength / subHashCount

	var subHashes [][]byte
	for i := 0; i < subHashCount; i++ {
		subHashes = append(subHashes, requestHash[i*subhashLength:(i+1)*subhashLength])
	}

	path := root
	for _, v := range subHashes {
		path += string(v) + "/"
	}

	err := os.MkdirAll(path, 0770)
	if err != nil {
		return err
	}

	file, err := os.Create(path + string(requestHash[:]))
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Println("Write to disk error: ", err)
		}
	}(file)

	_, err = file.Write(value)
	return err
}

// Удаляет запись из кэша
//func deleteRecord(db *bolt.DB, key []byte) error {
//	requestHash := hash(key)
//	subhashLength := hashLength / subHashCount
//
//	var subHashes [][]byte
//	for i := 0; i < subHashCount; i++ {
//		subHashes = append(subHashes, requestHash[i*subhashLength:(i+1)*subhashLength])
//	}
//
//	err := db.Update(func(tx *bolt.Tx) error {
//		treeBucket := tx.Bucket(subHashes[0])
//		if treeBucket == nil {
//			return errors.New("miss cache")
//		}
//		for i := 1; i < subHashCount; i++ {
//			treeBucket := treeBucket.Bucket(subHashes[i])
//			if treeBucket == nil {
//				return errors.New("miss cache")
//			}
//		}
//
//		err := treeBucket.Delete(key)
//
//		return err
//	})
//
//	return err
//}

func createCacheInfo(req *http.Request, header http.Header) *Info {
	var info Info

	info.RemoteAddr = req.RemoteAddr
	info.IsPrivate = false

	cacheControlString := header.Get("cache-control")

	// check if we shouldn't store the page
	cacheControl := strings.Split(cacheControlString, ";")
	for _, v := range cacheControl {
		if strings.Contains(v, "max-age") {
			_, t, _ := strings.Cut(v, "=")
			age, _ := strconv.Atoi(t)
			if age > 0 {
				info.DateOfDeath = time.Now().Add(time.Duration(age) * time.Second)
			}
		}
		if strings.Contains(v, "private") {
			info.IsPrivate = true
		}
	}

	return &info
}
