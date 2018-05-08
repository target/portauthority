// Copyright (c) 2018 Target Brands, Inc.

package docker

import (
	"encoding/json"
	"fmt"

	log "github.com/Sirupsen/logrus"

	"github.com/pkg/errors"
	"github.com/target/portauthority/pkg/docker/registry"
)

// ImageEnvelope struct init
type ImageEnvelope struct {
	Image
	Error error `json:"error"`
}

// Image struct init
type Image struct {
	Registry   string   `json:"registry"`
	Repo       string   `json:"repo"`
	Tag        string   `json:"tag"`
	Digest     string   `json:"digest"`
	Layers     []string `json:"layers"`
	ManifestV2 string   `json:"manifest"`
	ManifestV1 string   `json:"manifestv1"`
}

// CrawlConfig struct init
type CrawlConfig struct {
	URL      string                 `json:"url"`
	Username string                 `json:"username"`
	Password string                 `json:"password"`
	Repos    map[string]interface{} `json:"repos"`
	Tags     map[string]interface{} `json:"tags"`
}

const emptyLayer = "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"

// Crawl will search a registry for Docker images.
// Search can be filtered in the crawl Config.
// Will return results on images chan; will close images before returning.
// Returns error when it cannot proceed.
// Errors encountered getting image manifests are communicated in the
// ImageEnvelope.
func Crawl(conf CrawlConfig, images chan *ImageEnvelope) {
	// Open connection with the registry
	hub, err := registry.New(conf.URL, conf.Username, conf.Password)
	if err != nil {
		images <- &ImageEnvelope{Error: errors.Wrapf(err, "error connecting to registry: %s", conf.URL)}
		close(images)
		return
	}

	// List all the repos in the registry
	repos, err := hub.Repositories()
	if err != nil {
		images <- &ImageEnvelope{Error: errors.Wrapf(err, "error listing repositories for %s", conf.URL)}
		close(images)
		return
	}

	log.Debug("%v", len(conf.Repos))
	log.Debug("%v", conf.Repos)

	for _, repo := range repos {

		if _, ok := conf.Repos[repo]; ok || len(conf.Repos) == 0 { // Proceed if map empty or if this repo is in map
			// List all the tags in the repository
			tags, err := hub.Tags(repo)
			if err != nil {
				images <- &ImageEnvelope{Error: errors.Wrapf(err, "error listing tags for %s/%s", conf.URL, repo)}
				close(images)
				return
			}

			for _, tag := range tags {
				if _, ok := conf.Tags[tag]; ok || len(conf.Tags) == 0 { // Proceed if map empty or if this tag is in map

					image, err := GetImage(hub, repo, tag)
					if err != nil {
						log.Error(err, "error getting image: %s/%s:%s", hub.URL, repo, tag)
						continue
					}

					images <- &ImageEnvelope{
						Image: Image{
							Registry:   image.Registry,
							Repo:       image.Repo,
							Tag:        image.Tag,
							Digest:     image.Digest,
							Layers:     image.Layers,
							ManifestV2: image.ManifestV2,
							ManifestV1: image.ManifestV1,
						},
					}
				}
			}
		}
	}
	close(images)
}

// GetRegistry returns a registry object that can be used later
func GetRegistry(registryURL string, username string, password string) (*registry.Registry, error) {
	hub, err := registry.New(registryURL, username, password)
	if err != nil {
		return hub, errors.Wrap(err, "error initializing registry")
	}
	return hub, nil
}

// GetImage returns a Docker Image containing V1 and V2 manifests, its layers,
// and location information.
func GetImage(hub *registry.Registry, repo string, tag string) (*Image, error) {
	// Default maniftest will be a v2
	digest, err := hub.ManifestDigestV2(repo, tag)
	if err != nil {
		log.Debug(fmt.Sprintf("Error getting v2 content digest: %s", err))
		// Attempt to obtain v1 if v2 is unavailable
		digest, err = hub.ManifestDigest(repo, tag)
		if err != nil {
			return nil, fmt.Errorf("Unable to obtain either v1 or v2 digest: %s", err)
		}
	}

	// Both V1 and V2 manifests contain useful data we want to store
	var layers []string
	var marshaledManifestV2 []byte
	manifest, err := hub.ManifestV2(repo, tag)
	if err != nil {
		log.Debug(fmt.Sprintf("Error getting v2 manifest: %s for Image %s/%s:%s", err, hub.URL, repo, tag))
	} else {

		// Will use v2 manifest to build layers if its availble.
		// V1 and V2 layer order is reversed.
		for i := len(manifest.Layers) - 1; i >= 0; i-- {
			if string(manifest.Layers[i].Digest) != emptyLayer {
				layers = append(layers, string(manifest.Layers[i].Digest))
			}
		}

		// Format V2 Manifest into JSON for easy db storage
		marshaledManifestV2, err = json.Marshal(manifest)
		if err != nil {
			log.Debug(fmt.Sprintf("Error parsing v2 manifest: %s/%s:%s", hub.URL, repo, tag))
		}
	}

	var marshaledManifestV1 []byte
	manifestV1, err := hub.Manifest(repo, tag)
	if err != nil {
		log.Debug(fmt.Sprintf("Error getting v1 manifest: %s for Image %s/%s:%s", err, hub.URL, repo, tag))
	} else {

		// If layers from V1 aren't available attempt to use the V1.
		// V1 and V2 layer order is reversed.
		if len(layers) == 0 {
			for i := 0; i <= len(manifestV1.FSLayers)-1; i++ {
				if string(manifestV1.FSLayers[i].BlobSum) != emptyLayer {
					layers = append(layers, string(manifestV1.FSLayers[i].BlobSum))
				}
			}
		}

		// Format V1 Manifest into JSON for easy db storage
		marshaledManifestV1, err = json.Marshal(manifestV1)
		if err != nil {
			log.Debug(fmt.Sprintf("Error parsing v1 manifest: %s/%s:%s", hub.URL, repo, tag))
		}
	}

	if err != nil && manifest == nil && manifestV1 == nil {
		return nil, fmt.Errorf("Docker V1 or V2 could be obtained: %s", err)
	}

	if len(layers) == 0 {
		return nil, fmt.Errorf("Image manifest contaied no layers: %s/%s:%s", hub.URL, repo, tag)
	}

	image := &Image{
		Registry:   hub.URL,
		Repo:       repo,
		Tag:        tag,
		Digest:     string(digest),
		ManifestV1: string(marshaledManifestV1),
		ManifestV2: string(marshaledManifestV2),
		Layers:     layers,
	}
	return image, nil
}
