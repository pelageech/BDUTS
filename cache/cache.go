package cache

// запрос HEAD не на каждое обращение не каждые несколько секунд

// transfer encoding gz

// http 1.1 ranch

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/boltdb/bolt"
	"github.com/charmbracelet/log"
	"github.com/pelageech/BDUTS/metrics"
)

type Key int

const (
	// OnlyIfCachedKey is used for saving to request context the directive
	// 'only-if-cached' from Cache-Control.
	OnlyIfCachedKey = Key(iota)

	Hash
)

const (
	// DbDirectory is the directory of storing the BoltDB database.
	DbDirectory = "./cache-data"

	// DbName is a name of the database.
	DbName = "database.db"

	// DefaultKey is used if there's no key parameter of cache for url. Not used now.
	DefaultKey = "REQ_METHOD;REQ_HOST;REQ_URI"

	// PagesPath is the directory where the pages are written to.
	PagesPath = "./cache-data/db"

	hashLength      = sha1.Size * 2
	subHashCount    = 4 // Количество подотрезков хэша
	pageMetadataKey = "pageMetadataKey"
	usesKey         = "usesKey"
)

const (
	bufferSize          = 128 << 10
	infinityTimeYear    = 7999
	infinityTimeMonth   = 12
	infinityTimeDay     = 31
	initMemorySliceSize = 1024
	sizeOfInt32         = 4
)

const (
	readWriteOwner             = 0o600
	readWriteExecuteOwnerGroup = 0o770
)

var (
	// ErrOnlyIfCached is used for sending to the client an error about
	// missing cache while 'only-if-cached' is specified in Cache-Control.
	ErrOnlyIfCached = errors.New("HTTP 504 Unsatisfiable Request (only-if-cached)")

	logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		ReportCaller:    true,
	})
)

// UrlToKeyBuilder is a map that contains parsers for each string.
// The parser takes a part of URI by string from the request. Not used.
type UrlToKeyBuilder map[string]func(r *http.Request) string

// CachingProperties takes over the cache and the pages there.
// The structure of the cache reminds file systems in common operating systems.
// The pages are stored on a disk and the driver is stored in memory.
// The driver is a boltDB database containing buckets named by request hash.
// Each of buckets has metadata struct and an amount of usage the page during its life.
type CachingProperties struct {
	db *bolt.DB
	//	keyBuilderMap UrlToKeyBuilder
	cleaner    *CacheCleaner
	Size       int64
	PagesCount int
}

// Page is a structure that is the cache unit storing on a disk.
type Page struct {
	// Body is the body of the response saving to the cache.
	Body []byte

	// Header is the response header saving to the cache.
	Header http.Header
}

// PageMetadata contains a metadata about the page
// stored in a cache.
type PageMetadata struct {
	// Size is the response body size.
	Size int64

	ResponseDirectives responseDirectives
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

func NewCachingProperties(db *bolt.DB, cleaner *CacheCleaner) *CachingProperties {
	return &CachingProperties{
		db:         db,
		cleaner:    cleaner,
		Size:       0,
		PagesCount: 0,
	}
}

func (p *CachingProperties) DB() *bolt.DB {
	return p.db
}

func (p *CachingProperties) Cleaner() *CacheCleaner {
	return p.cleaner
}

func (p *CachingProperties) IncrementSize(delta int64) {
	atomic.AddInt64(&p.Size, delta)
	metrics.UpdateCacheSize(p.Size)
}

// CalculateSize counts how many pages are stored in a cache and puts
// a total size of cache to CachingProperties.
// The function runs through the boltDB and checks if the cache is stored
// on a disk. If success, it adds its size to the counter.
func (p *CachingProperties) CalculateSize() {
	size := int64(0)
	pagesCount := 0

	checkDisk := func(hash []byte) error {
		path := makePath(hash, subHashCount)
		path += "/" + string(hash)

		_, err := os.Stat(path)
		return err
	}

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

			if err := checkDisk(name); err != nil {
				logger.Info("Checking page on the disk: ", "err", err)
				return nil
			}

			size += m.Size
			pagesCount++
			return nil
		})
	})
	if err != nil {
		panic(err)
	}
	p.Size = size
	p.PagesCount = pagesCount
}

// OpenDatabase opens a database file.
func OpenDatabase(path string) (*bolt.DB, error) {
	db, err := bolt.Open(path, readWriteOwner, nil)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// CloseDatabase closes a database file.
func CloseDatabase(db *bolt.DB) {
	err := db.Close()
	if err != nil {
		logger.Fatal("Closing Bolt database: ", "err", err)
	}
}

// RequestHashKey returns hash of a request.
func (p *CachingProperties) RequestHashKey(req *http.Request) []byte {
	return hash([]byte(
		p.constructKeyFromRequest(req),
	))
}

func LoggerConfig(prefix string) {
	logger.SetPrefix(prefix)
}

// Returns a hash-encode byte array of a value.
func hash(value []byte) []byte {
	bytes := sha1.Sum(value)
	return []byte(hex.EncodeToString(bytes[:]))
}

func makePath(hash []byte, divide int) string {
	var subHashes [][]byte

	subhashLength := len(hash) / divide

	for i := 0; i < divide; i++ {
		subHashes = append(subHashes, hash[i*subhashLength:(i+1)*subhashLength])
	}

	path := PagesPath
	for _, v := range subHashes {
		path += "/" + string(v)
	}

	return path
}

// constructKeyFromRequest uses an array config.RequestKey
// in order to construct a key for mapping this one with
// values of page on a disk and its metadata in DB.
func (p *CachingProperties) constructKeyFromRequest(req *http.Request) string {
	return req.URL.String()
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
	cacheControl := strings.Split(cacheControlString, ",")
	for i := range cacheControl {
		cacheControl[i] = strings.TrimSpace(cacheControl[i])
	}
	for _, v := range cacheControl {
		switch v {
		case "only-if-cached":
			result.OnlyIfCached = true
		case "no-cache":
			result.NoCache = true
		case "no-store":
			result.NoStore = true
		case "no-transform":
			result.NoTransform = true
		default:
			switch {
			case strings.Contains(v, "max-age"):
				_, t, _ := strings.Cut(v, "=")
				age, _ := strconv.Atoi(t)
				if age == 0 {
					result.MaxAge = infinityTime
				} else {
					result.MaxAge = time.Now().Add(time.Duration(age) * time.Second)
				}
			case strings.Contains(v, "max-stale"):
				_, t, _ := strings.Cut(v, "=")
				age, _ := strconv.Atoi(t)
				result.MaxStale = int64(age)
			case strings.Contains(v, "min-fresh"):
				_, t, _ := strings.Cut(v, "=")
				age, _ := strconv.Atoi(t)
				result.MinFresh = time.Now().Add(time.Duration(age) * time.Second)
			}
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
	cacheControl := strings.Split(cacheControlString, ",")
	for i := range cacheControl {
		cacheControl[i] = strings.TrimSpace(cacheControl[i])
	}
	for _, v := range cacheControl {
		switch v {
		case "must-revalidate":
			result.MustRevalidate = true
		case "no-cache":
			result.NoCache = true
		case "no-store":
			result.NoStore = true
		case "no-transform":
			result.NoTransform = true
		case "private":
			result.Private = true
		case "proxy-revalidate":
			result.ProxyRevalidate = true
		default:
			switch {
			case strings.Contains(v, "max-age"):
				_, t, _ := strings.Cut(v, "=")
				age, _ := strconv.Atoi(t)
				if age == 0 {
					result.MaxAge = infinityTime
				} else {
					result.MaxAge = time.Now().Add(time.Duration(age) * time.Second)
				}
			case strings.Contains(v, "s-maxage"):
				_, t, _ := strings.Cut(v, "=")
				age, _ := strconv.Atoi(t)
				if age == 0 {
					result.SMaxAge = nullTime
				} else {
					result.SMaxAge = time.Now().Add(time.Duration(age) * time.Second)
				}
			}
		}
	}

	return result
}
