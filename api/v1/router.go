package v1

import (
	"net/http"
	"strconv"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/target/portauthority/pkg/clair/client"
	"github.com/target/portauthority/pkg/datastore"
)

var (
	promResponseDurationMilliseconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "portauthority_api_response_duration_milliseconds",
		Help:    "The duration of time it takes to receieve and write a response to an API request",
		Buckets: prometheus.ExponentialBuckets(9.375, 2, 10),
	}, []string{"route", "code"})
)

func init() {
	prometheus.MustRegister(promResponseDurationMilliseconds)
}

type handler func(http.ResponseWriter, *http.Request, httprouter.Params, *context) (route string, status int)

func httpHandler(h handler, ctx *context) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		start := time.Now()
		route, status := h(w, r, p, ctx)
		statusStr := strconv.Itoa(status)
		if status == 0 {
			statusStr = "???"
		}

		promResponseDurationMilliseconds.
			WithLabelValues(route, statusStr).
			Observe(float64(time.Since(start).Nanoseconds()) / float64(time.Millisecond))

		log.WithFields(log.Fields{"remote addr": r.RemoteAddr, "method": r.Method, "request uri": r.RequestURI, "status": statusStr, "elapsed time": time.Since(start)}).Info("Handled HTTP request")
	}
}

type context struct {
	Store                    datastore.Backend
	ClairClient              clairclient.Client
	ImageWebhookDefaultBlock bool
	RegAuth                  []map[string]string
}

// NewRouter creates an HTTP router for version 1 of the Port Authority API
func NewRouter(store datastore.Backend, cc clairclient.Client, imageWebhookDefaultBlock bool, regAuth []map[string]string) *httprouter.Router {
	router := httprouter.New()
	ctx := &context{store, cc, imageWebhookDefaultBlock, regAuth}

	// Images
	router.GET("/images", httpHandler(listImages, ctx))
	router.GET("/images/:id", httpHandler(getImage, ctx))
	router.POST("/images", httpHandler(postImage, ctx))

	// Policies
	router.GET("/policies", httpHandler(listPolicy, ctx))
	router.GET("/policies/:name", httpHandler(getPolicy, ctx))
	router.POST("/policies", httpHandler(postPolicy, ctx))

	// Kubernetes Image Policy Webhook
	router.POST("/k8s-image-policy-webhook", httpHandler(postK8sImagePolicy, ctx))

	// Crawlers
	router.GET("/crawlers/:id", httpHandler(getCrawler, ctx))
	router.POST("/crawlers/:type", httpHandler(postCrawler, ctx))

	// Containers place holder
	router.GET("/containers", httpHandler(listContainers, ctx))
	router.GET("/containers/:id", httpHandler(getContainer, ctx))

	// Metrics
	router.GET("/metrics", httpHandler(getMetrics, ctx))

	return router
}
