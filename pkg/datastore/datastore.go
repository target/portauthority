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

package datastore

import (
	"fmt"

	"github.com/pkg/errors"
)

// BackendFactory defines the function that should open a database connection
type BackendFactory func(config BackendConfig) (Backend, error)

var backendFactories = make(map[string]BackendFactory)

// Register adds a new datastore factory
func Register(name string, factory BackendFactory) {
	if factory == nil {
		panic(fmt.Sprintf("Command Factory %s does not exist", name))
	}

	_, registered := backendFactories[name]
	if registered {
		panic(fmt.Sprintf("Command factory %s already registered", name))
	}

	backendFactories[name] = factory
}

// Open opens a database connection
func Open(conf BackendConfig) (Backend, error) {
	factory, ok := backendFactories[conf.Type]
	if !ok {
		return nil, errors.Errorf("no datastore backend for type %s", conf.Type)
	}

	return factory(conf)
}

// Backend defines the functionality of a datastore
type Backend interface {
	GetImage(registry, repo, tag, digest string) (*Image, error)
	GetAllImages(registry, repo, tag, digest, dateStart, dateEnd, limit string) (*[]*Image, error)
	GetImageByID(id int) (*Image, error)
	GetImageByRrt(registry, repo, tag string) (*Image, error)
	GetImageByDigest(digest string) (*Image, error)
	UpsertImage(*Image) error

	DeleteImage(registry, repo, tag, digest string) (bool, error)
	GetContainer(namespace, cluster, name, image, imageID string) (*Container, error)
	GetContainerByID(id int) (*Container, error)

	GetAllContainers(namespace, cluster, name, image, imageID, dateStart, dateEnd, limit string) (*[]*Container, error)
	UpsertContainer(*Container) error

	GetPolicy(name string) (*Policy, error)
	GetAllPolicies(name string) (*[]*Policy, error)
	UpsertPolicy(*Policy) error

	GetCrawler(id int) (*Crawler, error)
	InsertCrawler(*Crawler) (int64, error)
	UpdateCrawler(int64, *Crawler) error

	// Ping returns the health status of the database
	Ping() bool

	// Close closes the database and frees any allocated resource
	Close()
}

// BackendConfig defines the type of datastore to create and its parameters
type BackendConfig struct {
	Type    string
	Options map[string]interface{}
}
