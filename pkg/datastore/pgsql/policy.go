// Copyright (c) 2018 Target Brands, Inc.

// Most of the code and structure on how the features and vulnerabilites are
// found and returned are ported from the Clair implementation.
// The biggest difference is that we base this on an image instead of layers.

package pgsql

import (
	"database/sql"

	"github.com/pkg/errors"
	"github.com/target/portauthority/pkg/datastore"
)

// GetPolicy will return a single database policy
func (p *pgsql) GetPolicy(name string) (*datastore.Policy, error) {
	var policy datastore.Policy
	err := p.QueryRow("SELECT id, name, array_to_json(allowed_risk_severity), array_to_json(allowed_cve_names), allow_not_fixed, array_to_json(not_allowed_cve_names), array_to_json(not_allowed_os_names), created, updated FROM policy_pa WHERE name=$1", name).Scan(&policy.ID, &policy.Name, &policy.AllowedRiskSeverity, &policy.AllowedCVENames, &policy.AllowNotFixed, &policy.NotAllowedCveNames, &policy.NotAllowedOSNames, &policy.Created, &policy.Updated)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "error querying for policy")
		}
	}
	return &policy, nil
}

// GetAllPolicies returns an an array of Policies based on the input parameters
func (p *pgsql) GetAllPolicies(name string) (*[]*datastore.Policy, error) {
	var policies []*datastore.Policy
	rows, err := p.Query("SELECT id, name, array_to_json(allowed_risk_severity), array_to_json(allowed_cve_names), allow_not_fixed, array_to_json(not_allowed_cve_names), array_to_json(not_allowed_os_names), created, updated FROM policy_pa WHERE name LIKE '%' || $1 || '%'", name)
	if err != nil {
		switch err {
		case sql.ErrNoRows:
			return nil, nil
		default:
			return nil, errors.Wrap(err, "error querying for policies")
		}
	}
	defer rows.Close()
	for rows.Next() {
		var policy datastore.Policy
		err = rows.Scan(&policy.ID, &policy.Name, &policy.AllowedRiskSeverity, &policy.AllowedCVENames, &policy.AllowNotFixed, &policy.NotAllowedCveNames, &policy.NotAllowedOSNames, &policy.Created, &policy.Updated)
		if err != nil {
			return nil, errors.Wrap(err, "error scanning policies")
		}
		policies = append(policies, &policy)
	}
	// Get any error encountered during iteration
	err = rows.Err()
	if err != nil {
		return nil, errors.Wrap(err, "error scanning policies")
	}
	return &policies, nil
}

func (p *pgsql) UpsertPolicy(policy *datastore.Policy) error {

	_, err := p.Exec("INSERT INTO policy_pa as p (name, allowed_risk_severity, allowed_cve_names, allow_not_fixed, not_allowed_cve_names, not_allowed_os_names, created, updated) VALUES($1, $2, $3, $4, $5, $6, $7, $8) ON CONFLICT (name) DO UPDATE SET allowed_risk_severity=$2, allowed_cve_names=$3, allow_not_fixed=$4, not_allowed_cve_names=$5, not_allowed_os_names=$6, updated=$8 WHERE p.name = $1",
		policy.Name,
		policy.AllowedRiskSeverity,
		policy.AllowedCVENames,
		policy.AllowNotFixed,
		policy.NotAllowedCveNames,
		policy.NotAllowedOSNames,
		policy.Created,
		policy.Updated,
	)
	if err != nil {
		return errors.Wrap(err, "error upserting policy")
	}

	return nil
}

func (p *pgsql) DeletePolicy(name string) (bool, error) {
	found := false
	res, err := p.Exec("DELETE FROM policy_pa WHERE name=$1", name)

	if err != nil {
		return false, errors.Wrap(err, "error deleting policy")
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
