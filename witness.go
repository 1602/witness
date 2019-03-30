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
	Timeline    RequestTimeline
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
}

type RequestTimeline struct {
	StartedAt   time.Time
	DnsDone     int64
	GotConn     int64
	Ttfb        int64
	Latency     string
	LatencyNano int64
}

func DebugClient(client *http.Client) {
	firstClientConnected := make(chan bool, 1)
	n := NewTransport(firstClientConnected)
	go (func() {
		// log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:1602", notifier))
	})()

	log.Println("hello", <-firstClientConnected)

	InstrumentClient(client, n, true)
}

type Notifier interface {
	Notify(RoundTripLog)
}

func InstrumentClient(client *http.Client, n Notifier, includeBody bool) {
	tr := client.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}

	client.Transport = customTransport(func(req *http.Request) (*http.Response, error) {
		var (
			dnsDone int64
			gotConn int64
			ttfb    int64
		)
		startedAt := time.Now()
		trace := &httptrace.ClientTrace{
			DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
				dnsDone = time.Now().Sub(startedAt).Nanoseconds()
				// fmt.Printf("DNS Info: %+v\n", dnsInfo)
			},
			GotConn: func(connInfo httptrace.GotConnInfo) {
				gotConn = time.Now().Sub(startedAt).Nanoseconds()
				// fmt.Printf("Got Conn: %+v\n", connInfo)
			},
			GotFirstResponseByte: func() {
				ttfb = time.Now().Sub(startedAt).Nanoseconds()
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		var requestBody string
		if includeBody {
			req.Body = &BodyWrapper{
				body: req.Body,
				onClose: func(bw *BodyWrapper) {
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
			},
			RequestTimeline{
				StartedAt:   startedAt,
				DnsDone:     dnsDone,
				GotConn:     gotConn,
				Ttfb:        ttfb,
				Latency:     latency.String(),
				LatencyNano: latency.Nanoseconds(),
			},
		}

		if includeBody {
			res.Body = &BodyWrapper{
				body: res.Body,
				onClose: func(bw *BodyWrapper) {
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
