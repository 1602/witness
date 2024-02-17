package witness

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNotify(t *testing.T) {
	url := "http://example.com"
	tr := NewSSENotifier()
	go tr.Notify(RoundTripLog{RequestLog: &RequestLog{Url: url}})
	msg := <-tr.distributor
	if !strings.Contains(string(msg), url) {
		t.Errorf(`Expected msg to contain "%v", got %s`, url, msg)
	}
}

func TestSerializeOrDie(t *testing.T) {
	t.Run("serialize", func(t *testing.T) {
		result := string(serializeOrDie(1))
		if result != "1" {
			t.Errorf("Expected 1, got %v", result)
		}
	})

	t.Run("die", func(t *testing.T) {
		expectedPanic := "json: unsupported type: chan bool"
		defer func() {
			err := recover().(error).Error()
			if err != expectedPanic {
				t.Errorf("expected panic %v, got %v", expectedPanic, err)
			}
		}()
		serializeOrDie(make(chan bool))
	})
}

func TestServeHTTP(t *testing.T) {
	t.Run("waiting for the first client", func(t *testing.T) {
		// create sse notifier - an http handler for streaming server
		tr := NewSSENotifier()
		// start streaming server (mock of sse server)
		sse := httptest.NewUnstartedServer(tr)
		defer sse.Close()
		ctx, cancel := context.WithCancel(context.TODO())

		// override startServer function to start our replacement test server
		// instead of the real one
		tr.startServer = func() {
			sse.Start()

			go func() {
				req, _ := http.NewRequest("GET", sse.URL, nil)
				client := &http.Client{Timeout: 10 * time.Second}
				res, err := client.Do(req)
				if err != nil {
					fmt.Println(res, err)
				}

				reader := bufio.NewReader(res.Body)
				// sse is a streaming server so we have to read line by line
				// just one line is enough here, because we only going to log one-1
				l, _ := reader.ReadBytes('\n')

				res.Body.Close()
				result := string(l)

				if !strings.HasPrefix(result, "data:") {
					t.Errorf("expected body to have prefix 'data:', got %s", result)
				}

				if !strings.Contains(result, "example.com") {
					t.Errorf("expected body to contain 'example.com', got %s", result)
				}

				cancel()
			}()
		}

		tr.Init(ctx)
		// this will cause sse to send one line to its connected client(s)
		tr.Notify(RoundTripLog{RequestLog: &RequestLog{Url: "http://example.com"}})
	})

	t.Run("flusher not supported", func(t *testing.T) {
		xx := &x{make(map[string][]string), 0, ""}
		tr := NewSSENotifier()
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		tr.ServeHTTP(xx, req)
		if xx.statusCode != 500 {
			t.Error("Expected status 500")
		}

		if xx.body != "the Flusher interface is not implemented by ResponseWriter\n" {
			t.Error("Expected flusher")
		}
	})
}

type x struct {
	header     http.Header
	statusCode int
	body       string
}

func (xx *x) Header() http.Header {
	return xx.header
}

func (xx *x) Write(p []byte) (int, error) {
	xx.body = string(p)
	return 0, nil
}

func (xx *x) WriteHeader(statusCode int) {
	xx.statusCode = statusCode
}
