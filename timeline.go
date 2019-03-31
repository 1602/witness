package witness

import (
	"crypto/tls"
	"net/http/httptrace"
	"net/textproto"
	"time"
)

type Timeline struct {
	StartedAt time.Time `json:"startedAt"`
	Events    []Event   `json:"events"`
}

type Event struct {
	Name    string      `json:"name"`
	Payload interface{} `json:"payload"`
	Delay   int64       `json:"delay"`
}

func newTimeline(startedAt time.Time) *Timeline {
	return &Timeline{
		StartedAt: startedAt,
		Events:    make([]Event, 0, 25),
	}
}

func (tl *Timeline) tracer() *httptrace.ClientTrace {
	return &httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			tl.logEvent("GetConn", hostPort)
		},

		GotConn: func(connInfo httptrace.GotConnInfo) {
			tl.logEvent("GotConn", connInfo)
		},

		PutIdleConn: func(err error) {
			tl.logEvent("PutIdleConn", err)
		},

		GotFirstResponseByte: func() {
			tl.logEvent("GotFirstResponseByte", nil)
		},

		Got100Continue: func() {
			tl.logEvent("Got100Continue", nil)
		},

		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			tl.logEvent("Got1xxResponse", struct {
				Code   int                  `json:"code"`
				Header textproto.MIMEHeader `json:"header"`
			}{code, header})
			return nil
		},

		DNSStart: func(i httptrace.DNSStartInfo) {
			tl.logEvent("DNSStart", i)
		},

		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			tl.logEvent("DNSDone", dnsInfo)
		},

		ConnectStart: func(network, addr string) {
			tl.logEvent("ConnectStart", struct {
				Network string `json:"network"`
				Addr    string `json:"addr"`
			}{network, addr})
		},

		ConnectDone: func(network, addr string, err error) {
			tl.logEvent("ConnectDone", struct {
				Network string `json:"network"`
				Addr    string `json:"addr"`
				Err     error  `json:"err"`
			}{network, addr, err})
		},

		TLSHandshakeStart: func() {
			tl.logEvent("TLSHandshakeStart", nil)
		},

		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			// TODO: enable if configurable (too much info)
			// tl.logEvent("TLSHandshakeDone", state)
			tl.logEvent("TLSHandshakeDone", nil)
		},

		WroteHeaderField: func(key string, value []string) {
			tl.logEvent("WroteHeaderField", struct {
				Key   string   `json:"key"`
				Value []string `json:"value"`
			}{key, value})
		},

		WroteHeaders: func() {
			tl.logEvent("WroteHeaders", nil)
		},

		Wait100Continue: func() {
			tl.logEvent("Wait100Continue", nil)
		},

		WroteRequest: func(i httptrace.WroteRequestInfo) {
			tl.logEvent("WroteRequestInfo", i)
		},
	}
}

func (tl *Timeline) logEvent(name string, payload interface{}) {
	tl.Events = append(
		tl.Events,
		Event{
			name,
			payload,
			time.Now().Sub(tl.StartedAt).Nanoseconds() / 100000,
		},
	)
}
