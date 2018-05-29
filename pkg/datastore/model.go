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
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// Model enforces the existence of id for model structs
type Model struct {
	ID uint64 `db:"id"`
}

// Image DB Struct
type Image struct {
	Model
	TopLayer   string    `db:"top_layer"`
	Registry   string    `db:"registry"`
	Repo       string    `db:"repo"`
	Tag        string    `db:"tag"`
	Digest     string    `db:"digest"`
	ManifestV2 JSONMap   `db:"manifest_v2"`
	ManifestV1 JSONMap   `db:"manifest_v1"`
	Metadata   JSONMap   `db:"metadata"`
	FirstSeen  time.Time `db:"first_seen"`
	LastSeen   time.Time `db:"last_seen"`
}

// Container DB struct
type Container struct {
	Model
	Namespace     string    `db:"namespace"`
	Cluster       string    `db:"cluster"`
	Name          string    `db:"name"`
	Image         string    `db:"image"`
	ImageID       string    `db:"image_id"`
	ImageRegistry string    `db:"image_registry"`
	ImageRepo     string    `db:"image_repo"`
	ImageTag      string    `db:"image_tag"`
	ImageDigest   string    `db:"image_digest"`
	Annotations   JSONMap   `db:"annotations"`
	FirstSeen     time.Time `db:"first_seen"`
	LastSeen      time.Time `db:"last_seen"`
}

// Policy DB struct
type Policy struct {
	Model
	Name                string    `db:"name"`
	AllowedRiskSeverity string    `db:"allowed_risk_severity"`
	AllowedCVENames     string    `db:"allowed_cve_names"`
	AllowNotFixed       bool      `db:"allow_not_fixed"`
	NotAllowedCveNames  string    `db:"not_allowed_cve_names"`
	NotAllowedOSNames   string    `db:"not_allowed_os_names"`
	Created             time.Time `db:"created"`
	Updated             time.Time `db:"updated"`
}

// Crawler DB struct
type Crawler struct {
	Model
	Type     string           `db:"type"`
	Status   string           `db:"status"`
	Messages *CrawlerMessages `db:"messages"`
	Started  time.Time        `db:"started"`
	Finished time.Time        `db:"finished"`
}

// CrawlerMessages Field struct will contain basic information in JSON db format.
// Detailed information about the scan will be still written to standard out.
type CrawlerMessages struct {
	Summary string `json:"summary,omitempty"`
	Error   string `json:"error,omitempty"`
}

// JSONMap is a type for jsonb columns
type JSONMap map[string]interface{}

// Value returns marshalled json from JSONMap
func (m JSONMap) Value() (driver.Value, error) {
	j, err := json.Marshal(m)
	return j, err
}

// Scan transforms raw jsonb data to JSONMap type
func (m *JSONMap) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	source, ok := src.([]byte)
	if !ok {
		return errors.New("type assertion .([]byte) failed")
	}

	var i interface{}
	err := json.Unmarshal(source, &i)
	if err != nil {
		return err
	}

	*m, ok = i.(map[string]interface{})
	if !ok {
		return errors.New("type assertion .(map[string]interface{}) failed")
	}

	return nil
}
