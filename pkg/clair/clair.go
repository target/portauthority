// Copyright (c) 2018 Target Brands, Inc.

package clair

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	log "github.com/Sirupsen/logrus"

	"github.com/target/portauthority/pkg/clair/client"

	"github.com/pkg/errors"
)

// Image struct init
type Image struct {
	Digest   string   `json:"digest"`
	Registry string   `json:"registry"`
	Repo     string   `json:"repo"`
	Tag      string   `json:"tag"`
	Layers   []string `json:"layers"`
	Headers  map[string]string
}

// Push will push an image's layers into Clair
// Headers are passed to Clair and reused on requests it makes to registries
func Push(client *clairclient.Client, image Image) error {
	parent := ""

	// We need to go in reverse order of the layers obtained from the manifest
	// The last layer in the list is the base parent layer
	for i := len(image.Layers) - 1; i >= 0; i-- {

		// Append the image id to the layer to create a unqiue value that allows us
		// to maintain the parent/child relationships.
		// As a result of doing this, we download each image at least once, but the
		// false postitive detection is worth it.
		if parent != "" {
			parent = strings.Join([]string{string(image.Digest), parent}, "")
			parent = GetMD5Hash(parent)
		}

		log.Debug("Clair Precalculated Hash: ", image.Digest, image.Layers[i])
		layer := strings.Join([]string{string(image.Digest), image.Layers[i]}, "")
		layer = GetMD5Hash(layer)

		le, err := client.PostLayers(&clairclient.Layer{
			Name:       layer,
			ParentName: parent,
			Path:       strings.Join([]string{image.Registry, "v2", image.Repo, "blobs", image.Layers[i]}, "/"),
			Format:     "Docker",
			Headers:    map[string]string{"Authorization": image.Headers["Authorization"]},
		})
		if err != nil {
			return errors.Wrapf(err, "error pushing layer %s to clair", image.Layers[i])
		}

		if le.Error != nil {
			log.Error("Error: \n", le.Error.Message)
		} else {
			log.Debug("Name: ", le.Layer.Name)
			log.Debug("Parent: ", le.Layer.ParentName)
			log.Debug("Indexed by Version: ", le.Layer.IndexedByVersion)
		}

		// Port Authority keeps the manifest record so we can put things back
		// together in the right order.
		parent = image.Layers[i]
	}

	return nil
}

// Get func init
func Get(client clairclient.Client, image Image) ([]*clairclient.LayerEnvelope, error) {
	var layers []*clairclient.LayerEnvelope
	for _, layer := range image.Layers {
		le, err := client.GetLayers(layer, false, false)
		if err != nil {
			if clairclient.IsStatusCodeError(err) && clairclient.ErrorStatusCode(err) == 404 {
				return nil, errors.Wrapf(err, "error getting data for layer %s in image %s/%s:%s", layer, image.Registry, image.Repo, image.Tag)
			}
		}
		layers = append(layers, le)
	}

	return layers, nil
}

// GetMD5Hash gets the MD5 hash of a string
func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
