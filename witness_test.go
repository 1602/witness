package witness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// uncomment this for manual testing using frontend inspector cliend
//*
func TestHttpObserver(t *testing.T) {
	client := &http.Client{}
	DebugClient(client)
	testServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := make(map[string]string)
			m["version"] = "v42.0.0"
			err := json.NewEncoder(w).Encode(m)
			if err != nil {
				fmt.Println(err)
			}
		}))
	defer testServer.Close()
	api := API{client, "https://api.automationcloud.net"}
	//for i := 0; i < 1; i++ {
	for {
		api.CheckStatus()
		log.Println("ping")
		time.Sleep(10 * time.Second)
	}

}

//*/

type fakeNotifier struct {
	payload interface{}
}

func (n *fakeNotifier) Notify(p interface{}) {
	n.payload = p
}

func TestInstrumentClient(t *testing.T) {
	client := &http.Client{}
	notifier := &fakeNotifier{}
	InstrumentClient(client, notifier, true)

	testServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// reqBody, err := ioutil.ReadAll(r.Body)
			// fmt.Println("inside handler", string(reqBody), err)

			m := make(map[string]string)
			m["version"] = "v42.0.0"
			err := json.NewEncoder(w).Encode(m)
			if err != nil {
				fmt.Println("encode error", err)
			}
		}))
	defer testServer.Close()

	api := API{client, testServer.URL}
	api.SendPostRequest()

	payload, ok := notifier.payload.(RoundTripLog)
	if !ok {
		t.Error("Unknown payload")
		t.FailNow()
	}

	reqBody := "hello"
	if payload.RequestLog.Body != reqBody {
		t.Errorf("Expected request body to be %v, got %v", reqBody, payload.RequestLog.Body)
	}

	respBody := "{\"version\":\"v42.0.0\"}\n"
	if payload.ResponseLog.Body != respBody {
		t.Errorf("Expected response body to be %v, got %v", respBody, payload.ResponseLog.Body)
	}

	if payload.RequestLog.Method != "POST" {
		t.Errorf("Expected request method to be %v, got %v", "POST", payload.RequestLog.Method)
	}
}

type API struct {
	Client  *http.Client
	baseURL string
}

func (api *API) SendPostRequest() ([]byte, error) {
	req, err := http.NewRequest("POST", api.baseURL+"/status?a=2&foo=bar&foo=baz", bytes.NewBuffer([]byte("hello")))
	if err != nil {
		return nil, err
	}
	req.Header.Set("user-agent", "Go test-runner, v1")
	resp, err := api.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func (api *API) CheckStatus() ([]byte, error) {
	req, err := http.NewRequest("GET", api.baseURL+"/status?a=2&foo=bar&foo=baz", nil)
	req.Header.Set("user-agent", "Go test-runner, v1")
	resp, err := api.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return body, err
}
