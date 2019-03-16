package witness

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHttpObserver(t *testing.T) {
	client := &http.Client{}
	DebugClient(client)
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := make(map[string]string)
		m["version"] = "v42.0.0"
		err := json.NewEncoder(w).Encode(m)
		if err != nil {
			fmt.Println(err)
		}
	}))
	defer testServer.Close()
	api := API{client, "https://api.automationcloud.net"}
	for i := 0; i < 1; i++ {
		api.CheckStatus()
		log.Println("ping")
		// time.Sleep(10 * time.Second)
	}

}

type API struct {
	Client  *http.Client
	baseURL string
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
	// handling error and doing stuff with body that needs to be unit tested
	return body, err
}
