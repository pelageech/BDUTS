package cache

// запрос HEAD не на каждое обращение не каждые несколько секунд

// transfer encoding gz

// http 1.1 ranch

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pelageech/BDUTS/config"
)

const (
	/*reading = iota
	writing
	silent*/
	hashLength   = sha1.Size * 2
	subHashCount = 4 // Количество подотрезков хэша
	root         = "./cache-data/"
)

type Item struct {
	Body   []byte
	Header http.Header
}

type Info struct {
	DateOfDeath time.Time // nil if undying
	RemoteAddr  string
	IsPrivate   bool
	//	status      int
}

// OpenDatabase Открывает базу данных для дальнейшего использования
func OpenDatabase(path string) (*bolt.DB, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// CloseDatabase Закрывает базу данных
func CloseDatabase(db *bolt.DB) {
	err := db.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

// Сохраняет копию базы данных в файл
/*func makeSnapshot(db *bolt.DB, filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}

	err = db.View(func(tx *bolt.Tx) error {
		_, err := tx.WriteTo(f)
		return err
	})

	return err
}*/

// Возвращает хэш от набора байт
func hash(value []byte) []byte {
	bytes := sha1.Sum(value)
	return []byte(hex.EncodeToString(bytes[:]))
}

func constructKeyFromRequest(req *http.Request) string {
	result := ""
	for _, addStringKey := range config.RequestKey {
		result += addStringKey(req)
	}
	return result
}

func isStorable(req *http.Request) bool {
	header := req.Header
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

func getBucket(tx *bolt.Tx, key []byte) (*bolt.Bucket, error) {
	bucket := tx.Bucket(key)
	if bucket != nil {
		return bucket, nil
	}

	return nil, errors.New("miss cache")
}

/*func setStatusReading(db *bolt.DB, requestHash []byte) error {

	err := db.Update(func(tx *bolt.Tx) error {
		treeBucket, err := getBucket(tx, requestHash)
		if err != nil {
			return err
		}

		status := treeBucket.Get([]byte("status"))
		if status[0] == writing {
			return errors.New("writing")
		}
		err = treeBucket.Put([]byte("status"), []byte{reading})

		return err
	})

	return err
}

func setStatusWriting(db *bolt.DB, requestHash []byte) error {

	err := db.Update(func(tx *bolt.Tx) error {
		treeBucket, err := getBucket(tx, requestHash)
		if err != nil {
			return err
		}

		err = treeBucket.Put([]byte("status"), []byte{writing})

		return err
	})

	return err
}

func setStatusSilent(db *bolt.DB, requestHash []byte) error {

	err := db.Update(func(tx *bolt.Tx) error {
		treeBucket, err := getBucket(tx, requestHash)
		if err != nil {
			return err
		}

		err = treeBucket.Put([]byte("status"), []byte{silent})

		return err
	})

	return err
}*/
