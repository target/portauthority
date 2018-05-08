// Copyright (c) 2018 Target Brands, Inc.

package crawler

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
	"github.com/target/portauthority/pkg/clair/client"
	"github.com/target/portauthority/pkg/datastore"
	"github.com/target/portauthority/pkg/docker"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// K8sCrawler struct init
type K8sCrawler struct {
	CrawlerID   int64
	Store       datastore.Backend
	Context     string
	KubeConfig  *clientcmdapi.Config
	MaxThreads  uint
	RegAuth     []map[string]string
	ClairClient clairclient.Client
	Scan        bool
}

// K8s func init
func K8s(c *K8sCrawler) {
	start := time.Now()
	err := c.Store.UpdateCrawler(c.CrawlerID, &datastore.Crawler{
		Status: "started",
	})
	if err != nil {
		log.Error(fmt.Sprintf("Unable to update DB status: %s", err))
		return
	}

	totalContainerImages, clusterHost, err := c.GetContainerImages()
	if err != nil {
		err = c.Store.UpdateCrawler(c.CrawlerID, &datastore.Crawler{
			Status: "error",
			Messages: &datastore.CrawlerMessages{
				Error: fmt.Sprintf("error getting container images: %s", err)},
			Finished: time.Now(),
		})
		if err != nil {
			log.Error(fmt.Sprintf("error updating crawler in db: %s", err))
			return
		}
		log.Error(fmt.Sprintf("error getting container images: %s", err))
		return
	}

	if c.Scan == true {

		// Setup counters to record scan summary details
		var scannedSuccess uint64
		var scannedFailed uint64
		var notScanned uint64

		log.Debug("Scanning Enabled. Begin gathering Docker images.")

		// Remove duplicate container images with the same content hash to prevent
		// repulling of the same image.
		dedupedContainerImages := removeDuplicates(totalContainerImages)

		// Update db status
		err = c.Store.UpdateCrawler(c.CrawlerID, &datastore.Crawler{
			Status: "scanning containers",
		})
		if err != nil {
			log.Error("Unable to update DB status")
			return
		}

		// A blocking semaphore channel to keep concurrency under control
		if c.MaxThreads == 0 {
			c.MaxThreads = 10
		}
		sem := make(chan interface{}, c.MaxThreads)
		defer close(sem)

		// A wait group enables the main process a wait for goroutines to finish
		wg := &sync.WaitGroup{}

		for _, container := range dedupedContainerImages {
			sem <- true // this will block if the semaphore is full
			wg.Add(1)

			go func(container *datastore.Container) {
				defer func() { <-sem }() // release hold on one of the semaphore items

				user, pass, match := c.GetStoredCredentials(container)

				if match == true {

					err = c.ScanContainerImage(container, user, pass)
					if err != nil {
						log.Errorf("K8s Crawler Image Scan: %v/%v:%v --- Unable to Send the Image to Clair, the error is: %v", container.ImageRegistry, container.ImageRepo, container.ImageTag, err)
						atomic.AddUint64(&scannedFailed, 1)
					} else {
						atomic.AddUint64(&scannedSuccess, 1)
					}
				} else {
					log.Debug(fmt.Sprintf("K8s Crawler Image Scan: No Creds Supplied for Registry -- %s -- In Repo %s", container.ImageRegistry, container.ImageRepo))
					atomic.AddUint64(&notScanned, 1)
				}

				// Tell the wait group that this scan is done
				wg.Done()

			}(container)
		}

		// Wait for all the goroutines to be done
		wg.Wait()

		ss := atomic.LoadUint64(&scannedSuccess)
		sf := atomic.LoadUint64(&scannedFailed)
		ns := atomic.LoadUint64(&notScanned)
		elapsed := time.Since(start)

		err = c.Store.UpdateCrawler(c.CrawlerID, &datastore.Crawler{
			Status: "finished",
			Messages: &datastore.CrawlerMessages{
				Summary: fmt.Sprintf("** %d images in %s processed in %s ** Scan Details: %d Successful -- %d Failed -- %d Skipped", len(dedupedContainerImages), clusterHost, elapsed, ss, sf, ns)},
			Finished: time.Now(),
		})
		if err != nil {
			log.Error(fmt.Sprintf("Unable to update DB status: %s", err))
			return
		}
		log.Debug(fmt.Sprintf("Duplicate Removal Summary: %d Total Containers -- %d Duplicates Purged -- %d Resulting Unique Containers ", len(totalContainerImages), len(totalContainerImages)-len(dedupedContainerImages), len(dedupedContainerImages)))

		log.Info(fmt.Sprintf("** K8s crawl #%d in %s of %d images completed in %s ** Scan Details: %d Successful -- %d Failed -- %d Skipped", c.CrawlerID, clusterHost, len(dedupedContainerImages), elapsed, ss, sf, ns))
	} else {

		elapsed := time.Since(start)
		err = c.Store.UpdateCrawler(c.CrawlerID, &datastore.Crawler{
			Status: "finished",
			Messages: &datastore.CrawlerMessages{
				Summary: fmt.Sprintf("** %d total images in %s processed in %s **", len(totalContainerImages), clusterHost, elapsed)},
			Finished: time.Now(),
		})

		if err != nil {
			log.Error(fmt.Sprintf("Unable to update DB status: %s", err))
			return
		}
		log.Info(fmt.Sprintf("** K8s crawl #%d in %s of %d total images completed in %s **", c.CrawlerID, clusterHost, len(totalContainerImages), elapsed))

	}

}

// GetContainerImages func init
func (c *K8sCrawler) GetContainerImages() ([]*datastore.Container, string, error) {

	var containers []*datastore.Container

	// Build the config with overrides
	config, err := clientcmd.NewDefaultClientConfig(
		*c.KubeConfig,
		&clientcmd.ConfigOverrides{
			ClusterInfo:    clientcmdapi.Cluster{Server: ""},
			CurrentContext: c.Context,
		}).ClientConfig()

	if err != nil {
		log.Error("error loading kubeconfig: ", err)
		return nil, "", err
	}

	// Set cluster host to be passed into db
	clusterHost := config.Host

	// Create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error("error creating clientset: ", err)
		return nil, clusterHost, err
	}

	// Get all namespaces
	namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		log.Error("error getting namespaces: ", err)
		return nil, clusterHost, err
	}

	// Update db status
	err = c.Store.UpdateCrawler(c.CrawlerID, &datastore.Crawler{
		Status: "getting containers",
	})
	if err != nil {
		log.Error(fmt.Sprintf("Unable to update DB status: %s", err))
		return nil, clusterHost, err
	}

	// Loop through list
	namespacesList := namespaces.Items
	for _, namespace := range namespacesList {

		pods, err := clientset.CoreV1().Pods(namespace.Name).List(metav1.ListOptions{})

		if err != nil {
			log.Error("error getting pods: ", err)
			return nil, clusterHost, err
		}

		log.Debug("There are ", len(pods.Items), " pods in the cluster")

		ns, err := clientset.CoreV1().Namespaces().Get(namespace.Name, metav1.GetOptions{})
		if err != nil {
			log.Error("\n", err.Error())
			return nil, clusterHost, err
		}

		annotations := datastore.AnnotationMap{}
		var JSONTemp interface{}
		for key, value := range ns.Annotations {
			err = json.Unmarshal([]byte(value), &JSONTemp)
			if err == nil {
				annotations[key] = JSONTemp
			} else {
				annotations[key] = value
			}
		}

		// Setup list of pods from each namespace and record unique container records in the database
		podList := pods.Items
		for _, kp := range podList {
			for _, kc := range kp.Status.ContainerStatuses {
				containerName := kc.Name
				containerImage := kc.Image
				containerImageID := kc.ImageID

				containerImageRegistry, containerImageRepo, containerImageTag, _, _ := ParseImageTagPath(kc.Image)
				containerImageDigest, _ := ParseImageDigest(kc.ImageID)

				log.Debug("Container Registry: ", containerImageRegistry)
				log.Debug("Container Repo: ", containerImageRepo)
				log.Debug("Container Tag: ", containerImageTag)
				log.Debug("Container Digest: ", containerImageDigest)
				log.Debug("Container Cluster: ", clusterHost)

				// Check to see if the container exists first
				dbContainer, err := c.Store.GetContainer(namespace.Name, clusterHost, containerName, containerImage, containerImageID)
				if err != nil {
					log.Error("error getting container from database: ", err)
					return nil, clusterHost, err
				}

				if dbContainer == nil {
					dbContainer = &datastore.Container{
						Namespace:     namespace.Name,
						Cluster:       clusterHost,
						Name:          containerName,
						Image:         containerImage,
						ImageID:       containerImageID,
						ImageRegistry: containerImageRegistry,
						ImageRepo:     containerImageRepo,
						ImageTag:      containerImageTag,
						ImageDigest:   containerImageDigest,
						Annotations:   annotations,
						FirstSeen:     time.Now(),
						LastSeen:      time.Now(),
					}
				} else {
					dbContainer.LastSeen = time.Now()
					dbContainer.Annotations = annotations
				}

				// Upsert the Container Info into Postgres
				err = c.Store.UpsertContainer(dbContainer)
				if err != nil {
					log.Error("error upserting up container in database: ", namespace.Name, "-", clusterHost, "-", containerName, "-", containerImage, "-", containerImageID, " : ", err)
					return nil, clusterHost, err
				}
				log.Debug(fmt.Sprintf("Container updated: %s %s", containerImage, containerImageID))

				containers = append(containers, dbContainer)
			}
		}
	}

	log.Info("Finished Getting Containers for Cluster: ", clusterHost)
	return containers, clusterHost, nil
}

//ScanContainerImage func init
func (c *K8sCrawler) ScanContainerImage(dc *datastore.Container, user, pass string) error {
	token, err := docker.AuthRegistry(&docker.AuthConfig{RegistryURL: dc.ImageRegistry, Username: user, Password: pass, Repo: dc.ImageRepo, Tag: dc.ImageTag})
	if err != nil {
		return err
	}

	dockerRegistry, err := docker.GetRegistry(dc.ImageRegistry, user, pass)
	if err != nil {
		return err
	}

	dockerImage, err := docker.GetImage(dockerRegistry, dc.ImageRepo, dc.ImageTag)
	if err != nil {
		return err
	}
	_, err = ScanImage(c.Store, c.ClairClient, token, dockerImage)
	if err != nil {
		return err
	}
	log.Debug("K8s Crawler Image Scan Finished Scan: ", dockerImage.Repo)
	return nil
}

//GetStoredCredentials func init
func (c *K8sCrawler) GetStoredCredentials(dc *datastore.Container) (username, password string, match bool) {
	// Format URL & Get UN/PW
	if strings.Contains(dc.ImageID, "docker-pullable") && (dc.ImageRegistry == "" || strings.Contains(dc.ImageRegistry, "docker.io")) {
		dc.ImageRegistry = "https://registry-1.docker.io"
		if !strings.ContainsAny(dc.ImageRepo, "/") {
			dc.ImageRepo = fmt.Sprintf("library/%s", dc.ImageRepo)
		}
	} else if strings.Contains(dc.ImageRegistry, "docker.io") {
		dc.ImageRegistry = "https://registry-1.docker.io"
		if !strings.ContainsAny(dc.ImageRepo, "/") {
			dc.ImageRepo = fmt.Sprintf("library/%s", dc.ImageRepo)
		}
	} else if strings.Contains(dc.ImageRegistry, "gcr.io") {
		dc.ImageRegistry = "https://gcr.io"
	} else if !strings.Contains(dc.ImageRegistry, "http") {
		dc.ImageRegistry = fmt.Sprintf("https://%s", dc.ImageRegistry)
	}
	for _, z := range c.RegAuth {
		if strings.Contains(dc.ImageRegistry, z["url"]) {
			username = os.Getenv(z["username"])
			password = os.Getenv(z["password"])
			match = true
			break
		}
	}
	return username, password, match
}

// ParseImageTagPath func init
func ParseImageTagPath(imagePath string) (imageRegistry, imageRepo, imagTag, imageDigest string, err error) {
	// Breaking out Image path into registry/repo/tag for easier mapping in the
	// image table.
	reg := regexp.MustCompile(`^(.*\..*?)/(.*)@(.*)|^(.*\..*?)/(.*):(.*)`)
	match := reg.FindStringSubmatch(imagePath)

	containerImageRegistry := ""
	containerImageRepo := ""
	containerImageTag := ""
	containerImageDigest := ""

	if len(match) > 0 {
		// registry.com/repo@digest detected, no tag
		if match[1] != "" {
			containerImageRegistry = match[1]
			containerImageRepo = match[2]
			containerImageDigest = match[3]
		}
		// registry.com/repo:tag detected
		if match[4] != "" {
			containerImageRegistry = match[4]
			containerImageRepo = match[5]
			containerImageTag = match[6]
		}
	} else {
		reg = regexp.MustCompile(`^(.*):(.*)`)
		match = reg.FindStringSubmatch(imagePath)
		if len(match) > 2 {
			// No registry listed, only repo/tag
			containerImageRepo = match[1]
			containerImageTag = match[2]
		} else {
			reg = regexp.MustCompile(`^(.*\..*?)/(.*)`)
			match = reg.FindStringSubmatch(imagePath)
			if len(match) > 2 {
				// Only registry/repo found, no tag
				containerImageRegistry = match[1]
				containerImageRepo = match[2]
			} else {
				// only repo found
				containerImageRepo = imagePath
			}
		}
	}

	return containerImageRegistry, containerImageRepo, containerImageTag, containerImageDigest, nil
}

// ParseImageDigest func init
func ParseImageDigest(imageDigestPath string) (imageDigest string, err error) {
	// Also pull out digest for easier image mapping
	reg := regexp.MustCompile(`^docker.*@(.*)`)
	match := reg.FindStringSubmatch(imageDigestPath)
	containerImageDigest := ""
	if len(match) > 0 {
		// Proper digest found
		containerImageDigest = match[1]
	} else {
		reg = regexp.MustCompile(`^docker://(.*)`)
		match = reg.FindStringSubmatch(imageDigestPath)
		if len(match) > 0 {
			// Digest for local image found
			containerImageDigest = match[1]
		} else {
			return "", errors.New("No Digest Match")
		}
	}
	return containerImageDigest, nil
}

func removeDuplicates(elements []*datastore.Container) []*datastore.Container {
	// Use map to record duplicates as we find them
	encountered := map[string]bool{}
	result := []*datastore.Container{}

	for v := range elements {
		if encountered[elements[v].ImageDigest] == true {
			log.Debug(fmt.Sprintf("Duplicate: %s %s\n", elements[v].ImageID, elements[v].ImageDigest))
			// Do not add duplicate
		} else {
			// Record this element as an encountered element
			encountered[elements[v].ImageDigest] = true
			log.Debug(fmt.Sprintf("First: %s %s\n", elements[v].ImageID, elements[v].ImageDigest))
			// Append to result slice
			result = append(result, elements[v])
		}
	}
	// Return the new slice
	return result
}
