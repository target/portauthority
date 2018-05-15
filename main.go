// Copyright (c) 2018 Target Brands, Inc.

package main

import (
	"os"

	"github.com/target/portauthority/cmd"

	//this registers the db driver
	_ "github.com/target/portauthority/pkg/datastore/pgsql"
)

var appVersion string

func main() {
	app := cmd.App(appVersion)
	app.Run(os.Args)

}
