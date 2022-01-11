package main

import (
	"context"
	"errors"
	"net/url"
	"os"
	"plane.watch/lib/stats"
	"plane.watch/lib/tile_grid"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"plane.watch/lib/export"
	"plane.watch/lib/logging"
	"plane.watch/lib/rabbitmq"
)

// queue suffixes for a low (only significant) and high (every message) tile queues
const (
	qSuffixLow  = "_low"
	qSuffixHigh = "_high"
)

type (
	pwRouter struct {
		rmq  *rabbitmq.RabbitMQ
		conf *rabbitmq.Config

		syncSamples sync.Map
	}

	planeLocationLast struct {
		lastSignificantUpdate export.EnrichedPlaneLocation
		candidateUpdate       export.EnrichedPlaneLocation
	}
)

var (
	updatesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pw_router_updates_processed_total",
		Help: "The total number of messages processed.",
	})
	updatesSignificant = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pw_router_updates_significant_total",
		Help: "The total number of messages determined to be significant.",
	})
	updatesIgnored = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pw_router_updates_ignored_total",
		Help: "The total number of messages determined to be insignificant and thus ignored.",
	})
	updatesPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pw_router_updates_published_total",
		Help: "The total number of messages published to the output queue.",
	})
	updatesError = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pw_router_updates_error_total",
		Help: "The total number of messages that could not be processed due to an error.",
	})
)

func main() {
	app := cli.NewApp()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	app.Version = "1.0.0"
	app.Name = "Plane Watch Router (pw_router)"
	app.Usage = "Reads location updates from AMQP and publishes only significant updates."

	app.Description = `This program takes a stream of plane tracking data (location updates) from an AMQP message bus  ` +
		`and filters messages and only returns significant changes for each aircraft.` +
		"\n\n" +
		`example: ./pw_router --rabbitmq="amqp://guest:guest@localhost:5672" --source-route-key=location-updates --num-workers=8 --prom-metrics-port=9601`

	app.Commands = cli.Commands{
		{
			Name:        "daemon",
			Description: "For prod, Logging is JSON formatted",
			Action:      runDaemon,
		},
		{
			Name:        "cli",
			Description: "Runs in your terminal with human readable output",
			Action:      runCli,
		},
	}

	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "rabbitmq",
			Usage:   "Rabbitmq URL for reaching and publishing updates.",
			Value:   "amqp://guest:guest@rabbitmq:5672/pw",
			EnvVars: []string{"RABBITMQ"},
		},
		&cli.StringFlag{
			Name:    "source-route-key",
			Usage:   "Name of the routing key to read location updates from.",
			Value:   "location-updates-enriched",
			EnvVars: []string{"SOURCE_ROUTE_KEY"},
		},
		&cli.StringFlag{
			Name:    "destination-route-key",
			Usage:   "Name of the routing key to publish significant updates to.",
			Value:   "location-updates-enriched-reduced",
			EnvVars: []string{"DEST_ROUTE_KEY"},
		},
		&cli.IntFlag{
			Name:    "num-workers",
			Usage:   "Number of workers to process updates.",
			Value:   4,
			EnvVars: []string{"NUM_WORKERS"},
		},
		&cli.BoolFlag{
			Name:    "spread-updates",
			Usage:   "publish location updates to their respective tileXX_high and tileXX_low routing keys as well",
			EnvVars: []string{"DEBUG"},
		},
		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "Show Extra Debug Information",
			EnvVars: []string{"DEBUG"},
		},
		&cli.BoolFlag{
			Name:    "quiet",
			Usage:   "Only show important messages",
			EnvVars: []string{"QUIET"},
		},
		&cli.BoolFlag{
			Name:  "register-test-queues",
			Usage: "Subscribes a bunch of queues to our routing keys",
		},
	}
	stats.IncludePrometheusFlags(app, 9601)

	app.Before = func(c *cli.Context) error {
		logging.SetVerboseOrQuiet(c.Bool("debug"), c.Bool("quiet"))
		return nil
	}

	if err := app.Run(os.Args); nil != err {
		log.Error().Err(err).Send()
	}
}

func runDaemon(c *cli.Context) error {
	return run(c)
}

func runCli(c *cli.Context) error {
	logging.ConfigureForCli()
	return run(c)
}

func (r *pwRouter) connect(config rabbitmq.Config, timeout time.Duration) error {
	log.Info().Str("host", config.String()).Msg("Connecting to RabbitMQ")
	r.rmq = rabbitmq.New(&config)
	return r.rmq.ConnectAndWait(timeout)
}

func (r *pwRouter) makeQueue(name, bindRouteKey string) error {
	_, err := r.rmq.QueueDeclare(name, 60000) // 60sec TTL
	if nil != err {
		log.Error().Err(err).Msgf("Failed to create queue '%s'", name)
		return err
	}

	if err = r.rmq.QueueBind(name, bindRouteKey, rabbitmq.PlaneWatchExchange); nil != err {
		log.Error().Err(err).Msgf("Failed to QueueBind to route-key:%s to queue %s", bindRouteKey, name)
		return err
	}
	log.Debug().Str("queue", name).Str("route-key", bindRouteKey).Msg("Setup Queue")
	return nil
}

func (r *pwRouter) setupTestQueues() error {
	log.Info().Msg("Setting up test queues")
	// we need a _low and a _high for each tile
	suffixes := []string{qSuffixLow, qSuffixHigh}
	for _, name := range tile_grid.GridLocationNames() {
		for _, suffix := range suffixes {
			if err := r.makeQueue(name+suffix, name+suffix); nil != err {
				return err
			}
		}
	}
	return nil
}

func run(c *cli.Context) error {
	// setup and start the prom exporter
	stats.RunPrometheusWebServer(c)

	var err error
	// connect to rabbitmq, create ourselves 2 queues
	r := pwRouter{
		syncSamples: sync.Map{},
	}

	if "" == c.String("rabbitmq") {
		return errors.New("please specify the --rabbitmq parameter")
	}

	rabbitUrl, err := url.Parse(c.String("rabbitmq"))
	if err != nil {
		return err
	}

	rabbitPassword, _ := rabbitUrl.User.Password()

	rabbitConfig := rabbitmq.Config{
		Host:     rabbitUrl.Hostname(),
		Port:     rabbitUrl.Port(),
		User:     rabbitUrl.User.Username(),
		Password: rabbitPassword,
		Vhost:    rabbitUrl.Path,
		Ssl:      rabbitmq.ConfigSSL{},
	}

	// connect to Rabbit
	if err = r.connect(rabbitConfig, time.Second*5); nil != err {
		return err
	}

	if err = r.makeQueue("reducer-in", c.String("source-route-key")); nil != err {
		return err
	}

	if c.Bool("register-test-queues") {
		if err = r.setupTestQueues(); nil != err {
			return err
		}
	}

	ch, err := r.rmq.Consume("reducer-in", "pw-router")
	if nil != err {
		log.Info().Msg("Failed to consume reducer-in")
		return err
	}

	var wg sync.WaitGroup

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Info().Msgf("Starting with %d workers...", c.Int("num-workers"))
	for i := 0; i < c.Int("num-workers"); i++ {
		wkr := worker{
			rabbit:         &r,
			destRoutingKey: c.String("destination-route-key"),
			spreadUpdates:  c.Bool("spread-updates"),
		}
		wg.Add(1)
		go func() {
			wkr.run(ctx, ch)
			wg.Done()
		}()
	}

	wg.Wait()

	return nil
}