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
	ManifestV2 string    `db:"manifest_v2"`
	ManifestV1 string    `db:"manifest_v1"`
	FirstSeen  time.Time `db:"first_seen"`
	LastSeen   time.Time `db:"last_seen"`
}

// Container DB struct
type Container struct {
	Model
	Namespace     string        `db:"namespace"`
	Cluster       string        `db:"cluster"`
	Name          string        `db:"name"`
	Image         string        `db:"image"`
	ImageID       string        `db:"image_id"`
	ImageRegistry string        `db:"image_registry"`
	ImageRepo     string        `db:"image_repo"`
	ImageTag      string        `db:"image_tag"`
	ImageDigest   string        `db:"image_digest"`
	Annotations   AnnotationMap `db:"annotations"`
	FirstSeen     time.Time     `db:"first_seen"`
	LastSeen      time.Time     `db:"last_seen"`
}

// AnnotationMap is a type for annotations jsonb column
type AnnotationMap map[string]interface{}

// Value returns marshalled json from AnnotationMap
func (a AnnotationMap) Value() (driver.Value, error) {
	j, err := json.Marshal(a)
	return j, err
}

// Scan transforms raw jsonb data to AnnotationMap type
func (a *AnnotationMap) Scan(src interface{}) error {
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

	*a, ok = i.(map[string]interface{})
	if !ok {
		return errors.New("type assertion .(map[string]interface{}) failed")
	}

	return nil
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
