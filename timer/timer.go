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
		WroteHeaders: func() {
			start = time.Now()
		},

		GotFirstResponseByte: func() {
			finishBackend = time.Since(start)
		},
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	return req, &finishBackend
}

func SaveTimerDataGotFromCache(cacheTime time.Duration) {
	log.Println("Full transferring time: ", cacheTime)
}

func SaveTimeDataBackend(backendTime time.Duration, fullTime time.Duration) {
	log.Println("Backend time: ", backendTime)
	log.Println("Full round trip time: ", fullTime)
}
