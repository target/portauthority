// Copyright 2017 clair authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright (c) 2018 Target Brands, Inc.

package api

import (
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/julienschmidt/httprouter"

	"github.com/target/portauthority/api/v1"
	"github.com/target/portauthority/pkg/clair/client"
	"github.com/target/portauthority/pkg/datastore"
)

// router is an HTTP router that forwards requests to the appropriate sub-router
// depending on the API version specified in the request URI.
type router map[string]*httprouter.Router

// Max API versions
const apiVersionLength = len("v99")

func newAPIHandler(cfg *Config, cc clairclient.Client, store datastore.Backend) http.Handler {
	router := make(router)
	router["/v1"] = v1.NewRouter(store, cc, cfg.ImageWebhookDefaultBlock, cfg.RegAuth)
	return router
}

func (rtr router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlStr := r.URL.String()
	var version string
	if len(urlStr) >= apiVersionLength {
		version = urlStr[:apiVersionLength]
	}

	if router, _ := rtr[version]; router != nil {
		// Remove the version number from the request path to let the router do its
		// job but do not update the RequestURI
		r.URL.Path = strings.Replace(r.URL.Path, version, "", 1)
		router.ServeHTTP(w, r)
		return
	}

	log.WithFields(log.Fields{"status": http.StatusNotFound, "method": r.Method, "request uri": r.RequestURI, "remote addr": r.RemoteAddr}).Info("Served HTTP request")
	http.NotFound(w, r)
}

func newHealthHandler(store datastore.Backend) http.Handler {
	router := httprouter.New()
	router.GET("/health", healthHandler(store))
	return router
}

func healthHandler(store datastore.Backend) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		header := w.Header()
		header.Set("Server", "portauthority")

		status := http.StatusInternalServerError
		if store.Ping() {
			status = http.StatusOK
		}

		w.WriteHeader(status)
	}
}
