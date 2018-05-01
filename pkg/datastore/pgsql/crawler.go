package pgsql

import (
	"database/sql"
	"encoding/json"

	"github.com/target/portauthority/pkg/commonerr"
	"github.com/target/portauthority/pkg/datastore"
	"github.com/pkg/errors"
)

func (p *pgsql) GetCrawler(id int) (*datastore.Crawler, error) {
	var crawler datastore.Crawler

	var messages string

	err := p.QueryRow(`SELECT id,
      type,
      status,
      messages,
      started,
      finished
      FROM crawler_pa WHERE id=$1`,
		id).Scan(&crawler.ID,
		&crawler.Type,
		&crawler.Status,
		&messages,
		&crawler.Started,
		&crawler.Finished)

	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, commonerr.ErrNotFound
		default:
			return nil, errors.Wrap(err, "error querying for crawler")
		}
	}

	crawlerMessage := &datastore.CrawlerMessages{}
	if messages != "" {
		err = json.Unmarshal([]byte(messages), &crawlerMessage)
		if err != nil {
			return nil, errors.Wrap(err, "improper crawler message formating")
		}
	}
	crawler.Messages = crawlerMessage

	return &crawler, nil
}

func (p *pgsql) InsertCrawler(crawler *datastore.Crawler) (int64, error) {

	// Upserting container in db
	var id int64
	err := p.QueryRow(`
    INSERT INTO crawler_pa as c (
        type,
        status,
        started)
        VALUES($1, $2, $3)
        RETURNING id`,
		crawler.Type,
		crawler.Status,
		crawler.Started).Scan(&id)

	if err != nil {
		return -1, errors.Wrap(err, "error inserting crawler")
	}

	return id, nil
}

func (p *pgsql) UpdateCrawler(id int64, crawler *datastore.Crawler) error {

	// Updating container in db
	messages, err := json.Marshal(crawler.Messages)
	if err != nil {
		return errors.Wrap(err, "error marshaling crawler messages")
	}

	_, err = p.Exec(`
    UPDATE crawler_pa SET status = $2, messages = $3, finished = $4 WHERE id = $1`,
		id,
		crawler.Status,
		string(messages),
		crawler.Finished)

	if err != nil {
		return errors.Wrap(err, "error updating crawler")
	}

	return nil
}
