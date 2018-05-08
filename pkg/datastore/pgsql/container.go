// Copyright (c) 2018 Target Brands, Inc.

package pgsql

import (
	"database/sql"
	"encoding/json"
	"regexp"

	"github.com/pkg/errors"
	"github.com/target/portauthority/pkg/commonerr"
	"github.com/target/portauthority/pkg/datastore"
)

func (p *pgsql) GetContainerByID(id int) (*datastore.Container, error) {
	var container datastore.Container

	err := p.QueryRow(`SELECT id,
    namespace,
    cluster,
    name,
    image,
    image_id,
    image_registry,
    image_repo,
    image_tag,
    image_digest,
    annotations,
    first_seen,
    last_seen
    FROM container_pa WHERE id=$1`,
		id).Scan(&container.ID,
		&container.Namespace,
		&container.Cluster,
		&container.Name,
		&container.Image,
		&container.ImageID,
		&container.ImageRegistry,
		&container.ImageRepo,
		&container.ImageTag,
		&container.ImageDigest,
		&container.Annotations,
		&container.FirstSeen,
		&container.LastSeen)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, commonerr.ErrNotFound
		default:
			return nil, errors.Wrap(err, "error querying for container")
		}
	}
	return &container, nil
}

func (p *pgsql) GetContainer(namespace, cluster, name, image, imageID string) (*datastore.Container, error) {
	var container datastore.Container

	err := p.QueryRow(`SELECT id,
    namespace,
    cluster,
    name,
    image,
    image_id,
    image_registry,
    image_repo,
    image_tag,
    image_digest,
    annotations,
    first_seen,
    last_seen
    FROM container_pa WHERE namespace=$1 AND cluster=$2 AND name=$3 AND image=$4 AND image_id=$5`,
		namespace, cluster, name, image, imageID).Scan(&container.ID,
		&container.Namespace,
		&container.Cluster,
		&container.Name,
		&container.Image,
		&container.ImageID,
		&container.ImageRegistry,
		&container.ImageRepo,
		&container.ImageTag,
		&container.ImageDigest,
		&container.Annotations,
		&container.FirstSeen,
		&container.LastSeen)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "error querying for container")
		}
	}
	return &container, nil
}

func (p *pgsql) GetAllContainers(namespace, cluster, name, image, imageID, dateStart, dateEnd, limit string) (*[]*datastore.Container, error) {
	var containers []*datastore.Container
	query := `SELECT id,
	  namespace,
	  cluster,
	  name,
	  image,
	  image_id,
	  image_registry,
	  image_repo,
	  image_tag,
	  image_digest,
	  annotations,
	  first_seen,
	  last_seen
	  FROM container_pa
	  WHERE namespace LIKE '%' || $1 || '%'
	  AND cluster LIKE '%' || $2 || '%'
	  AND name LIKE '%' || $3 || '%'
	  AND image LIKE '%' || $4 || '%'
	  AND image_id LIKE '%' || $5 || '%'`

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

	rows, err := p.Query(query, namespace, cluster, name, image, imageID)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "error querying for containers")
		}
	}
	defer rows.Close()
	for rows.Next() {
		var container datastore.Container
		err = rows.Scan(&container.ID, &container.Namespace, &container.Cluster, &container.Name, &container.Image, &container.ImageID, &container.ImageRegistry, &container.ImageRepo, &container.ImageTag, &container.ImageDigest, &container.Annotations, &container.FirstSeen, &container.LastSeen)
		if err != nil {
			return nil, errors.Wrap(err, "error scanning containers")
		}
		containers = append(containers, &container)
	}
	// Get any errors encountered during iteration
	err = rows.Err()
	if err != nil {
		return nil, errors.Wrap(err, "error scanning containers")
	}
	return &containers, nil
}

func (p *pgsql) UpsertContainer(container *datastore.Container) error {
	annotationsJSON, _ := json.Marshal(container.Annotations)

	// Upserting container in db
	_, err := p.Exec(`
    INSERT INTO container_pa AS c (
        namespace,
        cluster,
        name,
        image,
        image_id,
        image_registry,
        image_repo,
        image_tag,
        image_digest,
        annotations,
        first_seen,
        last_seen)
        VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
        ON CONFLICT (namespace, cluster, name, image, image_id)
        DO UPDATE SET last_seen = $12, annotations = $10 WHERE c.namespace = $1 AND c.cluster = $2 AND c.name = $3 AND c.image = $4 AND c.image_id = $5`,
		container.Namespace,
		container.Cluster,
		container.Name,
		container.Image,
		container.ImageID,
		container.ImageRegistry,
		container.ImageRepo,
		container.ImageTag,
		container.ImageDigest,
		string(annotationsJSON),
		container.FirstSeen,
		container.LastSeen)

	if err != nil {
		return errors.Wrap(err, "error upserting container")
	}

	return nil
}
