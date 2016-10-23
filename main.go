package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
)

var (
	apiHost          = "http://127.0.0.1:8001"
	resourceEndpoint = "/apis/ibawt.ca/v1/namespaces/default/elasticsearchs"
)

// ElasticSearchEvent thingy
type ElasticSearchEvent struct {
	Type   string        `json:"type"`
	Object ElasticSearch `json:"object"`
}

// ElasticSearch thingy
type ElasticSearch struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       ElasticSearchSpec      `json:"spec"`
}

// ElasticSearchSpec thingy
type ElasticSearchSpec struct {
	NumDataNodes   int `json:"dataNodes"`
	NumClientNodes int `json:"clientNodes"`
	NumMasterNodes int `json:"masterNodes"`
}

// ElasticSearchList thingy
type ElasticSearchList struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Items      []ElasticSearch        `json:"items"`
}

const defaultTimeout = 5 * time.Second

func watchEvents() {
	client := http.Client{Timeout: 90 * time.Second}
	endPoint := apiHost + resourceEndpoint + "?watch=true"

	for {
	top:
		log.WithField("EndPoint", endPoint).Debug("Polling...")
		resp, err := client.Get(endPoint)
		if err != nil {
			log.WithError(err).Warn("Get")
		} else if resp.StatusCode != http.StatusOK {
			log.WithField("StatusCode", resp.StatusCode).Warn("Invalid response sleeping...")
		}
		if err != nil || resp.StatusCode != http.StatusOK {
			time.Sleep(5 * time.Second)
		} else {
			decoder := json.NewDecoder(resp.Body)
			// keep decoding events
			for {
				var event ElasticSearchEvent
				err = decoder.Decode(&event)

				if err != nil {
					log.WithError(err).Warn("Error in decode")
					err = resp.Body.Close()
					if err != nil {
						log.WithError(err).Warn("Error closing response body")
					}
					goto top
				}
				// handle event
			}
		}
	}
}

func main() {
	log.Info("es-k8s starting...")
	log.SetLevel(log.DebugLevel)
	signalChan := make(chan os.Signal)

	go watchEvents()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	os.Exit(0)
}
