package witness

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptrace"
	"time"
)

// CustomTransport
type CustomTransport func(req *http.Request) (*http.Response, error)

// RoundTrip
func (f CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type BodyWrapper struct {
	body             io.ReadCloser
	ReadingStartedAt time.Time
	Content          []byte
}

func (bw *BodyWrapper) Read(p []byte) (n int, err error) {
	now := time.Now()
	if bw.ReadingStartedAt.IsZero() {
		bw.ReadingStartedAt = now
	}
	n, err = bw.body.Read(p)
	fmt.Println(string(p), n, err)
	if bw.Content == nil {
		bw.Content = p[:n]
	} else {
		bw.Content = append(bw.Content, p[:n]...)
	}
	if err == io.EOF {
		fmt.Println("Read body", now.Sub(bw.ReadingStartedAt))
		return
	}
	return
}

func (bw *BodyWrapper) Close() (err error) {
	fmt.Println("Close body", time.Now().Sub(bw.ReadingStartedAt))
	return bw.body.Close()
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
}

type ResponseLog struct {
	Status        string
	StatusCode    int
	Header        http.Header
	ContentLength int64
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
	broker := NewServer(firstClientConnected)
	go (func() {
		log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:1602", broker))
	})()

	log.Println("hello", <-firstClientConnected)

	tr := client.Transport
	if tr == nil {
		tr = http.DefaultTransport
	}

	client.Transport = CustomTransport(func(req *http.Request) (*http.Response, error) {
		var (
			dnsDone int64
			gotConn int64
			ttfb    int64
		)
		startedAt := time.Now()
		trace := &httptrace.ClientTrace{
			DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
				dnsDone = time.Now().Sub(startedAt).Nanoseconds()
				fmt.Printf("DNS Info: %+v\n", dnsInfo)
			},
			GotConn: func(connInfo httptrace.GotConnInfo) {
				gotConn = time.Now().Sub(startedAt).Nanoseconds()
				fmt.Printf("Got Conn: %+v\n", connInfo)
			},
			GotFirstResponseByte: func() {
				ttfb = time.Now().Sub(startedAt).Nanoseconds()
			},
		}
		req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
		fmt.Println("Request", req.URL)
		res, err := tr.RoundTrip(req)
		latency := time.Now().Sub(startedAt)
		fmt.Println("Response", res.Status, latency)
		// res.Body = &BodyWrapper{body: res.Body}

		payload := RoundTripLog{
			RequestLog{
				Method: req.Method,
				Url:    req.URL.String(),
				Query:  req.URL.Query(),
				Header: req.Header,
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
		json, err := json.Marshal(payload)
		broker.Notifier <- []byte(json)
		return res, err
	})
}
