package witness

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/google/uuid"
)

type customTransport func(req *http.Request) (*http.Response, error)

func (f customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type RoundTripLog struct {
	ID           string        `json:"id"`
	RequestLog   *RequestLog   `json:"requestLog"`
	ResponseLog  *ResponseLog  `json:"responseLog"`
	Error        *RequestError `json:"error"`
	Timeline     *Timeline     `json:"timeline"`
	Duration     string        `json:"duration"`
	DurationNano int64         `json:"durationNano"`
	Done         bool          `json:"done"`
}

type RequestError struct {
	Message string `json:"message"`
	Details error  `json:"details"`
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
		var requestBody string
		startedAt := time.Now()
		id := uuid.NewString()
		timeline := newTimeline(startedAt)
		requestLog := &RequestLog{
			Method: req.Method,
			Url:    req.URL.String(),
			Query:  req.URL.Query(),
			Header: req.Header,
			Body:   requestBody,
		}
		payload := &RoundTripLog{
			ID:         id,
			RequestLog: requestLog,
			Timeline:   timeline,
		}
		trace := timeline.tracer(func() {
			n.Notify(*payload)
		})
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

		var duration time.Duration
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
					requestLog.Body = string(bw.content)
				},
			}
		}

		res, err := tr.RoundTrip(req)

		if res != nil {
			payload.ResponseLog = &ResponseLog{
				Status:        string(res.Status),
				StatusCode:    res.StatusCode,
				Header:        res.Header,
				ContentLength: res.ContentLength,
			}
			n.Notify(*payload)
		}

		if err != nil {
			payload.Error = &RequestError{
				Message: err.Error(),
				Details: err,
			}
		}

		if includeBody && res != nil && res.Body != nil {
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
					duration = time.Now().Sub(startedAt)
					payload.Done = true
					if payload.ResponseLog != nil {
						payload.ResponseLog.Body = string(bw.content)
						if payload.ResponseLog.ContentLength == -1 {
							payload.ResponseLog.ContentLength = int64(len(bw.content))
						}
					}
					payload.Duration = roundDuration(duration, 1).String()
					payload.DurationNano = duration.Nanoseconds()
					n.Notify(*payload)
				},
			}
		} else {
			duration = time.Now().Sub(startedAt)
			payload.Done = true
			payload.Duration = roundDuration(duration, 1).String()
			payload.DurationNano = duration.Nanoseconds()
			n.Notify(*payload)
		}
		return res, err
	})
}

var divs = []time.Duration{
	time.Duration(1), time.Duration(10), time.Duration(100), time.Duration(1000)}

func roundDuration(d time.Duration, digits int) time.Duration {
	switch {
	case d > time.Second:
		d = d.Round(time.Second / divs[digits])
	case d > time.Millisecond:
		d = d.Round(time.Millisecond / divs[digits])
	case d > time.Microsecond:
		d = d.Round(time.Microsecond / divs[digits])
	}
	return d
}
