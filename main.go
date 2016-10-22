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
	resourceEndpoint = "apis/ibawt.ca/v1/namespaces/default/elastic-search"
)

type Event struct {
	Type   string `json:"type"`
	Object string `json:"object"`
}
type Metadata map[string]string

type Object struct {
	ApiVersion string
	Kind       string
	Metadata   Metadata
}

type ResourceAdapter interface {
	OnInit(Metadata) error
	OnChange(Metadata) error
	OnDestroy(Metadata) error
}

const defaultTimeout = 5 * time.Second

func watchEvents(quitChan <-chan bool) (<-chan Event, <-chan error) {
	eventChan := make(chan Event)
	errorChan := make(chan error)

	go func() {
		client := http.Client{Timeout: defaultTimeout}
		endPoint := apiHost + "/" + resourceEndpoint + "?watch=true"
		for {
		top:
			resp, err := client.Get(endPoint)
			if err != nil {
				errorChan <- err
			} else if resp.StatusCode != http.StatusOK {
				log.WithField("StatusCode", resp.StatusCode).Warn("Invalid response sleeping...")
				time.Sleep(5 * time.Second)
			} else {
				decoder := json.NewDecoder(resp.Body)
				// keep decoding events
				for {
					select {
					case <-quitChan:
						log.Info("watchEvents rx'd a quit, exiting...")
						err = resp.Body.Close()
						if err != nil {
							log.WithError(err).Warn("Error closing response body")
						}
						close(eventChan)
						close(errorChan)
						return
					default:
						var event Event
						err = decoder.Decode(&event)
						log.Info("Decoded an event!")
						if err != nil {
							errorChan <- err
							err = resp.Body.Close()
							if err != nil {
								log.WithError(err).Warn("Error closing response body")
							}
							goto top
						} else {
							eventChan <- event
						}
					}
				}
			}
		}
	}()

	return eventChan, errorChan
}

func main() {
	quitChan := make(chan bool)
	signalChan := make(chan os.Signal)

	eventChan, errChan := watchEvents(quitChan)

	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case err := <-errChan:
			log.WithError(err).Warn("Error in main loop")
		case evt, ok := <-eventChan:
			log.Infof("recieved event!: %v\n", evt)
			if !ok {
				log.Warn("Event Channel is closed exiting...")
				return
			}
		case <-signalChan:
			log.Info("Received signal exiting...")
			quitChan <- true
			return
		}
	}
}
