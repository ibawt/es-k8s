package main

import (
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

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

func createOrModifyDeployment(evt ElasticSearchEvent) error {
	return nil
}

func createConfig() *rest.Config {
	config, err := clientcmd.BuildConfigFromFlags("", "/Users/ian/.kube/config")
	if err != nil {
		log.WithError(err).Warn("building client config")
	}
	return config
}

func watchWithDynamicThingy() {
	config := createConfig()

	gv := unversioned.GroupVersion{Group: "ibawt.ca", Version: "v1"}
	config.GroupVersion = &gv

	client, err := dynamic.NewClient(config)
	if err != nil {
		log.WithError(err).Warn("dynamic client")
	}
	log.WithField("client", client).Info("dynamic")
	resource := unversioned.APIResource{Name: "elasticsearch", Namespaced: true}
	foo, err := client.Resource(&resource, "default").List(&v1.ListOptions{})
	log.WithError(err).WithField("foo", foo).Warn("output")

}

func watchWithClient() {
	config := createConfig()

	config.GroupVersion = &unversioned.GroupVersion{Group: "ibawt.ca", Version: "v1"}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.WithError(err).Warn("NewForConfig")
	}
	b, err := clientset.Core().GetRESTClient().Get().Resource("elasticsearch").Do().Raw()

	if err != nil {
		log.WithError(err).Warn("Get()")
	}
	log.WithField("bytes", string(b)).Warn("output")
	// watchChan := watch.ResultChan()

	// for {
	//	select {
	//	case evt := <-watchChan:
	//		log.WithField("event", evt).Info("returned from watch")
	//	}
	// }
}

func watchEvents() {
	client := http.Client{}
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
				log.WithField("event", event).Info("Processing event!")
				// handle event
				switch event.Type {
				case "MODIFIED":
					err = createOrModifyDeployment(event)
					if err != nil {
						log.WithError(err).Warn("createOrModifyDeployment")
					}
				default:
					log.WithField("Type", event.Type).Warn("Not handled!")
				}
			}
		}
	}
}

func main() {
	log.Info("es-k8s starting...")
	log.SetLevel(log.DebugLevel)
	signalChan := make(chan os.Signal)

	//go watchEvents()
	go watchWithDynamicThingy()
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	os.Exit(0)
}
