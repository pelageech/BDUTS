package timer

import (
	"net/http"
	"os"
	"time"

	"github.com/charmbracelet/log"
)

var logger = log.NewWithOptions(os.Stderr, log.Options{
	ReportTimestamp: true,
	ReportCaller:    true,
})

func LoggerConfig(prefix string) {
	logger.SetPrefix(prefix)
}

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
	logger.Infof("Full transferring time: %v", cacheTime)
}

// SaveTimeDataBackend is used for saving backend to DB.
// Uses pointer for using in functions with `defer` prefix.
func SaveTimeDataBackend(backendTime time.Duration) {
	logger.Infof("Backend time: %v", backendTime)
}

func SaveTimeFullTrip(fullTime time.Duration) {
	logger.Infof("Full round trip time: %v", fullTime)
}
