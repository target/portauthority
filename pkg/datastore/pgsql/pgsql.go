package pgsql

import (
	"database/sql"
	"log"

	"github.com/pkg/errors"

	"github.com/target/portauthority/pkg/datastore"

	// Import Postgres driver for use through database/sql
	_ "github.com/lib/pq"
)

func init() {
	datastore.Register("pgsql", openDatabase)
}

// Config parameterizes a pgsql datastore backend
type Config struct {
	Source string
}

type pgsql struct {
	*sql.DB
}

// Ping verifies that the database is accessible
func (p *pgsql) Ping() bool {
	return p.DB.Ping() == nil
}

func openDatabase(backendConfig datastore.BackendConfig) (datastore.Backend, error) {
	var pg pgsql
	var err error

	config := &Config{
		Source: "host=localhost port=5432 user=postgres sslmode=disable statement_timeout=60000",
	}

	src, exists := backendConfig.Options["source"]
	if exists {
		str, ok := src.(string)
		if ok {
			config.Source = str
		}
	}

	pg.DB, err = sql.Open("postgres", config.Source)
	if err != nil {
		pg.Close()
		return nil, errors.Wrap(err, "error opening database connection")
	}

	// Verify database state
	if err = pg.DB.Ping(); err != nil {
		pg.Close()
		return nil, errors.Wrap(err, "error communicating with database")
	}

	pg.initDatabase()

	return &pg, nil
}

func (p *pgsql) initDatabase() error {

	// Create image table if it doesn't exist
	initSQL := `
			CREATE TABLE IF NOT EXISTS image_pa(
					id SERIAL PRIMARY KEY,
					top_layer VARCHAR,
					registry VARCHAR,
					repo VARCHAR,
					tag VARCHAR,
					digest VARCHAR,
					manifest_v2 JSONB,
					manifest_v1 JSONB,
					first_seen TIMESTAMPTZ,
					last_seen TIMESTAMPTZ,
					unique (registry, repo, tag, digest)
			);

			CREATE INDEX IF NOT EXISTS idx_image_id ON image_pa (id);
			CREATE INDEX IF NOT EXISTS idx_image_top_layer ON image_pa (top_layer);
			CREATE INDEX IF NOT EXISTS idxcreated ON image_pa (((manifest_v1->'history'->0->>'v1Compatibility')::JSON ->>'created'));

			CREATE TABLE IF NOT EXISTS container_pa(
					id SERIAL PRIMARY KEY,
					namespace VARCHAR,
					cluster VARCHAR,
					name VARCHAR,
					image VARCHAR,
					image_id VARCHAR,
					image_registry VARCHAR,
					image_repo VARCHAR,
					image_tag VARCHAR,
					image_digest VARCHAR,
					annotations JSONB,
					first_seen TIMESTAMPTZ,
					last_seen TIMESTAMPTZ,
					unique (namespace, cluster, name, image, image_id)
			);

			CREATE INDEX IF NOT EXISTS idx_image_registry ON container_pa (image_registry);
			CREATE INDEX IF NOT EXISTS idx_layer_image_repo ON container_pa (image_repo);
			CREATE INDEX IF NOT EXISTS idx_layer_image_tag ON container_pa (image_tag);
			CREATE INDEX IF NOT EXISTS idx_layer_image_digest ON container_pa (image_digest);

			CREATE TABLE IF NOT EXISTS policy_pa(
					id SERIAL PRIMARY KEY,
					name VARCHAR NOT NULL,
					allowed_risk_severity VARCHAR[] DEFAULT '{}',
					allowed_cve_names VARCHAR[] DEFAULT '{}',
					allow_not_fixed BOOLEAN,
					not_allowed_cve_names VARCHAR[] DEFAULT '{}',
					not_allowed_os_names VARCHAR[] DEFAULT '{}',
					created TIMESTAMPTZ,
					updated TIMESTAMPTZ,
					unique (name)
			);

			CREATE INDEX IF NOT EXISTS idx_policy_id ON policy_pa (id);

			INSERT INTO policy_pa as p (name,allow_not_fixed,created,updated) VALUES('default','false',now(),now()) ON CONFLICT (name) DO NOTHING;

			CREATE TABLE IF NOT EXISTS crawler_pa(
					id SERIAL PRIMARY KEY,
					type VARCHAR NOT NULL,
					status VARCHAR NOT NULL,
					messages VARCHAR,
					started TIMESTAMPTZ,
					finished TIMESTAMPTZ,
					unique (id)
			);

			CREATE INDEX IF NOT EXISTS idx_crawler_id ON crawler_pa (id);
			`
	res, err := p.Exec(initSQL)
	if err != nil {
		return errors.Wrap(err, "error initializing port authority database tables")
	}

	rowCnt, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "error resolving number of rows affected by init sql")
	}
	if rowCnt > 0 {
		log.Printf("Port Authority DB Initialized")
	}

	return nil
}

// Close closes the database
func (p *pgsql) Close() {
	if p.DB != nil {
		p.DB.Close()
	}
}
