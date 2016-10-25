package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/errors"
	"k8s.io/client-go/pkg/api/unversioned"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/fields"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
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

var client *kubernetes.Clientset

func getClient() *kubernetes.Clientset {
	if client == nil {
		client = kubernetes.NewForConfigOrDie(createConfig())
	}
	return client
}

func createServiceAccount() error {
	log.Info("creating service account")
	c := getClient()

	svcAcct, err := c.Core().ServiceAccounts(api.NamespaceDefault).Get("elasticsearch")
	if err != nil && !errors.IsNotFound(err) {
		// ignore NotFound
		return err
	} else if err == nil {
		// ServiceAccount already exists
		return nil
	}

	svcAcct = &v1.ServiceAccount{}
	svcAcct.ObjectMeta.SetName("elasticsearch")
	svcAcct, err = c.Core().ServiceAccounts(api.NamespaceDefault).Create(svcAcct)
	if err != nil {
		return err
	}
	log.WithField("ServiceAccount", svcAcct).Info("Created ServiceAccount")
	return nil
}

func createService() error {
	log.Info("Creating Service")
	c := getClient()

	service, err := c.Core().Services(api.NamespaceDefault).Get("elasticsearch")
	if err != nil && !errors.IsNotFound(err) {
		return err
	} else if err == nil {
		return nil
	}

	service = &v1.Service{}
	service.ObjectMeta.SetName("elasticsearch")
	service.ObjectMeta.Labels()
}

func createConfig() *rest.Config {
	config, err := clientcmd.BuildConfigFromFlags("", "/Users/ian/.kube/config")
	if err != nil {
		log.WithError(err).Fatal("building client config")
	}
	return config
}

func (e *ElasticSearchEvent) SetGroupVersionKind(kind unversioned.GroupVersionKind) {
	panic("wtf")
}

func (e *ElasticSearchEvent) GroupVersionKind() unversioned.GroupVersionKind {
	panic("do I get here?")
	return unversioned.FromAPIVersionAndKind("ibawt.ca", "ElasticSearch")
}

func (e *ElasticSearchEvent) GetObjectKind() unversioned.ObjectKind {
	panic("how about here?")
	return e
}

func controllerThingy() {
	config := createConfig()
	config.GroupVersion = &unversioned.GroupVersion{Group: "ibawt.ca", Version: "v1"}
	config.APIPath = "/apis"
	// clientset := kubernetes.NewForConfigOrDie(config)
	// config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: api.Codecs}

	client, _ := rest.RESTClientFor(config)

	// foo, err := client.Get().Namespace(api.NamespaceAll).
	//	Resource("elasticsearchs").Do().Get()

	// log.WithField("err", err).WithField("foo", foo).Info("raw rest")

	source := cache.NewListWatchFromClient(client, "elasticsearchs", api.NamespaceAll, fields.Everything())
	handler := func(obj interface{}) {
		log.Warn(obj)
	}

	store, c := cache.NewInformer(
		source,
		&runtime.Unknown{},
		30*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    handler,
			DeleteFunc: handler,
		})

	stop := make(chan struct{})
	c.Run(stop)
	log.Warn(store, c)
}
func watchWithDynamicThingy() {
	config := createConfig()
	config.APIPath = "/apis"
	gv := unversioned.GroupVersion{Group: "ibawt.ca", Version: "v1"}
	config.GroupVersion = &gv

	client, err := dynamic.NewClient(config)
	if err != nil {
		log.WithError(err).Warn("dynamic client")
	}
	log.WithField("client", client).Info("dynamic")
	resource := unversioned.APIResource{Name: "elasticsearchs", Namespaced: false}
	foo, err := client.Resource(&resource, "").List(&v1.ListOptions{})
	log.WithError(err).WithField("foo", foo).Warnf("output %#v", foo)
}

func watchWithClient() {
	config := createConfig()
	config.GroupVersion = &unversioned.GroupVersion{Group: "ibawt.ca", Version: "v1"}
	config.APIPath = "/apis"
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.WithError(err).Warn("NewForConfig")
	}

	watch, err := clientset.Core().GetRESTClient().Get().Resource("elasticsearchs").Watch()
	if err != nil {
		log.WithError(err).Warn("Get()")
		return
	}
	watchChan := watch.ResultChan()

	for {
		select {
		case evt := <-watchChan:
			log.WithField("event", evt).Info("returned from watch")
		}
	}
}

func listInstances() ([]ElasticSearch, error) {
	endPoint := apiHost + resourceEndpoint

	resp, err := http.Get(endPoint)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Invalid return status code: %d", resp.StatusCode)
	}

	var list ElasticSearchList
	decoder := json.NewDecoder(resp.Body)

	if err = decoder.Decode(&list); err != nil {
		return nil, err
	}

	return list.Items, nil
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
				case "ADDED":
					err = createOrModifyDeployment(event)
					if err != nil {
						log.WithError(err).Warn("createOrModifyDeployment")
					}
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
	flag.Parse()
	log.Info("es-k8s starting...")
	log.SetLevel(log.DebugLevel)
	go watchEvents()
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	os.Exit(0)
}
