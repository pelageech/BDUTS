package timer

import (
	"log"
	"net/http"
	"time"
)

// MakeRequestTimeTracker sticks functions to the request that is called while
// the request processes. Functions put some time points for calculating backend time
// and full-trip time.
func MakeRequestTimeTracker(
	handler func(rw http.ResponseWriter, req *http.Request) error,
	saver func(t time.Duration),
	saveOnError bool,
) func(rw http.ResponseWriter, req *http.Request) error {
	return func(rw http.ResponseWriter, req *http.Request) error {
		start := time.Now()
		err := handler(rw, req)
		if err == nil || saveOnError {
			saver(time.Since(start))
		}
		return err
	}
}

// SaveTimerDataGotFromCache uses a pointer at time for saving it to some DB
// if the response is got from cache.
// Pointer is used for saving calling in `defer` functions.
func SaveTimerDataGotFromCache(cacheTime time.Duration) {
	log.Println("Full transferring time: ", cacheTime)
}

// SaveTimeDataBackend is used for saving backend to DB.
// Uses pointer for using in functions with `defer` prefix.
func SaveTimeDataBackend(backendTime time.Duration) {
	log.Println("Backend time: ", backendTime)

}

func SaveTimeFullTrip(fullTime time.Duration) {
	log.Println("Full round trip time: ", fullTime)
}
