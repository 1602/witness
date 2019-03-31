package witness

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type sse struct {
	distributor          chan []byte
	openingClients       chan chan []byte
	connectedClients     map[chan []byte]bool
	closingClients       chan chan []byte
	firstClient          chan bool
	firstClientConnected bool
	done                 chan bool
}

func (t *sse) Notify(rtl RoundTripLog) {
	json := serializeOrDie(rtl)
	t.distributor <- json
}

func serializeOrDie(stuff interface{}) []byte {
	json, err := json.Marshal(stuff)
	if err != nil {
		panic(err)
	}
	return json
}

func NewTransport(firstClientReady, done chan bool) (transport *sse) {
	transport = &sse{
		distributor:          make(chan []byte),
		openingClients:       make(chan chan []byte),
		connectedClients:     make(map[chan []byte]bool),
		closingClients:       make(chan chan []byte),
		firstClient:          firstClientReady,
		firstClientConnected: false,
		done:                 done,
	}

	go transport.route()

	return transport
}

func (t *sse) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	flusher, flusherSupported := rw.(http.Flusher)

	if !flusherSupported {
		http.Error(rw, "the Flusher interface is not implemented by ResponseWriter", http.StatusInternalServerError)
		return
	}

	header := rw.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("Access-Control-Allow-Origin", "*")

	ch := make(chan []byte)

	if !t.firstClientConnected {
		t.firstClientConnected = true
		t.firstClient <- true
	}

	t.openingClients <- ch

	defer func() {
		t.closingClients <- ch
	}()

	go func() {
		<-req.Context().Done()
		t.closingClients <- ch
	}()

	for {
		select {
		case data := <-ch:
			fmt.Fprintf(rw, "data: %s\n\n", data)
			flusher.Flush()
		case <-t.done:
			return
		}
	}
}

func (t *sse) route() {
	for {
		select {
		case s := <-t.openingClients:
			t.connectedClients[s] = true
		case event := <-t.distributor:
			for c := range t.connectedClients {
				c <- event
			}
		case s := <-t.closingClients:
			delete(t.connectedClients, s)
		}
	}
}
