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
func App() *cli.App {
	app := cli.NewApp()
	app.Name = "Port Authority"
	app.Version = "v0.4.0"
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
