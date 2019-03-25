package witness

import (
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
	readingStartedAt time.Time
	readingStoppedAt time.Time
	content          []byte
	onClose          func(*BodyWrapper)
}

func (bw *BodyWrapper) Read(p []byte) (n int, err error) {
	if bw.readingStartedAt.IsZero() {
		bw.readingStartedAt = time.Now()
	}
	if bw.body == nil {
		return 0, io.EOF
	}
	n, err = bw.body.Read(p)
	// fmt.Println(string(p), n, err)
	if bw.content == nil {
		bw.content = p[:n]
	} else {
		bw.content = append(bw.content, p[:n]...)
	}
	if err == io.EOF {
		// fmt.Println("Read body", now.Sub(bw.readingStartedAt))
		bw.readingStoppedAt = time.Now()
		return
	}
	return
}

func (bw *BodyWrapper) Close() (err error) {
	bw.onClose(bw)
	if bw.body != nil {
		return bw.body.Close()
	}
	return nil
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
	broker := NewServer(firstClientConnected)
	go (func() {
		log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:1602", broker))
	})()

	log.Println("hello", <-firstClientConnected)

	InstrumentClient(client, broker, true)
}

type Notifier interface {
	Notify(interface{})
}

func InstrumentClient(client *http.Client, n Notifier, includeBody bool) {
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
