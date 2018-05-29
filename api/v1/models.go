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
	"time"

	"github.com/target/portauthority/pkg/clair/client"
	"github.com/target/portauthority/pkg/datastore"
)

// Error struct init
type Error struct {
	Message string `json:"Message,omitempty"`
}

// Image struct init
type Image struct {
	ID               uint64                `json:"ID,omitempty"`
	TopLayer         string                `json:"TopLayer,omitempty"`
	Registry         string                `json:"Registry,omitempty"`
	Repo             string                `json:"Repo,omitempty"`
	Tag              string                `json:"Tag,omitempty"`
	Digest           string                `json:"Digest,omitempty"`
	FirstSeen        time.Time             `json:"FirstSeen,omitempty"`
	LastSeen         time.Time             `json:"LastSeen,omitempty"`
	RegistryUser     string                `json:"RegistryUser,omitempty"`
	RegistryPassword string                `json:"RegistryPassword,omitempty"`
	Features         []clairclient.Feature `json:"Features,omitempty"`
	Violations       []Violation           `json:"Violations,omitempty"`
	Metadata         datastore.JSONMap     `json:"Metadata,omitempty"`
}

// ImageFromDatabaseModel func init
func ImageFromDatabaseModel(dbImage *datastore.Image) Image {
	image := Image{
		ID:        dbImage.ID,
		Registry:  dbImage.Registry,
		Repo:      dbImage.Repo,
		Tag:       dbImage.Tag,
		Digest:    dbImage.Digest,
		Metadata:  dbImage.Metadata,
		FirstSeen: dbImage.FirstSeen,
		LastSeen:  dbImage.LastSeen,
	}

	return image
}

// ImageEnvelope struct init
type ImageEnvelope struct {
	Image *Image `json:"Image,omitempty"`
	Error *Error `json:"Error,omitempty"`
}

// ImagesEnvelope struct init
type ImagesEnvelope struct {
	Images *[]*Image `json:"Images,omitempty"`
	Error  *Error    `json:"Error,omitempty"`
}

// Container struct init
type Container struct {
	ID              uint64                `json:"ID,omitempty"`
	Namespace       string                `json:"Namespace"`
	Cluster         string                `json:"Cluster"`
	Name            string                `json:"Name"`
	Image           string                `json:"Image"`
	ImageScanned    bool                  `json:"ImageScanned"`
	ImageID         string                `json:"ImageID"`
	ImageRegistry   string                `json:"ImageRegistry"`
	ImageRepo       string                `json:"ImageRepo"`
	ImageTag        string                `json:"ImageTag"`
	ImageDigest     string                `json:"ImageDigest"`
	ImageFeatures   []clairclient.Feature `json:"Features,omitempty"`
	ImageViolations []Violation           `json:"Violations,omitempty"`
	Annotations     datastore.JSONMap     `json:"Annotations,omitempty"`
	FirstSeen       time.Time             `json:"FirstSeen"`
	LastSeen        time.Time             `json:"LastSeen"`
}

// ContainerFromDatabaseModel func init
func ContainerFromDatabaseModel(dbContainer *datastore.Container) Container {
	container := Container{
		ID:            dbContainer.ID,
		Namespace:     dbContainer.Namespace,
		Cluster:       dbContainer.Cluster,
		Name:          dbContainer.Name,
		Image:         dbContainer.Image,
		ImageID:       dbContainer.ImageID,
		ImageRegistry: dbContainer.ImageRegistry,
		ImageRepo:     dbContainer.ImageRepo,
		ImageTag:      dbContainer.ImageTag,
		ImageDigest:   dbContainer.ImageDigest,
		Annotations:   dbContainer.Annotations,
		FirstSeen:     dbContainer.FirstSeen,
		LastSeen:      dbContainer.LastSeen,
	}

	return container
}

// ContainerEnvelope struct init
type ContainerEnvelope struct {
	Container *Container `json:"Container,omitempty"`
	Error     *Error     `json:"Error,omitempty"`
}

// ContainersEnvelope struct init
type ContainersEnvelope struct {
	Containers *[]*Container `json:"Containers,omitempty"`
	Error      *Error        `json:"Error,omitempty"`
}

// Policy struct init
type Policy struct {
	ID                  uint64    `json:"ID,omitempty"`
	Name                string    `json:"Name,omitempty"`
	AllowedRiskSeverity string    `json:"AllowedRiskSeverity,omitempty"`
	AllowedCVENames     string    `json:"AllowedCVENames,omitempty"`
	AllowNotFixed       bool      `json:"AllowNotFixed"`
	NotAllowedCveNames  string    `json:"NotAllowedCveNames,omitempty"`
	NotAllowedOSNames   string    `json:"NotAllowedOSNames,omitempty"`
	Created             time.Time `json:"Created,omitempty"`
	Updated             time.Time `json:"Updated,omitempty"`
}

// PolicyEnvelope struct init
type PolicyEnvelope struct {
	Policy *Policy `json:"Policy,omitempty"`
	Error  *Error  `json:"Error,omitempty"`
}

// PoliciesEnvelope struct init
type PoliciesEnvelope struct {
	Policies *[]*Policy `json:"Policies,omitempty"`
	Error    *Error     `json:"Error,omitempty"`
}

// PolicyFromDatabaseModel func init
func PolicyFromDatabaseModel(dbPolicy *datastore.Policy) Policy {
	policy := Policy{
		ID:                  dbPolicy.ID,
		Name:                dbPolicy.Name,
		AllowedRiskSeverity: dbPolicy.AllowedRiskSeverity,
		AllowedCVENames:     dbPolicy.AllowedCVENames,
		AllowNotFixed:       dbPolicy.AllowNotFixed,
		NotAllowedCveNames:  dbPolicy.NotAllowedCveNames,
		NotAllowedOSNames:   dbPolicy.NotAllowedOSNames,
		Created:             dbPolicy.Created,
		Updated:             dbPolicy.Updated,
	}
	return policy
}

// Violation struct init
type Violation struct {
	Type           ViolationType
	FeatureName    string `json:"FeatureName,omitempty"`
	FeatureVersion string `json:"FeatureVersion,omitempty"`
	Vulnerability  clairclient.Vulnerability
}

// ViolationType string init
type ViolationType string

const (
	// BlacklistedOsViolation const init
	BlacklistedOsViolation ViolationType = "BlacklistedOs"
	// BlacklistedCveViolation const init
	BlacklistedCveViolation ViolationType = "BlacklistedCve"
	// BasicViolation const init
	BasicViolation ViolationType = "Basic"
)

// K8sImagePolicyEnvelope struct init
type K8sImagePolicyEnvelope struct {
	K8sImagePolicy *K8sImagePolicy `json:"K8sImagePolicy,omitempty"`
	Error          *Error          `json:"Error,omitempty"`
}

// K8sImagePolicy struct init
type K8sImagePolicy struct {
	APIVersion string             `json:"apiVersion,omitempty"`
	Kind       string             `json:"kind,omitempty"`
	Spec       *K8sImageSpec      `json:"spec,omitempty"`
	Status     *ImageReviewStatus `json:"status,omitempty"`
}

// K8sImageSpec struct init
type K8sImageSpec struct {
	Containers  []K8sContainers   `json:"containers,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Namespace   string            `json:"namespace,omitempty"`
}

// K8sContainers struct init
type K8sContainers struct {
	Image string `json:"image,omitempty"`
}

// ImageReviewStatus is the result of a port authority policy review
type ImageReviewStatus struct {
	// Allowed indicates that all images were allowed to be run
	Allowed bool `json:"allowed"`
	// Reason should be empty unless Allowed is false in which case it
	// may contain a short description of what is wrong.  Kubernetes
	// may truncate excessively long errors when displaying to the user.
	Reason string `json:"reason,omitempty"`
}

// Crawler struct init
type Crawler struct {
	ID       uint64                     `json:"ID,omitempty"`
	Type     string                     `json:"Type,omitempty"`
	Status   string                     `json:"Status,omitempty"`
	Scan     string                     `json:"Scan,omitempty"`
	Messages *datastore.CrawlerMessages `json:"Messages,omitempty"`
	Started  time.Time                  `json:"Started,omitempty"`
	Finished time.Time                  `json:"Finished,omitempty"`
}

// CrawlerMessages struct init
type CrawlerMessages struct {
	Summary string `json:"Summary,omitempty"`
	Error   string `json:"Error,omitempty"`
}

// CrawlerEnvelope struct init
type CrawlerEnvelope struct {
	Crawler *Crawler `json:"Crawler,omitempty"`
	Error   *Error   `json:"Error,omitempty"`
}

// CrawlerFromDatabaseModel func init
func CrawlerFromDatabaseModel(dbCrawler *datastore.Crawler) Crawler {
	crawler := Crawler{
		ID:       dbCrawler.ID,
		Type:     dbCrawler.Type,
		Status:   dbCrawler.Status,
		Messages: dbCrawler.Messages,
		Started:  dbCrawler.Started,
		Finished: dbCrawler.Finished,
	}
	return crawler
}

// RegCrawler struct init
type RegCrawler struct {
	Crawler    Crawler
	MaxThreads uint     `json:"MaxThreads,omitempty"`
	Registry   string   `json:"Registry,omitempty"`
	Username   string   `json:"Username,omitempty"`
	Password   string   `json:"Password,omitempty"`
	Repos      []string `json:"Repos,omitempty"`
	Tags       []string `json:"Tags,omitempty"`
}

// RegCrawlerEnvelope struct init
type RegCrawlerEnvelope struct {
	RegCrawler *RegCrawler `json:"RegCrawler,omitempty"`
	Error      *Error      `json:"Error,omitempty"`
}

// K8sCrawler struct init
type K8sCrawler struct {
	Crawler    Crawler
	Context    string `json:"Context,omitempty"`
	KubeConfig string `json:"KubeConfig,omitempty"`
	Scan       bool   `json:"Scan,omitempty"`
	MaxThreads uint   `json:"MaxThreads,omitempty"`
}

// K8sCrawlerEnvelope struct init
type K8sCrawlerEnvelope struct {
	K8sCrawler *K8sCrawler `json:"K8sCrawler,omitempty"`
	Error      *Error      `json:"Error,omitempty"`
}
