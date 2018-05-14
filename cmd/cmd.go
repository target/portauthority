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

package cmd

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// CommandFactory func init
type CommandFactory func() cli.Command

var commandFactories = make(map[string]CommandFactory)

func init() {
	RegisterCommand("serve", newServeCommand)
}

// App func init
func App(appVersion string) *cli.App {
	app := cli.NewApp()
	app.Name = "Port Authority"
	app.Version = appVersion
	app.Usage = ""
	app.Flags = []cli.Flag{}

	for _, factory := range commandFactories {
		app.Commands = append(app.Commands, factory())
	}

	return app
}

// RegisterCommand adds a new command factory that will be used to build cli
// commands and returns error if factory is nil or command has always been
// registered
func RegisterCommand(name string, factory CommandFactory) error {
	if factory == nil {
		return errors.Errorf("Command Factory %s does not exist", name)
	}

	_, registered := commandFactories[name]
	if registered {
		return errors.Errorf("Command factory %s already registered", name)
	}

	commandFactories[name] = factory

	return nil
}
