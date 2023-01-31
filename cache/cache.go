package cache

import (
	"crypto/sha256"
	"errors"
	"github.com/boltdb/bolt"
	"log"
	"net/http"
)

const (
	hashLength   = sha256.Size
	subHashCount = 8 // Количество подотрезков хэша
)

// GetCacheIfExists Обращается к диску для нахождения ответа на запрос.
// Если таковой имеется - он возвращается, в противном случае выдаётся ошибка
func GetCacheIfExists(req *http.Request) (*http.Response, error) {
	return nil, nil
}

// Открывает базу данных для дальнейшего использования
func initDatabase(path string) *bolt.DB {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		log.Fatalln(err)
	}
	return db
}

// Закрывает базу данных
func closeDatabase(db *bolt.DB) {
	err := db.Close()
	if err != nil {
		log.Fatalln(err)
	}
}

// Добавляет новую запись в кэш.
func addNewRecord(db *bolt.DB, key, value []byte) error {
	requestHash := hash(key)
	subhashLength := hashLength / subHashCount

	var subHashes [][]byte
	for i := 0; i < subHashCount; i++ {
		subHashes = append(subHashes, requestHash[i*subhashLength:(i+1)*subhashLength])
	}

	err := db.Update(func(tx *bolt.Tx) error {
		treeBucket, err := tx.CreateBucketIfNotExists(subHashes[0])

		if err != nil {
			return err
		}
		for i := 1; i < subHashCount; i++ {
			treeBucket, err = treeBucket.CreateBucketIfNotExists(subHashes[i])
			if err != nil {
				return err
			}
		}

		err = treeBucket.Put(key, value)
		return err
	})

	return err
}

// Найти элемент по ключу
// Ключ переводится в хэш, тот разбивается на подотрезки - названия бакетов
// Проходом по подотрезкам находим по ключу ответ на запрос
func findRecord(db *bolt.DB, key []byte) []byte {
	var result []byte = nil

	requestHash := hash(key)
	subhashLength := hashLength / subHashCount

	var subHashes [][]byte
	for i := 0; i < subHashCount; i++ {
		subHashes = append(subHashes, requestHash[i*subhashLength:(i+1)*subhashLength])
	}

	err := db.View(func(tx *bolt.Tx) error {
		treeBucket := tx.Bucket(subHashes[0])
		if treeBucket == nil {
			return errors.New("miss cache")
		}
		for i := 1; i < subHashCount; i++ {
			treeBucket := treeBucket.Bucket(subHashes[i])
			if treeBucket == nil {
				return errors.New("miss cache")
			}
		}

		result = treeBucket.Get(key)
		if result == nil {
			return errors.New("no record in cache")
		}

		return nil
	})

	if err != nil {
		return nil
	}

	return result
}

// Удаляет запись из кэша
func deleteRecord(db *bolt.DB, key []byte) {

}

// Сохраняет копию базы данных в файл
func makeSnapshot(db *bolt.DB, filename string) {

}

// Возвращает хэш от набора байт
func hash(value []byte) [hashLength]byte {
	hash := sha256.Sum256(value)
	return hash
}
