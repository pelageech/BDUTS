package cache

// запрос HEAD не на каждое обращение не каждые несколько секунд

// transfer encoding gz

// http 1.1 ranch

import (
	"crypto/sha1"
	"encoding/hex"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pelageech/BDUTS/config"
)

const (
	hashLength   = sha1.Size * 2
	subHashCount = 4 // Количество подотрезков хэша
	root         = "./cache-data"
	pageInfo     = "pageInfo"
)

// Item структура, хранящая на диске страницу, которая
// возвращается клиенту из кэша.
type Item struct {
	Body   []byte
	Header http.Header
}

// Info - метаданные страницы, хранящейся в базе данных
type Info struct {
	Size        int64
	DateOfDeath time.Time // nil if undying
	RemoteAddr  string
	IsPrivate   bool
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

// Возвращает хэш-encode от набора байт
func hash(value []byte) []byte {
	bytes := sha1.Sum(value)
	return []byte(hex.EncodeToString(bytes[:]))
}

// constructKeyFromRequest использует массив config.RequestKey
// для того, чтобы составить строку-ключ, по которому будет сохраняться
// страница в кэше и её метаданные в БД.
func constructKeyFromRequest(req *http.Request) string {
	result := ""
	for _, addStringKey := range config.RequestKey {
		result += addStringKey(req)
	}
	return result
}
