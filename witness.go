package witness

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"time"
)

type customTransport func(req *http.Request) (*http.Response, error)

func (f customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type RoundTripLog struct {
	RequestLog  RequestLog  `json:"requestLog"`
	ResponseLog ResponseLog `json:"responseLog"`
	Timeline    *Timeline   `json:"timeline"`
}

type RequestLog struct {
	Method string              `json:"method"`
	Url    string              `json:"url"`
	Query  map[string][]string `json:"query"`
	Header http.Header         `json:"header"`
	Body   string              `json:"body"`
}

type ResponseLog struct {
	Status        string      `json:"status"`
	StatusCode    int         `json:"statusCode"`
	Header        http.Header `json:"header"`
	ContentLength int64       `json:"contentLength"`
	Body          string      `json:"body"`
	Latency       string      `json:"latency"`
	LatencyNano   int64       `json:"latencyNano"`
}

// Notifier interface must be implemented by a transport.
type Notifier interface {
	Init(context.Context)
	Notify(RoundTripLog)
}

var DefaultNotifier Notifier = NewSSENotifier()

func DebugClient(client *http.Client, ctx context.Context) {
	DefaultNotifier.Init(ctx)
	InstrumentClient(client, DefaultNotifier, true)
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
		var latency time.Duration
		if includeBody {
			req.Body = &bodyWrapper{
				body: req.Body,
				onReadingStart: func() {
					timeline.logEvent("RequestBodyReadingStart", nil)
				},
				onReadingDone: func() {
					timeline.logEvent("RequestBodyReadingDone", nil)
				},
				onClose: func(bw *bodyWrapper) {
					timeline.logEvent("RequestBodyClosed", nil)
					requestBody = string(bw.content)
				},
			}
		}
		res, err := tr.RoundTrip(req)

		// fmt.Println("read req body", requestBody, "okay")

		payload := &RoundTripLog{
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
			},
			timeline,
		}

		if includeBody {
			res.Body = &bodyWrapper{
				body: res.Body,
				onReadingStart: func() {
					timeline.logEvent("ResponseBodyReadingStart", nil)
				},
				onReadingDone: func() {
					timeline.logEvent("ResponseBodyReadingDone", nil)
				},
				onClose: func(bw *bodyWrapper) {
					timeline.logEvent("ResponseBodyClosed", nil)
					payload.ResponseLog.Body = string(bw.content)
					latency = time.Now().Sub(startedAt)
					payload.ResponseLog.Latency = latency.String()
					payload.ResponseLog.LatencyNano = latency.Nanoseconds()
					n.Notify(*payload)
				},
			}
		} else {
			latency = time.Now().Sub(startedAt)
			payload.ResponseLog.Latency = latency.String()
			payload.ResponseLog.LatencyNano = latency.Nanoseconds()
			n.Notify(*payload)
		}
		return res, err
	})
}
