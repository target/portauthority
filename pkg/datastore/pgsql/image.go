// Copyright (c) 2018 Target Brands, Inc.

// Most of the code and structure on how the features and vulnerabilites are
// found and returned are ported from the Clair implementation.
// The biggest difference is that we base this on an image instead of layers.

package pgsql

import (
	"database/sql"
	"regexp"

	log "github.com/sirupsen/logrus"
	"github.com/target/portauthority/pkg/commonerr"

	"github.com/pkg/errors"
	"github.com/target/portauthority/pkg/datastore"
)

// GetAllImages returns all images from the psql table 'images'
func (p *pgsql) GetAllImages(registry, repo, tag, digest, dateStart, dateEnd, limit string) (*[]*datastore.Image, error) {
	var images []*datastore.Image
	query := "SELECT id, top_layer, registry, repo, tag, digest, manifest_v2, manifest_v1, first_seen, last_seen FROM image_pa WHERE registry LIKE '%' || $1 || '%' AND repo LIKE '%' || $2 || '%' AND tag LIKE '%' || $3 || '%' AND digest LIKE '%' || $4 || '%'"

	// Regex conditionals should prevent possible SQL injection from unescaped
	// concatenation to query string.
	if isDate, _ := regexp.MatchString(`^[\d]{4}-[\d]{2}-[\d]{2}$`, dateStart); isDate {
		query += " AND last_seen>='" + dateStart + "'::date"
	}

	if isDate, _ := regexp.MatchString(`^[\d]{4}-[\d]{2}-[\d]{2}$`, dateEnd); isDate {
		query += " AND (last_seen<'" + dateEnd + "'::date + '1 day'::interval)"
	}

	if isInt, _ := regexp.MatchString(`^[\d]{1,}$`, limit); isInt {
		query += " LIMIT " + limit
	}

	rows, err := p.Query(query, registry, repo, tag, digest)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "error querying for images")
		}
	}
	defer rows.Close()
	for rows.Next() {
		var image datastore.Image
		err = rows.Scan(&image.ID, &image.TopLayer, &image.Registry, &image.Repo, &image.Tag, &image.Digest, &image.ManifestV2, &image.ManifestV1, &image.FirstSeen, &image.LastSeen)
		if err != nil {
			return nil, errors.Wrap(err, "error scanning images")
		}
		images = append(images, &image)
	}
	// Get any error encountered during iteration
	err = rows.Err()
	if err != nil {
		return nil, errors.Wrap(err, "error scanning images")
	}
	return &images, nil
}

// TODO: Improve this func
func (p *pgsql) GetImage(registry, repo, tag, digest string) (*datastore.Image, error) {
	var image datastore.Image
	err := p.QueryRow("SELECT id, top_layer, registry, repo, tag, digest, manifest_v2, manifest_v1, first_seen, last_seen FROM image_pa WHERE registry=$1 AND repo=$2 AND tag=$3 AND digest=$4", registry, repo, tag, digest).Scan(&image.ID, &image.TopLayer, &image.Registry, &image.Repo, &image.Tag, &image.Digest, &image.ManifestV2, &image.ManifestV1, &image.FirstSeen, &image.LastSeen)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "error querying for image")
		}
	}
	return &image, nil
}

// GetImageRrt retures the last image seen with a matching registry repo tag.
// The preferred way to obtain the correct image from the database is via the
// GetImage function which requires the SHA.
// This was put in place for K8s ImagePolicyWebhook that may only have this
// information available.
// TODO: WHERE registry LIKE '%' || exists because if the registry has http,
// then it needs to be stripped from the addition of the image. This is a
// workaround.
func (p *pgsql) GetImageByRrt(registry, repo, tag string) (*datastore.Image, error) {
	var image datastore.Image
	err := p.QueryRow("SELECT id, top_layer, registry, repo, tag, digest, manifest_v2, manifest_v1, first_seen, last_seen FROM image_pa WHERE registry LIKE '%' || $1 AND repo=$2 AND tag=$3 order by last_seen desc limit 1", registry, repo, tag).Scan(&image.ID, &image.TopLayer, &image.Registry, &image.Repo, &image.Tag, &image.Digest, &image.ManifestV2, &image.ManifestV1, &image.FirstSeen, &image.LastSeen)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, commonerr.ErrNotFound
		default:
			return nil, errors.Wrap(err, "error querying for image")
		}
	}
	return &image, nil
}

func (p *pgsql) GetImageByDigest(digest string) (*datastore.Image, error) {
	var image datastore.Image
	err := p.QueryRow("SELECT id, top_layer, registry, repo, tag, digest, manifest_v2, manifest_v1, first_seen, last_seen FROM image_pa WHERE digest=$1 order by last_seen desc limit 1", digest).Scan(&image.ID, &image.TopLayer, &image.Registry, &image.Repo, &image.Tag, &image.Digest, &image.ManifestV2, &image.ManifestV1, &image.FirstSeen, &image.LastSeen)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, commonerr.ErrNotFound
		default:
			return nil, errors.Wrap(err, "error querying for image")
		}
	}
	return &image, nil
}

func (p *pgsql) GetImageByID(id int) (*datastore.Image, error) {
	var image datastore.Image
	err := p.QueryRow("SELECT id, top_layer, registry, repo, tag, digest, manifest_v2, manifest_v1, first_seen, last_seen FROM image_pa WHERE id=$1", id).Scan(&image.ID, &image.TopLayer, &image.Registry, &image.Repo, &image.Tag, &image.Digest, &image.ManifestV2, &image.ManifestV1, &image.FirstSeen, &image.LastSeen)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, errors.WithMessage(commonerr.ErrNotFound, "Image not found")
		default:
			return nil, errors.Wrap(err, "error querying for image")
		}
	}

	return &image, nil
}

func (p *pgsql) UpsertImage(image *datastore.Image) error {
	// Checking for unsafe chars within manifests
	reg, err := regexp.Compile("'")
	if err != nil {
		log.Fatal(err)
	}

	safeManifestV1 := reg.ReplaceAllString(image.ManifestV1, "''")
	if safeManifestV1 == "" {
		safeManifestV1 = "{}"
	}
	safeManifestV2 := reg.ReplaceAllString(image.ManifestV2, "''")
	if safeManifestV2 == "" {
		safeManifestV2 = "{}"
	}

	_, err = p.Exec("INSERT INTO image_pa as i (top_layer, registry, repo, tag, digest, manifest_v2, manifest_v1, first_seen, last_seen) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (registry, repo, tag, digest) DO UPDATE SET last_seen = $9 WHERE i.registry = $2 AND i.repo = $3 AND i.tag = $4 AND i.digest = $5",
		image.TopLayer,
		image.Registry,
		image.Repo,
		image.Tag,
		image.Digest,
		safeManifestV2,
		safeManifestV1,
		image.FirstSeen,
		image.LastSeen,
	)
	if err != nil {
		return errors.Wrap(err, "error upserting image")
	}

	return nil
}

func (p *pgsql) DeleteImage(registry, repo, tag, digest string) (bool, error) {
	found := false
	res, err := p.Exec("DELETE FROM image_pa WHERE registry=$1 AND repo=$2 AND tag=$3 AND digest=$4", registry, repo, tag, digest)
	// Should delete layers in layer table now, too
	if err != nil {
		return false, errors.Wrap(err, "error deleting image")
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, errors.Wrap(err, "error checking rows affected")
	}

	if affected > 0 {
		// TODO: Should we have some checking to make sure this is
		// never more than 1 with dry run? It's probably a bug, anyway, if there are
		// multiple images found with these search params.
		found = true
	}

	return found, nil
}
