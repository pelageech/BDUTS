package cache

// запрос HEAD не на каждое обращение не каждые несколько секунд

// transfer encoding gz

// http 1.1 ranch

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pelageech/BDUTS/config"
)

type Key int

const (
	// OnlyIfCachedKey is used for saving to request context the directive
	// 'only-if-cached' from Cache-Control.
	OnlyIfCachedKey = Key(iota)

	Hash

	// OnlyIfCachedError is used for sending to the client an error about
	// missing cache while 'only-if-cached' is specified in Cache-Control.
	OnlyIfCachedError = "HTTP 504 Unsatisfiable Request (only-if-cached)"
)

const (
	// DbDirectory is the directory of storing the BoltDB database.
	DbDirectory = "./cache-data"

	// DbName is a name of the database.
	DbName = "database.db"

	// DefaultKey is used if there's no key parameter of cache for url
	DefaultKey = "REQ_METHOD;REQ_HOST;REQ_URI"

	// PagesPath is the directory where the pages are written to.
	PagesPath = "./cache-data/db"

	hashLength      = sha1.Size * 2
	subHashCount    = 4 // Количество подотрезков хэша
	pageMetadataKey = "pageMetadataKey"
)

type UrlToKeyBuilder map[string][]func(r *http.Request) string

type CachingProperties struct {
	db            *bolt.DB
	keyBuilderMap UrlToKeyBuilder
	cleaner       *CacheCleaner
	Size          int64
}

func NewCachingProperties(DB *bolt.DB, cacheConfig *config.CacheConfig, cleaner *CacheCleaner) *CachingProperties {
	keyBuilder := make(UrlToKeyBuilder)

	for _, v := range cacheConfig.Pairs() {
		keyBuilder[v.Location] = config.ParseRequestKey(v.RequestKey)
	}

	return &CachingProperties{
		db:            DB,
		keyBuilderMap: keyBuilder,
		cleaner:       cleaner,
		Size:          0,
	}
}

func (p *CachingProperties) DB() *bolt.DB {
	return p.db
}

func (p *CachingProperties) KeyBuilderMap() UrlToKeyBuilder {
	return p.keyBuilderMap
}

func (p *CachingProperties) Cleaner() *CacheCleaner {
	return p.cleaner
}

func (p *CachingProperties) IncrementSize(delta int64) {
	atomic.AddInt64(&p.Size, delta)
}

func (p *CachingProperties) CalculateSize() {
	size := int64(0)
	err := p.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			metaBytes := b.Get([]byte(pageMetadataKey))
			if metaBytes == nil {
				return errors.New("all the buckets must have pageMetadataKey-value, you should clear the database and cache")
			}

			var m PageMetadata
			if err := json.Unmarshal(metaBytes, &m); err != nil {
				return errors.New("Non-persistent json-data, clear the cache! " + err.Error())
			}
			size += m.Size
			return nil
		})
	})
	if err != nil {
		log.Fatalln(err)
	}
	p.Size = size
}

// Page is a structure that is the cache unit storing on a disk.
type Page struct {
	// Body is the body of the response saving to the cache.
	Body []byte

	// Header is the response header saving to the cache.
	Header http.Header
}

// PageMetadata is a struct of page metadata
type PageMetadata struct {
	// Size is the response body size.
	Size int64

	ResponseDirectives responseDirectives

	// Uses is the count of referring to the page
	Uses int
}

//	MaxAge:       +
//	MaxStale:     +
//	MinFresh:     +
//	NoCache:
//	NoStore:	  +
//	NoTransform:
//	OnlyIfCached: +

type requestDirectives struct {
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

type responseDirectives struct {
	MustRevalidate  bool
	NoCache         bool
	NoStore         bool
	NoTransform     bool
	Private         bool
	ProxyRevalidate bool
	MaxAge          time.Time
	SMaxAge         time.Time
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
/*func MakeSnapshot(db *bolt.DB, filename string) error {
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

func (p *CachingProperties) RequestHashKey(req *http.Request) []byte {
	return hash([]byte(
		p.constructKeyFromRequest(req),
	))
}

// Returns a hash-encode byte array of a value
func hash(value []byte) []byte {
	bytes := sha1.Sum(value)
	return []byte(hex.EncodeToString(bytes[:]))
}

// constructKeyFromRequest uses an array config.RequestKey
// in order to construct a key for mapping this one with
// values of page on a disk and its metadata in DB.
func (p *CachingProperties) constructKeyFromRequest(req *http.Request) string {
	result := ""

	keyBuilderMap := p.KeyBuilderMap()
	keyBuilder, ok := keyBuilderMap[req.URL.Path]
	if !ok {
		keyBuilder = config.ParseRequestKey(DefaultKey)
	}

	for _, addStringKey := range keyBuilder {
		result += addStringKey(req)
	}

	return result
}

func isExpired(info *PageMetadata, afterDeath time.Duration) bool {
	return time.Now().After(info.ResponseDirectives.MaxAge.Add(afterDeath))
}

func loadRequestDirectives(header http.Header) *requestDirectives {
	result := &requestDirectives{
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
			if age == 0 {
				result.MaxAge = infinityTime
			} else {
				result.MaxAge = time.Now().Add(time.Duration(age) * time.Second)
			}
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

func loadResponseDirectives(header http.Header) *responseDirectives {
	result := &responseDirectives{
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
			if age == 0 {
				result.MaxAge = infinityTime
			} else {
				result.MaxAge = time.Now().Add(time.Duration(age) * time.Second)
			}
		} else if strings.Contains(v, "s-maxage") {
			_, t, _ := strings.Cut(v, "=")
			age, _ := strconv.Atoi(t)
			if age == 0 {
				result.SMaxAge = nullTime
			} else {
				result.SMaxAge = time.Now().Add(time.Duration(age) * time.Second)
			}
		}
	}

	return result
}
