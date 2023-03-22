package cache

// запрос HEAD не на каждое обращение не каждые несколько секунд

// transfer encoding gz

// http 1.1 ranch

import (
	"crypto/sha1"
	"encoding/hex"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pelageech/BDUTS/config"
)

type Key int

const (
	// OnlyIfCachedKey is used for saving to request context the directive
	// 'only-if-cached' from Cache-Control.
	OnlyIfCachedKey = Key(iota)

	// OnlyIfCachedError is used for sending to the client an error about
	// missing cache while 'only-if-cached' is specified in Cache-Control.
	OnlyIfCachedError = "HTTP 504 Unsatisfiable Request (only-if-cached)"
)

const (
	// DbDirectory is the directory of storing the BoltDB database.
	DbDirectory = "./cache-data"

	// DbName is a name of the database.
	DbName = "database.db"

	// PagesPath is the directory where the pages are written to.
	PagesPath = "./cache-data/db"

	hashLength   = sha1.Size * 2
	subHashCount = 4 // Количество подотрезков хэша
	pageInfo     = "pageInfo"
)

// Item структура, хранящая на диске страницу, которая
// возвращается клиенту из кэша.
type Item struct {
	// Body is the body of the response saving to the cache.
	Body []byte

	// Header is the response header saving to the cache.
	Header http.Header
}

//	MaxAge:       +
//	MaxStale:     +
//	MinFresh:     +
//	NoCache:
//	NoStore:	  +
//	NoTransform:
//	OnlyIfCached: +

// RequestDirectives
type RequestDirectives struct {
	MaxAge       time.Time
	MaxStale     int64
	MinFresh     time.Time
	NoCache      bool
	NoStore      bool
	NoTransform  bool
	OnlyIfCached bool
}

//	MustRevalidate:  +
//	NoCache:
//	NoStore:	     +
//	NoTransform:
//	Private:
//	ProxyRevalidate:
//	MaxAge:          +
//	SMaxAge:         +

type ResponseDirectives struct {
	MustRevalidate  bool
	NoCache         bool
	NoStore         bool
	NoTransform     bool
	Private         bool
	ProxyRevalidate bool
	MaxAge          time.Time
	SMaxAge         time.Time
}

// Info is a struct of page metadata
type Info struct {
	// Size is the response body size.
	Size int64

	ResponseDirectives ResponseDirectives
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

func isExpired(info *Info, afterDeath time.Duration) bool {
	return time.Now().After(info.ResponseDirectives.MaxAge.Add(afterDeath))
}

func loadRequestDirectives(header http.Header) *RequestDirectives {
	result := &RequestDirectives{
		MaxAge:       nullTime,
		MaxStale:     0,
		MinFresh:     nullTime,
		NoCache:      false,
		NoStore:      false,
		NoTransform:  false,
		OnlyIfCached: false,
	}

	cacheControlString := header.Get("cache-control")
	cacheControl := strings.Split(cacheControlString, ";")
	for _, v := range cacheControl {
		if v == "only-if-cached" {
			result.OnlyIfCached = true
		} else if v == "no-cache" {
			result.NoCache = true
		} else if v == "no-store" {
			result.NoStore = true
		} else if v == "no-transform" {
			result.NoTransform = true
		} else if strings.Contains(v, "max-age") {
			_, t, _ := strings.Cut(v, "=")
			age, _ := strconv.Atoi(t)
			result.MaxAge = time.Now().Add(time.Duration(age) * time.Second)
		} else if strings.Contains(v, "max-stale") {
			_, t, _ := strings.Cut(v, "=")
			age, _ := strconv.Atoi(t)
			result.MaxStale = int64(age)
		} else if strings.Contains(v, "min-fresh") {
			_, t, _ := strings.Cut(v, "=")
			age, _ := strconv.Atoi(t)
			result.MinFresh = time.Now().Add(time.Duration(age) * time.Second)
		}
	}

	return result
}

func loadResponseDirectives(header http.Header) *ResponseDirectives {
	result := &ResponseDirectives{
		MustRevalidate:  false,
		NoCache:         false,
		NoStore:         false,
		NoTransform:     false,
		Private:         false,
		ProxyRevalidate: false,
		MaxAge:          infinityTime,
		SMaxAge:         nullTime,
	}

	cacheControlString := header.Get("cache-control")
	cacheControl := strings.Split(cacheControlString, ";")
	for _, v := range cacheControl {
		if v == "must-revalidate" {
			result.MustRevalidate = true
		} else if v == "no-cache" {
			result.NoCache = true
		} else if v == "no-store" {
			result.NoStore = true
		} else if v == "no-transform" {
			result.NoTransform = true
		} else if v == "private" {
			result.Private = true
		} else if v == "proxy-revalidate" {
			result.ProxyRevalidate = true
		} else if strings.Contains(v, "max-age") {
			_, t, _ := strings.Cut(v, "=")
			age, _ := strconv.Atoi(t)
			result.MaxAge = time.Now().Add(time.Duration(age) * time.Second)
		} else if strings.Contains(v, "s-maxage") {
			_, t, _ := strings.Cut(v, "=")
			age, _ := strconv.Atoi(t)
			result.SMaxAge = time.Now().Add(time.Duration(age) * time.Second)
		}
	}

	return result
}
