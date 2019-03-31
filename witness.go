package witness

import (
	"log"
	"net/http"
	"net/http/httptrace"
	"time"
)

type customTransport func(req *http.Request) (*http.Response, error)

func (f customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type RoundTripLog struct {
	RequestLog  RequestLog
	ResponseLog ResponseLog
	Timeline    *Timeline
}

type RequestLog struct {
	Method string
	Url    string
	Query  map[string][]string
	Header http.Header
	Body   string
}

type ResponseLog struct {
	Status        string
	StatusCode    int
	Header        http.Header
	ContentLength int64
	Body          string
	Latency       string
	LatencyNano   int64
}

// Notifier interface must be implemented by a transport.
type Notifier interface {
	Notify(RoundTripLog)
}

func DebugClient(client *http.Client) {
	firstClientConnected := make(chan bool, 1)
	n := NewTransport(firstClientConnected, nil)

	go (func() {
		// TODO: make configurable
		log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:1602", n))
	})()

	// TODO: make configurable
	// wait until first client connected
	<-firstClientConnected

	InstrumentClient(client, n, true)
}

func InstrumentClient(client *http.Client, n Notifier, includeBody bool) {
	tr := client.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}

	client.Transport = customTransport(func(req *http.Request) (*http.Response, error) {
		startedAt := time.Now()
		timeline := newTimeline(startedAt)
		trace := timeline.tracer()
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		var requestBody string
		if includeBody {
			req.Body = &BodyWrapper{
				body: req.Body,
				onReadingStart: func() {
					timeline.logEvent("RequestBodyReadingStart", nil)
				},
				onReadingDone: func() {
					timeline.logEvent("RequestBodyReadingDone", nil)
				},
				onClose: func(bw *BodyWrapper) {
					timeline.logEvent("RequestBodyClosed", nil)
					requestBody = string(bw.content)
				},
			}
		}
		res, err := tr.RoundTrip(req)
		latency := time.Now().Sub(startedAt)

		// fmt.Println("read req body", requestBody, "okay")

		payload := RoundTripLog{
			RequestLog{
				Method: req.Method,
				Url:    req.URL.String(),
				Query:  req.URL.Query(),
				Header: req.Header,
				Body:   requestBody,
			},
			ResponseLog{
				Status:        string(res.Status),
				StatusCode:    res.StatusCode,
				Header:        res.Header,
				ContentLength: res.ContentLength,
				Latency:       latency.String(),
				LatencyNano:   latency.Nanoseconds(),
			},
			timeline,
		}

		if includeBody {
			res.Body = &BodyWrapper{
				body: res.Body,
				onReadingStart: func() {
					timeline.logEvent("ResponseBodyReadingStart", nil)
				},
				onReadingDone: func() {
					timeline.logEvent("ResponseBodyReadingDone", nil)
				},
				onClose: func(bw *BodyWrapper) {
					timeline.logEvent("ResponseBodyClosed", nil)
					payload.ResponseLog.Body = string(bw.content)
					n.Notify(payload)
				},
			}
		} else {
			n.Notify(payload)
		}
		return res, err
	})
}
