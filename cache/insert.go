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

var (
	infinityTime = time.Unix(0, 0).AddDate(7999, 12, 31)
)

// PutRecordInCache Помещает новую страницу в кэш или перезаписывает её.
// Сначала добавляет в базу данных метаданные о странице, хранимой в cache.Info.
// Затем начинает транзакционную запись на диск.
//
// Сохраняется json-файл, хранящий Item - тело страницы и заголовок.
func PutRecordInCache(db *bolt.DB, req *http.Request, resp *http.Response, item *Item) error {
	var byteInfo, bytePage []byte
	var err error

	if !isStorable(&item.Header) {
		return errors.New("can't be stored in cache:(")
	}

	info := createCacheInfo(req, resp, item.Header)
	if byteInfo, err = json.Marshal(*info); err != nil {
		return err
	}

	if bytePage, err = json.Marshal(*item); err != nil {
		return err
	}

	keyString := constructKeyFromRequest(req)
	requestHash := hash([]byte(keyString))

	if err = putRecord(db, requestHash, byteInfo); err != nil {
		return err
	}

	if err = writePageToDisk(requestHash, bytePage); err != nil {
		return err
	}

	log.Println("Successfully saved, page's size = ", info.Size)

	return nil
}

// putRecord Помещает в базу данных метаданные страницы, помещаемой в кэш
func putRecord(db *bolt.DB, requestHash []byte, value []byte) error {
	return db.Update(func(tx *bolt.Tx) error {
		treeBucket, err := tx.CreateBucketIfNotExists(requestHash)
		if err != nil {
			return err
		}

		err = treeBucket.Put([]byte(pageInfo), value)
		return err
	})
}

func writePageToDisk(requestHash []byte, value []byte) error {
	subhashLength := hashLength / subHashCount

	var subHashes [][]byte
	for i := 0; i < subHashCount; i++ {
		subHashes = append(subHashes, requestHash[i*subhashLength:(i+1)*subhashLength])
	}

	path := CachePath
	for _, v := range subHashes {
		path += "/" + string(v)
	}

	err := os.MkdirAll(path, 0770)
	if err != nil {
		return err
	}

	file, err := os.Create(path + "/" + string(requestHash[:]))
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

// Создаёт экземпляр структуры cache.Info, в которой хранится
// информация о странице, помещаемой в кэш.
func createCacheInfo(req *http.Request, resp *http.Response, header http.Header) *Info {
	info := &Info{
		Size:        resp.ContentLength,
		DateOfDeath: infinityTime,
		RemoteAddr:  req.RemoteAddr,
		IsPrivate:   false,
	}

	// check if we shouldn't store the page
	cacheControlString := header.Get("cache-control")
	cacheControl := strings.Split(cacheControlString, ";")

	for _, v := range cacheControl {
		if strings.Contains(v, "max-age") {
			_, t, _ := strings.Cut(v, "=")
			age, _ := strconv.Atoi(t)
			if age > 0 { // Create a point after that the page goes off
				info.DateOfDeath = time.Now().Add(time.Duration(age) * time.Second)
			}
		}

		if strings.Contains(v, "private") {
			info.IsPrivate = true
		}
	}

	return info
}

// isStorable проверяет, можно ли поместить в кэш страницу,
// по её директивам в Cache-Control.
func isStorable(header *http.Header) bool {
	cacheControlString := header.Get("cache-control")

	// check if we shouldn't store the page
	cacheControl := strings.Split(cacheControlString, ";")
	for _, v := range cacheControl {
		if v == "no-store" {
			return false
		}
	}
	return true
}
