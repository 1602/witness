package witness

import (
	"crypto/tls"
	"net/http/httptrace"
	"net/textproto"
	"testing"
	"time"
)

func TestTimeline(t *testing.T) {
	t.Run("event names", func(t *testing.T) {
		tl := newTimeline(time.Now())
		tr := tl.tracer()
		cases := map[string]func(){
			"Got100Continue":    func() { tr.Got100Continue() },
			"Got1xxResponse":    func() { tr.Got1xxResponse(0, textproto.MIMEHeader{}) },
			"DNSStart":          func() { tr.DNSStart(httptrace.DNSStartInfo{}) },
			"DNSDone":           func() { tr.DNSDone(httptrace.DNSDoneInfo{}) },
			"TLSHandshakeStart": func() { tr.TLSHandshakeStart() },
			"TLSHandshakeDone":  func() { tr.TLSHandshakeDone(tls.ConnectionState{}, nil) },
			"Wait100Continue":   func() { tr.Wait100Continue() },
		}
		for name, fn := range cases {
			tl.Events = make([]Event, 0, 1)
			fn()
			if tl.Events[0].Name != name {
				t.Errorf("expected event with name %v, got %v", name, tl.Events[0].Name)
			}
		}
	})
}
