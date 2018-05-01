package main

import (
	"os"

	"github.com/target/portauthority/cmd"

	//this registers the db driver
	_ "github.com/target/portauthority/pkg/datastore/pgsql"
)

func main() {
	app := cmd.App()
	app.Run(os.Args)

}
