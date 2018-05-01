package cmd

import (
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/target/portauthority/api"
	"github.com/target/portauthority/pkg/clair/client"
	"github.com/target/portauthority/pkg/datastore"
	"github.com/target/portauthority/pkg/formatter"
	"github.com/target/portauthority/pkg/stopper"
	log "github.com/Sirupsen/logrus"
	"github.com/urfave/cli"
)

func newServeCommand() cli.Command {
	return cli.Command{
		Name:        "serve",
		Description: "Starts Port Authority as a daemon",
		Usage:       "portauthority serve [OPTIONS]",
		Action:      serve,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "config, c",
				Usage:  "path to configuration file",
				EnvVar: "PA_CONFIG",
			},
			cli.BoolFlag{
				Name:   "insecure-tls, i",
				Usage:  "Disable TLS server's certificate chain and hostname verification when talking to other services",
				EnvVar: "PA_INSECURE_TLS",
				Hidden: false,
			},
			cli.StringFlag{
				Name:   "log-level, l",
				Usage:  "Define the logging level.",
				EnvVar: "PA_LOG_LEVEL",
				Value:  "info",
			},
		},
	}
}

func waitForSignals(signals ...os.Signal) {
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, signals...)
	<-interrupts
}

func serve(ctx *cli.Context) error {

	// Load configuration
	config, err := LoadConfig(ctx.String("config"))
	if err != nil {
		log.WithError(err).Fatal("failed to load configuration")
	}

	logLevel, err := log.ParseLevel(strings.ToUpper(ctx.String("log-level")))
	log.SetLevel(logLevel)
	log.SetOutput(os.Stdout)
	log.SetFormatter(&formatter.JSONExtendedFormatter{ShowLn: true})

	rand.Seed(time.Now().UnixNano())
	st := stopper.NewStopper()

	// Open database
	db, err := datastore.Open(config.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create clair client
	cc := clairclient.DefaultConfig()
	cc.Address = config.API.ClairURL
	cc.HTTPClient.Timeout = time.Second * time.Duration(config.API.ClairTimeout)
	client, err := clairclient.NewClient(cc)
	if err != nil {
		log.Fatal(err, "error creating clair client")
	}

	// Start API
	st.Begin()
	go api.Run(config.API, *client, db, st)
	st.Begin()
	go api.RunHealth(config.API, db, st)

	// Wait for interruption and shutdown gracefully.
	waitForSignals(syscall.SIGINT, syscall.SIGTERM)
	log.Info("Received interruption, gracefully stopping ...")
	st.Stop()

	return nil
}
