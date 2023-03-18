package timer

import (
	"log"
	"net/http"
	"net/http/httptrace"
	"time"
)

func MakeRequestTimeTracker(req *http.Request) (*http.Request, *time.Duration) {
	var start time.Time
	var finishBackend time.Duration

	trace := &httptrace.ClientTrace{
		WroteRequest: func(_ httptrace.WroteRequestInfo) {
			start = time.Now()
		},

		GotFirstResponseByte: func() {
			finishBackend = time.Since(start)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	return req, &finishBackend
}

// SaveTimerDataGotFromCache uses a pointer at time for saving it to some DB
// if the response is got from cache.
// Pointer is used for saving calling in `defer` functions.
func SaveTimerDataGotFromCache(cacheTime *time.Duration) {
	log.Println("Full transferring time: ", *cacheTime)
}

// SaveTimeDataBackend is used for saving backend and full-trip time to some DB.
// Uses pointer for using in functions with `defer` prefix.
func SaveTimeDataBackend(backendTime *time.Duration, fullTime *time.Duration) {
	log.Println("Backend time: ", *backendTime)
	log.Println("Full round trip time: ", *fullTime)
}
