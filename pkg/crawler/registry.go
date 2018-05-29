// Copyright (c) 2018 Target Brands, Inc.

package crawler

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	"github.com/target/portauthority/pkg/clair"
	"github.com/target/portauthority/pkg/clair/client"
	"github.com/target/portauthority/pkg/datastore"
	"github.com/target/portauthority/pkg/docker"
)

// RegCrawler configuration struct
type RegCrawler struct {
	CrawlerID   int64
	MaxThreads  uint
	Username    string
	Password    string
	RegistryURL string
	Token       *docker.Token
	Repos       map[string]interface{} `json:"repos"`
	Tags        map[string]interface{} `json:"tags"`
}

// Registry crawler gets a list of images including their manifests and layers
// from a defined Docker V2 registry.
// Then a semaphore channel is opened and the images are fed to the Clair
// instances for scanning.
func Registry(backend datastore.Backend, cc clairclient.Client, regCrawler *RegCrawler) {
	start := time.Now()
	// Log the start of the crawl in the database
	err := backend.UpdateCrawler(regCrawler.CrawlerID, &datastore.Crawler{
		Status: "started",
	})

	if err != nil {
		log.Error(err, "could not update crawler in db")
		return
	}

	// Open a channel for collecting Docker images
	dockerImages := make(chan *docker.ImageEnvelope, 50)

	var totalDockerImages uint64
	var failedScan uint64

	go docker.Crawl(docker.CrawlConfig{
		URL:      regCrawler.RegistryURL,
		Username: regCrawler.Username,
		Password: regCrawler.Password,
		Repos:    regCrawler.Repos,
		Tags:     regCrawler.Tags,
	}, dockerImages)

	// A blocking channel to keep concurrency under control
	sem := make(chan interface{}, regCrawler.MaxThreads)
	defer close(sem)

	wg := &sync.WaitGroup{}
	var regError error

	for image := range dockerImages {
		sem <- true // This will block if the semaphore is full
		wg.Add(1)

		if image.Error != nil {
			regError = image.Error
			wg.Done()
			break
		}

		go func(image *docker.ImageEnvelope) {
			defer func() { <-sem }() // Release hold on one of the semaphore items
			_, err = ScanImage(backend, cc, regCrawler.Token, &image.Image)

			atomic.AddUint64(&totalDockerImages, 1)
			if err != nil {
				log.Error(fmt.Sprintf("Crawl Scan Error for Image %s/%s:%s: %s", image.Registry, image.Repo, image.Tag, err.Error()))
				atomic.AddUint64(&failedScan, 1)
			}
			// Tell the wait group that this scan is done
			wg.Done()
		}(image)
	}

	// Wait for all the goroutines to be done
	wg.Wait()
	ti := atomic.LoadUint64(&totalDockerImages)
	fs := atomic.LoadUint64(&failedScan)
	elapsed := time.Since(start)
	if regError != nil {
		err = backend.UpdateCrawler(regCrawler.CrawlerID, &datastore.Crawler{
			Status: "finished",
			Messages: &datastore.CrawlerMessages{
				Error: fmt.Sprintf("** Crawl of %s produced error: %s **", regCrawler.RegistryURL, regError.Error())},
			Finished: time.Now(),
		})
		if err != nil {
			log.Error(err, "could not update crawler in db")
			return
		}
		log.Error(fmt.Sprintf("** Crawl of %s produced error: %s **", regCrawler.RegistryURL, regError))
	} else {
		err = backend.UpdateCrawler(regCrawler.CrawlerID, &datastore.Crawler{
			Status: "finished",
			Messages: &datastore.CrawlerMessages{
				Summary: fmt.Sprintf("** %d images in %s processed in %s with %d scan failures **", ti, regCrawler.RegistryURL, elapsed, fs)},
			Finished: time.Now(),
		})
		if err != nil {
			log.Error(err, "could not update crawler in db")
			return
		}
		log.Info(fmt.Sprintf("Registry crawl #%d in %s of %d images completed in %s with %d scan failures **", regCrawler.CrawlerID, regCrawler.RegistryURL, ti, elapsed, fs))

	}

}

// ScanImage sends a single image to Clair
func ScanImage(db datastore.Backend, cc clairclient.Client, token *docker.Token, image *docker.Image) (*datastore.Image, error) {
	// Make another call to the DB to get the image ID
	dbImage, err := db.GetImage(image.Registry, image.Repo, image.Tag, image.Digest)
	if err != nil {
		return nil, errors.Wrap(err, "error looking up image in database")
	}

	// Get the TopLayer and store in the PA DB
	topLayerHash := ""
	if len(image.Layers) > 0 {
		topLayer := strings.Join([]string{image.Digest, string(image.Layers[0])}, "")
		topLayerHash = GetMD5Hash(topLayer)
	}

	if dbImage == nil {
		dbImage = &datastore.Image{
			TopLayer:   topLayerHash,
			Registry:   image.Registry,
			Repo:       image.Repo,
			Tag:        image.Tag,
			Digest:     image.Digest,
			ManifestV2: image.ManifestV2,
			ManifestV1: image.ManifestV1,
			Metadata:   image.Metadata,
			FirstSeen:  time.Now(),
			LastSeen:   time.Now(),
		}
	} else {
		dbImage.LastSeen = time.Now()
	}

	err = db.UpsertImage(dbImage)
	if err != nil {
		log.Error(fmt.Sprintf("error updating image %s/%s:%s: %+v", image.Registry, image.Repo, image.Tag, err))
		return nil, errors.Wrap(err, "error updating image")
	}

	log.Debug("updated image ", image.Digest)

	dbImage, err = db.GetImage(image.Registry, image.Repo, image.Tag, image.Digest)
	if err != nil {
		return nil, errors.Wrap(err, "error getting id after image insert into the database")
	}

	bearerToken := ""
	if token.Token != "" {
		bearerToken = strings.Join([]string{"Bearer", token.Token}, " ")
	}

	err = clair.Push(&cc, clair.Image{
		Digest:   image.Digest,
		Registry: image.Registry,
		Repo:     image.Repo,
		Layers:   image.Layers,
		Headers:  map[string]string{"Authorization": bearerToken},
	})

	if err != nil {
		return nil, errors.Wrap(err, "Error while pushing image to Clair")
	}

	log.Debug("Clair finished scanning layers:", image.Digest)

	return dbImage, err
}

// GetMD5Hash returns a MD5 hash of a string
func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
