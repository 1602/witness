package witness

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type sse struct {
	distributor          chan []byte
	openingClients       chan chan []byte
	connectedClients     map[chan []byte]bool
	closingClients       chan chan []byte
	firstClient          chan bool
	firstClientConnected bool
	ctx                  context.Context
	startServer          func()
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

//go:embed ui
var content embed.FS

func NewSSENotifier() (transport *sse) {
	transport = &sse{
		distributor:          make(chan []byte),
		openingClients:       make(chan chan []byte),
		connectedClients:     make(map[chan []byte]bool),
		closingClients:       make(chan chan []byte),
		firstClientConnected: false,
		startServer: func() {
			mux := http.NewServeMux()
			mux.Handle("/events", transport)
			mux.Handle("/", rootPath("/ui", http.FileServer(http.FS(content))))
			log.Fatal("HTTP server error: ", http.ListenAndServe("localhost:8989", mux))
		},
	}

	return transport
}

func rootPath(staticDir string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			r.URL.Path = fmt.Sprintf("/%s/", staticDir)
		} else {
			b := strings.Split(r.URL.Path, "/")[0]
			if b != staticDir {
				r.URL.Path = fmt.Sprintf("/%s%s", staticDir, r.URL.Path)
			}
		}
		h.ServeHTTP(w, r)
	})
}

func (t *sse) Init(ctx context.Context) {
	t.ctx = ctx
	t.firstClient = make(chan bool, 1)
	go t.route()
	go t.startServer()

	// wait until first client connected
	// TODO: make waiting configurable
	fmt.Println("waiting for the first client to connect to http://localhost:8989/ events streaming server")

	<-t.firstClient

	fmt.Println("first client connected")
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
			// t := time.Now()
			// fmt.Printf("%s sending %d bytes of data\n", t.Format("2006/01/02 15:04:05"), len(data))
			fmt.Fprintf(rw, "data: %s\n\n", data)
			flusher.Flush()
		case <-t.ctx.Done():
			return
		}
	}
}

func (t *sse) route() {
	for {
		select {
		case s := <-t.openingClients:
			fmt.Println("new client connected")
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
