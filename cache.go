package main

import (
	"github.com/boltdb/bolt"
	"log"
	"net/http"
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
func addNewRecord(key, value []byte) {

}

// Удаляет запись из кэша
func deleteRecord(key []byte) {

}

// Сохраняет копию базы данных в файл
func makeSnapshot(filename string) {

}

// Возвращает хэш от набора байт
func hash(value []byte) []byte {
	return nil
}
