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

package v1

import (
	"compress/gzip"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"strconv"
	"strings"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/target/portauthority/pkg/commonerr"
	"github.com/target/portauthority/pkg/crawler"
	"github.com/target/portauthority/pkg/datastore"
	"github.com/target/portauthority/pkg/docker"

	log "github.com/sirupsen/logrus"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// These are the route identifiers for Prometheus
	listImagesRoute = "v1/listImages"
	getImageRoute   = "v1/getImage"
	postImageRoute  = "v1/postImage"

	listPolicyRoute = "v1/listPolicy"
	getPolicyRoute  = "v1/getPolicy"
	postPolicyRoute = "v1/postPolicy"

	postK8sImagePolicyRoute = "v1/postK8sImagePolicy"

	getCrawlerRoute  = "v1/getCrawler"
	postCrawlerRoute = "v1/postCrawler"

	getContainerRoute   = "v1/getContainer"
	listContainersRoute = "v1/listContainers"

	getMetricsRoute = "v1/getMetrics"

	// maxBodySize restricts client request bodies to 1MiB
	maxBodySize int64 = 1048576

	// publicDocker is the default registry used for any image post made to
	// docker.io, docker.com, or blank registry values.
	publicDocker string = "https://registry-1.docker.io"

	// dateTimeLayout is the required format for any datetime query parameters
	dateTimeLayout = "2006-01-02"
)

var (
	promK8sPolicyWebhookResponseStatusTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "portauthority_api_k8s_image_policy_webhook_response_status",
		Help: "Number of allowed or denied responses recorded.",
	}, []string{"enabled", "namespace", "policy", "allowed"})
)

func init() {
	prometheus.MustRegister(promK8sPolicyWebhookResponseStatusTotal)
}

func decodeJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, maxBodySize)).Decode(v)
}

func writeResponse(w http.ResponseWriter, r *http.Request, status int, resp interface{}) {
	// Headers must be written before the response
	header := w.Header()
	header.Set("Content-Type", "application/json;charset=utf-8")
	header.Set("Server", "portauthority")

	// Gzip the response if the client supports it
	var writer io.Writer = w
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		gzipWriter := gzip.NewWriter(w)
		defer gzipWriter.Close()
		writer = gzipWriter

		header.Set("Content-Encoding", "gzip")
	}

	// Write the response
	w.WriteHeader(status)
	err := json.NewEncoder(writer).Encode(resp)

	if err != nil {
		switch err.(type) {
		case *json.MarshalerError, *json.UnsupportedTypeError, *json.UnsupportedValueError:
			panic("v1: failed to marshal response: " + err.Error())
		default:
			log.WithError(err).Warning("failed to write response")
		}
	}
}

func getContainer(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {
	_, withFeatures := r.URL.Query()["features"]
	_, withVulnerabilities := r.URL.Query()["vulnerabilities"]
	_, withPolicy := r.URL.Query()["policy"]

	policyName := ""
	if withPolicy {
		policyName = r.URL.Query().Get("policy")
	}

	id, _ := strconv.Atoi(p.ByName("id"))
	dbContainer, err := ctx.Store.GetContainerByID(id)
	if err == commonerr.ErrNotFound {
		writeResponse(w, r, http.StatusNotFound, ContainerEnvelope{Error: &Error{err.Error()}})
		return getContainerRoute, http.StatusNotFound
	} else if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, ContainerEnvelope{Error: &Error{err.Error()}})
		return getContainerRoute, http.StatusInternalServerError
	}

	container := ContainerFromDatabaseModel(dbContainer)
	// Now need to get vulns for the image.
	// Only topLayer stored in PA DB needs to be passed to Clair to return list of
	// vulns.
	dbImage, err := ctx.Store.GetImageByDigest(dbContainer.ImageDigest)
	if err == commonerr.ErrNotFound {
		container.ImageScanned = false
		writeResponse(w, r, http.StatusOK, ContainerEnvelope{Container: &container})
		return getContainerRoute, http.StatusOK
	} else if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
		return getContainerRoute, http.StatusInternalServerError
	}
	container.ImageScanned = true
	image := ImageFromDatabaseModel(dbImage)

	if withFeatures || withVulnerabilities {
		// Now need to get vulns for the image.
		// Only topLayer stored in PA DB needs to be passed to Clair to return list
		// of vulns.

		// Using the TopLayer of the Image stored in PA we can the merged view of
		// the features and vulns through Clair
		clairLayerData, err := ctx.ClairClient.GetLayers(dbImage.TopLayer, withFeatures, withVulnerabilities)
		if err != nil {
			log.Warn(fmt.Sprintf("Error getting clair layer data: %s", err))
			writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
			return getContainerRoute, http.StatusInternalServerError
		}

		// Apply policy to Container image
		container.ImageFeatures = clairLayerData.Layer.Features
	}
	if withPolicy {
		policyImage := image
		// Using the TopLayer of the Image stored in PA we can the merged view of
		// the features and vulns through Clair.
		clairLayerData, err := ctx.ClairClient.GetLayers(dbImage.TopLayer, true, true)
		if err != nil {
			log.Warn("error getting layerdata client")
			writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
			return getContainerRoute, http.StatusInternalServerError
		}
		policyImage.Features = clairLayerData.Layer.Features
		dbPolicy, err := ctx.Store.GetPolicy(policyName)
		if err != nil {
			writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
			return getContainerRoute, http.StatusInternalServerError
		}
		if dbPolicy == nil {
			writeResponse(w, r, http.StatusNotFound, ImageEnvelope{Error: &Error{"policy requested is not valid"}})
			return getContainerRoute, http.StatusNotFound
		}

		policy := PolicyFromDatabaseModel(dbPolicy)

		violations, err := getViolations(policy, policyImage)
		if err != nil {
			log.Warn(fmt.Sprintf("Error getting violations: %v", err))
			writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
			return getContainerRoute, http.StatusInternalServerError
		}
		container.ImageViolations = violations
	}

	writeResponse(w, r, http.StatusOK, ContainerEnvelope{Container: &container})
	return getContainerRoute, http.StatusOK
}

func listContainers(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {
	namespaceQueryParm := r.URL.Query().Get("namespace")
	clusterQueryParm := r.URL.Query().Get("cluster")
	nameQueryParm := r.URL.Query().Get("name")
	imageQueryParm := r.URL.Query().Get("image")
	imageIDQueryParm := r.URL.Query().Get("image_id")
	dateStartQueryParm := r.URL.Query().Get("date_start")
	dateEndQueryParm := r.URL.Query().Get("date_end")
	limitQueryParm := r.URL.Query().Get("limit")

	if _, err := time.Parse(dateTimeLayout, dateStartQueryParm); dateStartQueryParm != "" && err != nil {
		writeResponse(w, r, http.StatusBadRequest, ContainersEnvelope{Error: &Error{fmt.Sprintf("Error: %s - Dates must be in the following format %s", err.Error(), dateTimeLayout)}})
		return listContainersRoute, http.StatusBadRequest
	}
	if _, err := time.Parse(dateTimeLayout, dateEndQueryParm); dateEndQueryParm != "" && err != nil {
		writeResponse(w, r, http.StatusBadRequest, ContainersEnvelope{Error: &Error{fmt.Sprintf("Error: %s - Dates must be in the following format %s", err.Error(), dateTimeLayout)}})
		return listContainersRoute, http.StatusBadRequest
	}

	dbContainers, err := ctx.Store.GetAllContainers(namespaceQueryParm, clusterQueryParm, nameQueryParm, imageQueryParm, imageIDQueryParm, dateStartQueryParm, dateEndQueryParm, limitQueryParm)

	if err == commonerr.ErrNotFound {
		writeResponse(w, r, http.StatusNotFound, ContainerEnvelope{Error: &Error{err.Error()}})
		return listContainersRoute, http.StatusNotFound
	} else if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, ContainerEnvelope{Error: &Error{err.Error()}})
		return listContainersRoute, http.StatusInternalServerError
	}

	var containers []*Container
	for _, container := range *dbContainers {
		container := ContainerFromDatabaseModel(container)
		containers = append(containers, &container)
	}

	writeResponse(w, r, http.StatusOK, ContainersEnvelope{Containers: &containers})
	return listContainersRoute, http.StatusOK
}

func getImage(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {
	_, withFeatures := r.URL.Query()["features"]
	_, withVulnerabilities := r.URL.Query()["vulnerabilities"]
	_, withPolicy := r.URL.Query()["policy"]

	policyName := ""
	if withPolicy {
		policyName = r.URL.Query().Get("policy")
	}

	id, _ := strconv.Atoi(p.ByName("id"))
	dbImage, err := ctx.Store.GetImageByID(id)
	if err == commonerr.ErrNotFound {
		writeResponse(w, r, http.StatusNotFound, ImageEnvelope{Error: &Error{err.Error()}})
		return getImageRoute, http.StatusNotFound
	} else if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
		return getImageRoute, http.StatusInternalServerError
	}

	image := ImageFromDatabaseModel(dbImage)

	if withFeatures || withVulnerabilities {
		// Now need to get vulns for the image.
		// Only topLayer stored in PA DB needs to be passed to Clair to return list
		// of vulns.

		// Using the TopLayer of the Image stored in PA we can the merged view of
		// the features and vulns through Clair.
		clairLayerData, err := ctx.ClairClient.GetLayers(dbImage.TopLayer, withFeatures, withVulnerabilities)
		if err != nil {
			log.Warn(fmt.Sprintf("Error: error getting layerdata client: %s", err))
			writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
			return getImageRoute, http.StatusInternalServerError
		}

		// Apply policy to image
		image.Features = clairLayerData.Layer.Features
	}
	if withPolicy {
		policyImage := image
		// Using the TopLayer of the Image stored in PA we can the merged view of
		// the features and vulns through Clair.
		clairLayerData, err := ctx.ClairClient.GetLayers(dbImage.TopLayer, true, true)
		if err != nil {
			log.Warn(fmt.Sprintf("Error: error getting layerdata client: %s", err))
			writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
			return getImageRoute, http.StatusInternalServerError
		}
		policyImage.Features = clairLayerData.Layer.Features
		dbPolicy, err := ctx.Store.GetPolicy(policyName)
		if err != nil {
			writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
			return getImageRoute, http.StatusInternalServerError
		}
		if dbPolicy == nil {
			writeResponse(w, r, http.StatusNotFound, ImageEnvelope{Error: &Error{"policy requested is not valid"}})
			return getImageRoute, http.StatusNotFound
		}

		policy := PolicyFromDatabaseModel(dbPolicy)

		violations, err := getViolations(policy, policyImage)
		if err != nil {
			log.Warn(fmt.Sprintf("Error getting violations: %s", err))
			writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
			return getImageRoute, http.StatusInternalServerError
		}
		image.Violations = violations
	}

	writeResponse(w, r, http.StatusOK, ImageEnvelope{Image: &image})
	return getImageRoute, http.StatusOK
}

// listImages returns all images unless the specified query parameters are
// supplied.
func listImages(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {
	registryQueryParm := r.URL.Query().Get("registry")
	repoQueryParm := r.URL.Query().Get("repo")
	tagQueryParm := r.URL.Query().Get("tag")
	digestQueryParm := r.URL.Query().Get("digest")
	dateStartQueryParm := r.URL.Query().Get("date_start")
	dateEndQueryParm := r.URL.Query().Get("date_end")
	limitQueryParm := r.URL.Query().Get("limit")

	if _, err := time.Parse(dateTimeLayout, dateStartQueryParm); dateStartQueryParm != "" && err != nil {
		writeResponse(w, r, http.StatusBadRequest, ContainersEnvelope{Error: &Error{fmt.Sprintf("Error: %s - Dates must be in the following format %s", err.Error(), dateTimeLayout)}})
		return listContainersRoute, http.StatusBadRequest
	}
	if _, err := time.Parse(dateTimeLayout, dateEndQueryParm); dateEndQueryParm != "" && err != nil {
		writeResponse(w, r, http.StatusBadRequest, ContainersEnvelope{Error: &Error{fmt.Sprintf("Error: %s - Dates must be in the following format %s", err.Error(), dateTimeLayout)}})
		return listContainersRoute, http.StatusBadRequest
	}

	dbImages, err := ctx.Store.GetAllImages(registryQueryParm, repoQueryParm, tagQueryParm, digestQueryParm, dateStartQueryParm, dateEndQueryParm, limitQueryParm)
	if err == commonerr.ErrNotFound {
		writeResponse(w, r, http.StatusNotFound, ImagesEnvelope{Error: &Error{err.Error()}})
		return listImagesRoute, http.StatusNotFound
	} else if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, ImagesEnvelope{Error: &Error{err.Error()}})
		return listImagesRoute, http.StatusInternalServerError
	}

	var images []*Image
	for _, image := range *dbImages {
		image := ImageFromDatabaseModel(image)
		images = append(images, &image)
	}

	writeResponse(w, r, http.StatusOK, ImagesEnvelope{Images: &images})
	return listImagesRoute, http.StatusOK
}

func postImage(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {

	request := ImageEnvelope{}
	var token *docker.Token

	err := decodeJSON(r, &request)
	if err != nil {
		writeResponse(w, r, http.StatusBadRequest, ImageEnvelope{Error: &Error{err.Error()}})
		return postImageRoute, http.StatusBadRequest
	}
	if request.Image == nil {
		writeResponse(w, r, http.StatusBadRequest, ImageEnvelope{Error: &Error{"failed to provide image"}})
		return postImageRoute, http.StatusBadRequest
	}

	// Assume it's public Docker if no registry is supplied
	registryURL := request.Image.Registry
	repo := request.Image.Repo
	if request.Image.Registry == "" || strings.ToLower(request.Image.Registry) == "https://docker.io" {
		registryURL = publicDocker

		if !strings.ContainsAny(request.Image.Repo, "/") {
			repo = fmt.Sprintf("library/%s", request.Image.Repo)
		}
	}

	token, err = docker.AuthRegistry(&docker.AuthConfig{
		RegistryURL: registryURL,
		Username:    request.Image.RegistryUser,
		Password:    request.Image.RegistryPassword,
		Repo:        repo,
		Tag:         request.Image.Tag})

	if err != nil {
		writeResponse(w, r, http.StatusBadRequest, ImageEnvelope{Error: &Error{"error making request to get registry for auth token"}})
		return postImageRoute, http.StatusBadRequest
	}

	// Get Docker registry client
	dockerRegistry, err := docker.GetRegistry(registryURL, request.Image.RegistryUser, request.Image.RegistryPassword)
	if err != nil {
		writeResponse(w, r, http.StatusBadRequest, ImageEnvelope{Error: &Error{"error making initial request to registry for auth"}})
		return postImageRoute, http.StatusBadRequest
	}

	dockerImage, err := docker.GetImage(dockerRegistry, repo, request.Image.Tag)
	if err != nil {
		writeResponse(w, r, http.StatusBadRequest, ImageEnvelope{Error: &Error{err.Error()}})
		return postImageRoute, http.StatusBadRequest
	}

	// Now we scan the image
	dbImage, err := crawler.ScanImage(ctx.Store, ctx.ClairClient, token, dockerImage)
	if err == commonerr.ErrNotFound {
		writeResponse(w, r, http.StatusNotFound, ImageEnvelope{Error: &Error{err.Error()}})
		return postImageRoute, http.StatusNotFound
	} else if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, ImageEnvelope{Error: &Error{err.Error()}})
		return postImageRoute, http.StatusInternalServerError
	}

	image := ImageFromDatabaseModel(dbImage)

	writeResponse(w, r, http.StatusCreated, ImageEnvelope{Image: &image})
	return postImageRoute, http.StatusCreated
}

// listPolicies returns all policies unless the specified query parameters are supplied
func listPolicy(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {

	nameQueryParm := r.URL.Query().Get("name")

	dbPolicies, err := ctx.Store.GetAllPolicies(nameQueryParm)
	if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, PolicyEnvelope{Error: &Error{err.Error()}})
		return listPolicyRoute, http.StatusInternalServerError
	}
	if dbPolicies == nil {
		writeResponse(w, r, http.StatusNotFound, PolicyEnvelope{Error: &Error{"policies requested are not valid"}})
		return listPolicyRoute, http.StatusNotFound
	}

	var policies []*Policy
	for _, policy := range *dbPolicies {
		policy := PolicyFromDatabaseModel(policy)
		policies = append(policies, &policy)
	}

	writeResponse(w, r, http.StatusOK, PoliciesEnvelope{Policies: &policies})
	return listPolicyRoute, http.StatusOK
}

func getPolicy(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {

	dbPolicy, err := ctx.Store.GetPolicy(p.ByName("name"))
	if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, PolicyEnvelope{Error: &Error{err.Error()}})
		return getPolicyRoute, http.StatusInternalServerError
	}
	if dbPolicy == nil {
		writeResponse(w, r, http.StatusNotFound, PolicyEnvelope{Error: &Error{"policy requested is not valid"}})
		return getPolicyRoute, http.StatusNotFound
	}

	policy := PolicyFromDatabaseModel(dbPolicy)

	writeResponse(w, r, http.StatusOK, PolicyEnvelope{Policy: &policy})
	return getPolicyRoute, http.StatusOK
}

func postPolicy(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {

	request := PolicyEnvelope{}

	err := decodeJSON(r, &request)
	if err != nil {
		writeResponse(w, r, http.StatusBadRequest, PolicyEnvelope{Error: &Error{err.Error()}})
		return postPolicyRoute, http.StatusBadRequest
	}
	if request.Policy == nil {
		writeResponse(w, r, http.StatusBadRequest, PolicyEnvelope{Error: &Error{"failed to provide policy"}})
		return postPolicyRoute, http.StatusBadRequest
	}

	// Build DB Policy based on inputs. Probably need some validation here.
	buildPolicy := &datastore.Policy{
		Name:                request.Policy.Name,
		AllowedRiskSeverity: "{" + request.Policy.AllowedRiskSeverity + "}",
		AllowedCVENames:     "{" + request.Policy.AllowedCVENames + "}",
		AllowNotFixed:       request.Policy.AllowNotFixed,
		NotAllowedCveNames:  "{" + request.Policy.NotAllowedCveNames + "}",
		NotAllowedOSNames:   "{" + request.Policy.NotAllowedOSNames + "}",
		Created:             time.Now(),
		Updated:             time.Now(),
	}

	err = ctx.Store.UpsertPolicy(buildPolicy)
	if err == commonerr.ErrNotFound {
		writeResponse(w, r, http.StatusNotFound, PolicyEnvelope{Error: &Error{err.Error()}})
		return postPolicyRoute, http.StatusNotFound
	} else if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, PolicyEnvelope{Error: &Error{err.Error()}})
		return postPolicyRoute, http.StatusInternalServerError
	}

	dbPolicy, err := ctx.Store.GetPolicy(request.Policy.Name)
	if err == commonerr.ErrNotFound {
		writeResponse(w, r, http.StatusNotFound, PolicyEnvelope{Error: &Error{err.Error()}})
		return postPolicyRoute, http.StatusNotFound
	} else if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, PolicyEnvelope{Error: &Error{err.Error()}})
		return postPolicyRoute, http.StatusInternalServerError
	}

	returnPolicy := PolicyFromDatabaseModel(dbPolicy)

	writeResponse(w, r, http.StatusCreated, PolicyEnvelope{Policy: &returnPolicy})
	return postPolicyRoute, http.StatusCreated
}

func getCrawler(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {

	id, _ := strconv.Atoi(p.ByName("id"))
	dbCrawler, err := ctx.Store.GetCrawler(id)
	if err == commonerr.ErrNotFound {
		writeResponse(w, r, http.StatusNotFound, CrawlerEnvelope{Error: &Error{err.Error()}})
		return getCrawlerRoute, http.StatusNotFound
	} else if err != nil {
		writeResponse(w, r, http.StatusInternalServerError, CrawlerEnvelope{Error: &Error{err.Error()}})
		return getCrawlerRoute, http.StatusInternalServerError
	}

	crawler := CrawlerFromDatabaseModel(dbCrawler)

	writeResponse(w, r, http.StatusOK, CrawlerEnvelope{Crawler: &crawler})
	return getCrawlerRoute, http.StatusOK
}

func postCrawler(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {

	var crawlerID int64
	var crawler1 Crawler

	validCrawlerTypes := map[string]bool{
		"registry": true,
		"k8s":      true,
	}

	crawlerType := strings.ToLower(p.ByName("type"))

	// Check if Crawler Type is valid
	if !validCrawlerTypes[crawlerType] {
		writeResponse(w, r, http.StatusBadRequest, CrawlerEnvelope{Error: &Error{fmt.Sprintf("'" + crawlerType + "' is not valid a valid type for /crawler/':type'")}})
		return postCrawlerRoute, http.StatusBadRequest
	}

	if crawlerType == "registry" {
		request := RegCrawlerEnvelope{}

		err := decodeJSON(r, &request)
		if err != nil {
			writeResponse(w, r, http.StatusBadRequest, CrawlerEnvelope{Error: &Error{err.Error()}})
			return postCrawlerRoute, http.StatusBadRequest
		}
		if request.RegCrawler == nil {
			writeResponse(w, r, http.StatusBadRequest, CrawlerEnvelope{Error: &Error{"failed to provide registry crawler"}})
			return postCrawlerRoute, http.StatusBadRequest
		}

		// This is a shared token for the entire crawl of the registry to reduce the
		// number of logins.
		// Public docker is not supported within the Reg Crawler because tokens must
		// be scoped to a repo and tag.
		token, err := docker.AuthRegistry(&docker.AuthConfig{RegistryURL: request.RegCrawler.Registry, Username: request.RegCrawler.Username, Password: request.RegCrawler.Password})
		if err != nil {
			writeResponse(w, r, http.StatusBadRequest, CrawlerEnvelope{Error: &Error{"error making request to get registry for auth token"}})
			return postCrawlerRoute, http.StatusBadRequest
		}

		// Here is where we are going to batch the job and keep track of it from
		// within the crawler.
		crawlerID, err = ctx.Store.InsertCrawler(&datastore.Crawler{
			Type:    crawlerType,
			Status:  "initializing",
			Started: time.Now(),
		})
		if err != nil {
			writeResponse(w, r, http.StatusBadRequest, CrawlerEnvelope{Error: &Error{err.Error()}})
			return postCrawlerRoute, http.StatusBadRequest
		}

		repoFilter := make(map[string]interface{}, len(request.RegCrawler.Repos))
		for _, repo := range request.RegCrawler.Repos {
			repoFilter[repo] = true
		}

		tagFilter := make(map[string]interface{}, len(request.RegCrawler.Tags))
		for _, tags := range request.RegCrawler.Tags {
			tagFilter[tags] = true
		}

		config := &crawler.RegCrawler{
			CrawlerID:   crawlerID,
			Token:       token,
			MaxThreads:  request.RegCrawler.MaxThreads,
			RegistryURL: request.RegCrawler.Registry,
			Repos:       repoFilter,
			Tags:        tagFilter,
			Username:    request.RegCrawler.Username,
			Password:    request.RegCrawler.Password,
		}

		// Go routine runs Crawler in the background and status is maintained within
		// the database.
		go crawler.Registry(ctx.Store, ctx.ClairClient, config)
	}

	if crawlerType == "k8s" {
		request := K8sCrawlerEnvelope{}

		err := decodeJSON(r, &request)
		if err != nil {
			writeResponse(w, r, http.StatusBadRequest, CrawlerEnvelope{Error: &Error{err.Error()}})
			return postCrawlerRoute, http.StatusBadRequest
		}
		if request.K8sCrawler == nil {
			writeResponse(w, r, http.StatusBadRequest, CrawlerEnvelope{Error: &Error{"failed to provide k8s crawler"}})
			return postCrawlerRoute, http.StatusBadRequest
		}

		// Read in Kube Config that was passed in as a base64 encoded string
		sDec, _ := b64.StdEncoding.DecodeString(request.K8sCrawler.KubeConfig)
		apiConfig, err := clientcmd.Load(sDec)
		if err != nil {
			writeResponse(w, r, http.StatusBadRequest, CrawlerEnvelope{Error: &Error{err.Error()}})
			return postCrawlerRoute, http.StatusBadRequest
		}

		// Here is where we are going to batch the job and keep track of it from
		// within the crawler.
		crawlerID, err = ctx.Store.InsertCrawler(&datastore.Crawler{
			Type:    crawlerType,
			Status:  "initializing",
			Started: time.Now(),
		})
		if err != nil {
			writeResponse(w, r, http.StatusBadRequest, CrawlerEnvelope{Error: &Error{err.Error()}})
			return postCrawlerRoute, http.StatusBadRequest
		}

		config := &crawler.K8sCrawler{
			CrawlerID:   crawlerID,
			Store:       ctx.Store,
			KubeConfig:  apiConfig,
			Context:     request.K8sCrawler.Context,
			RegAuth:     ctx.RegAuth,
			ClairClient: ctx.ClairClient,
			Scan:        request.K8sCrawler.Scan,
			MaxThreads:  request.K8sCrawler.MaxThreads,
		}

		// Go routine runs Crawler in the background and status is maintained within
		// the database.
		go crawler.K8s(config)

		// Formatted Scan status to return with request response
		crawler1.Scan = strconv.FormatBool(request.K8sCrawler.Scan)
	}

	if crawlerID != -1 {
		crawler1.ID = uint64(crawlerID)
		crawler1.Type = crawlerType
		crawler1.Started = time.Now()
	}

	writeResponse(w, r, http.StatusCreated, CrawlerEnvelope{Crawler: &crawler1})
	return postCrawlerRoute, http.StatusCreated
}

// The postK8sImagePolicy function is a webhook intended for the use of K8's
// clusters running an ImagePolicyWebhook admission controller.
func postK8sImagePolicy(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {

	request := K8sImagePolicy{}

	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		log.Error(fmt.Sprintf("Request Error: %s", err))
	}
	log.Debug(string(requestDump))

	err = decodeJSON(r, &request)

	if err != nil {
		log.Error(fmt.Sprintf("Request: %s, Error: %v", requestDump, err))
		writeResponse(w, r, http.StatusBadRequest, K8sImagePolicyEnvelope{Error: &Error{err.Error()}})
		return postK8sImagePolicyRoute, http.StatusBadRequest
	}

	k8sImagePolicyEnvelope := K8sImagePolicyEnvelope{K8sImagePolicy: &request}

	if k8sImagePolicyEnvelope.K8sImagePolicy == nil {
		log.Error(fmt.Sprintf("Failed to provide k8s policy"))
		writeResponse(w, r, http.StatusBadRequest, K8sImagePolicyEnvelope{Error: &Error{"Failed to provide k8s policy"}})
		return postK8sImagePolicyRoute, http.StatusBadRequest
	}

	// User will be able to override the default behavior of the webhook set in
	// the config.yml.
	var enabledImageWebhook bool
	userEnabledImageWebHook, ok := request.Spec.Annotations["alpha.image-policy.k8s.io/portauthority-webhook-enable"]
	if ok {
		var userEnabledImageWebHookBool bool
		userEnabledImageWebHookBool, err = strconv.ParseBool(userEnabledImageWebHook)
		if err != nil {
			log.Error(fmt.Sprintf("Improperly formated portauthority-webhook user annotation %s: %s", userEnabledImageWebHook, err))
			writeResponse(w, r, http.StatusBadRequest, K8sImagePolicyEnvelope{Error: &Error{"Improperly formated portauthority-webhook user annotation"}})
			return postK8sImagePolicyRoute, http.StatusBadRequest
		}
		if userEnabledImageWebHookBool == false {
			writeResponse(w, r, http.StatusOK, &K8sImagePolicy{
				APIVersion: request.APIVersion,
				Kind:       request.Kind,
				Status: &ImageReviewStatus{
					Allowed: true,
					Reason:  "User disabled image webhook",
				},
				Spec: &K8sImageSpec{
					Namespace: request.Spec.Namespace,
				},
			})
			promK8sPolicyWebhookResponseStatusTotal.WithLabelValues("false", request.Spec.Namespace, "", "true").Inc()
			return postK8sImagePolicyRoute, http.StatusOK
		}
		enabledImageWebhook = userEnabledImageWebHookBool
	} else {
		enabledImageWebhook = ctx.ImageWebhookDefaultBlock
	}

	log.Debug(fmt.Sprintf("Image Webhook Enabled: %v", enabledImageWebhook))
	if enabledImageWebhook == false {
		writeResponse(w, r, http.StatusOK, &K8sImagePolicy{
			APIVersion: request.APIVersion,
			Kind:       request.Kind,
			Status: &ImageReviewStatus{
				Allowed: true,
				Reason:  "Image webhook disabled by an administrator",
			},
			Spec: &K8sImageSpec{
				Namespace: request.Spec.Namespace,
			},
		})
		promK8sPolicyWebhookResponseStatusTotal.WithLabelValues("false", request.Spec.Namespace, "", "true").Inc()
		return postK8sImagePolicyRoute, http.StatusOK
	}

	// Create default image policy
	imageReviewPolicy := &K8sImagePolicy{
		APIVersion: request.APIVersion,
		Kind:       request.Kind,
		Status: &ImageReviewStatus{
			Allowed: true,
		},
		Spec: &K8sImageSpec{
			Namespace: request.Spec.Namespace,
		},
	}

	// Check to see if the deployment supplied any containers
	if len(request.Spec.Containers) < 1 {
		writeResponse(w, r, http.StatusOK, &K8sImagePolicy{
			APIVersion: request.APIVersion,
			Kind:       request.Kind,
			Status: &ImageReviewStatus{
				Allowed: false,
				Reason:  "Invalid number of container images supplied",
			},
			Spec: &K8sImageSpec{
				Namespace: request.Spec.Namespace,
			},
		})
		promK8sPolicyWebhookResponseStatusTotal.WithLabelValues("true", request.Spec.Namespace, "", "false").Inc()
		return postK8sImagePolicyRoute, http.StatusOK
	}

	// Use Default Policy if none was supplied in the annotations
	policyName, ok := request.Spec.Annotations["alpha.image-policy.k8s.io/policy"]
	if !ok {
		policyName = "default"
	}

	dbPolicy, err := ctx.Store.GetPolicy(policyName)
	if err != nil {
		log.Error(fmt.Sprintf("An error occured looking looking up the policy \"%s\" in the database: %s", policyName, err))
		writeResponse(w, r, http.StatusInternalServerError, K8sImagePolicyEnvelope{Error: &Error{err.Error()}})
		return postK8sImagePolicyRoute, http.StatusInternalServerError
	}

	// If the user supplied a policy that doesn't exist in the database, then we
	// will not allow the deployment to pass.
	if dbPolicy == nil {
		log.Debug(fmt.Sprintf("policy requested could not found: %s", policyName))
		writeResponse(w, r, http.StatusOK, &K8sImagePolicy{
			APIVersion: request.APIVersion,
			Kind:       request.Kind,
			Status: &ImageReviewStatus{
				Allowed: false,
				Reason:  fmt.Sprintf("policy requested could not be found: %s", policyName),
			},
			Spec: &K8sImageSpec{
				Namespace: request.Spec.Namespace,
			},
		})
		promK8sPolicyWebhookResponseStatusTotal.WithLabelValues("true", request.Spec.Namespace, policyName, "false").Inc()
		return postK8sImagePolicyRoute, http.StatusOK
	}

	policy := PolicyFromDatabaseModel(dbPolicy)

	// Loop through each of the containers within the deployment.
	// If any contain vulnerabilites, the loop will break and return the container
	// with vulns.
	for c := range request.Spec.Containers {

		dbImage := &datastore.Image{}

		registry, repo, tag, digest, _ := crawler.ParseImageTagPath(request.Spec.Containers[c].Image)

		if digest != "" {
			dbImage, err = ctx.Store.GetImageByDigest(digest)
			if err == commonerr.ErrNotFound {
				imageReviewPolicy.Status.Allowed = false
				imageReviewPolicy.Status.Reason = fmt.Sprintf("Image Not Scanned: %s", request.Spec.Containers[c].Image)
				log.Debug(imageReviewPolicy.Status.Reason)
				break
			} else if err != nil {
				log.Error(fmt.Sprintf("%v", err))
				writeResponse(w, r, http.StatusInternalServerError, K8sImagePolicyEnvelope{Error: &Error{err.Error()}})
				return postK8sImagePolicyRoute, http.StatusInternalServerError
			}
		} else {

			dbImage, err = ctx.Store.GetImageByRrt(registry, repo, tag)
			if err == commonerr.ErrNotFound {
				imageReviewPolicy.Status.Allowed = false
				imageReviewPolicy.Status.Reason = fmt.Sprintf("Image Not Scanned: %s", request.Spec.Containers[c].Image)
				log.Debug(imageReviewPolicy.Status.Reason)
				break
			} else if err != nil {
				log.Error(fmt.Sprintf("%v", err))
				writeResponse(w, r, http.StatusInternalServerError, K8sImagePolicyEnvelope{Error: &Error{err.Error()}})
				return postK8sImagePolicyRoute, http.StatusInternalServerError
			}
		}
		image := ImageFromDatabaseModel(dbImage)

		// Using the TopLayer of the Image stored in PA we can the merged view of the features and vulns through Clair
		clairLayerData, err := ctx.ClairClient.GetLayers(dbImage.TopLayer, true, true)
		if err != nil {
			log.Error(fmt.Sprintf("Error getting layerdata: %v", err))
			writeResponse(w, r, http.StatusInternalServerError, K8sImagePolicyEnvelope{Error: &Error{err.Error()}})
			return postK8sImagePolicyRoute, http.StatusInternalServerError
		}

		// Apply policy to image
		image.Features = clairLayerData.Layer.Features

		// Get Violations for the image
		violations, err := getViolations(policy, image)
		if err != nil {
			log.Error(fmt.Sprintf("Error getting Violoations: %v", err))
			writeResponse(w, r, http.StatusInternalServerError, K8sImagePolicyEnvelope{Error: &Error{err.Error()}})
			return postK8sImagePolicyRoute, http.StatusInternalServerError
		}
		image.Violations = violations

		if len(image.Violations) > 0 {
			imageReviewPolicy.Status.Allowed = false
			imageReviewPolicy.Status.Reason = fmt.Sprintf("Scan policy \"%s\" detected \"%d\" violations for image: %s", policyName, len(image.Violations), request.Spec.Containers[c].Image)
			log.Debug(imageReviewPolicy.Status.Reason)
			break
		}
	}

	promK8sPolicyWebhookResponseStatusTotal.WithLabelValues("true", imageReviewPolicy.Spec.Namespace, policy.Name, strconv.FormatBool(imageReviewPolicy.Status.Allowed)).Inc()
	writeResponse(w, r, http.StatusOK, imageReviewPolicy)
	return postK8sImagePolicyRoute, http.StatusOK
}

func getMetrics(w http.ResponseWriter, r *http.Request, p httprouter.Params, ctx *context) (string, int) {
	prometheus.Handler().ServeHTTP(w, r)
	return getMetricsRoute, 0
}

// The getViolations function returns a list of violations based on the supplied
// policy.
func getViolations(policy Policy, image Image) ([]Violation, error) {

	var violations []Violation

	// Check for violations in order of precedence
	// 1.	OS is not approved
	// 2.	Is in CVE blacklist
	// 3.	Is not in CVE whitelist
	// 4.	Does not have an approved severity (e.g. low, unknown, negligible)
	// 5.	Has fix available

FeatureLoop:
	for f := range image.Features {
		vulnerabilities := image.Features[f].Vulnerabilities

		var notAllowedOSNames []string
		err := json.Unmarshal([]byte(policy.NotAllowedOSNames), &notAllowedOSNames)
		if err != nil {
			return nil, err
		}
		for o := range notAllowedOSNames {
			if notAllowedOSNames[o] == image.Features[f].NamespaceName {
				// OS is blacklisted, add violation
				violations = append(violations, Violation{
					Type: BlacklistedOsViolation,
				})
				continue FeatureLoop
			}
		}

	VulnLoop:
		for v := range vulnerabilities {

			var notAllowedCVEs []string
			err = json.Unmarshal([]byte(policy.NotAllowedCveNames), &notAllowedCVEs)
			if err != nil {
				return nil, err
			}
			for n := range notAllowedCVEs {
				if notAllowedCVEs[n] == vulnerabilities[v].Name {
					// CVE is in blacklist, add violation
					violations = append(violations, Violation{
						Type:           BlacklistedCveViolation,
						FeatureName:    image.Features[f].Name,
						FeatureVersion: image.Features[f].Version,
						Vulnerability:  vulnerabilities[v],
					})
					continue VulnLoop
				}
			}

			var allowedCVEs []string
			err = json.Unmarshal([]byte(policy.AllowedCVENames), &allowedCVEs)
			if err != nil {
				return nil, err
			}
			for c := range allowedCVEs {
				if allowedCVEs[c] == vulnerabilities[v].Name {
					// CVE is whitelisted, continue
					continue VulnLoop
				}
			}

			var severities []string
			err = json.Unmarshal([]byte(policy.AllowedRiskSeverity), &severities)
			if err != nil {
				return nil, err
			}
			for s := range severities {
				if severities[s] == vulnerabilities[v].Severity {
					// Severity is allowed, continue
					continue VulnLoop
				}
			}

			if policy.AllowNotFixed == true && len(vulnerabilities[v].FixedBy) == 0 {
				// CVE has no fix, continue
				continue VulnLoop
			}

			// Catch all
			violations = append(violations, Violation{
				Type:           BasicViolation,
				FeatureName:    image.Features[f].Name,
				FeatureVersion: image.Features[f].Version,
				Vulnerability:  vulnerabilities[v],
			})

		}
	}
	return violations, nil
}
