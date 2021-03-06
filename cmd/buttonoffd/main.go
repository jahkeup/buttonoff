package main

import (
	"context"
	"flag"
	"sync"

	butt "github.com/jahkeup/buttonoff"
	"github.com/sirupsen/logrus"
)

var (
	flagInterface   = flag.String("interface", "", "Interface name to listen on")
	flagMQTTBroker  = flag.String("broker", "", `MQTT Broker to publish to (ex: "tcp://127.0.0.1:1883")`)
	flagLogLevel    = flag.String("log-level", "INFO", "Log level to write out at")
	flagConfig      = flag.String("config", "./buttonoff.toml", "Configuration file")
	flagWriteConfig = flag.Bool("write-default", false, "Write default config to the -config path")
)

func main() {
	flag.Parse()
	logger := logrus.WithField("comp", "buttonoffd")

	level := logrus.InfoLevel
	if parsedLevel, err := logrus.ParseLevel(*flagLogLevel); err == nil {
		level = parsedLevel
	} else {
		logger.Warnf("Could not parse provided log level %q, falling back to %s", *flagLogLevel, level)
	}
	logrus.SetLevel(level)
	butt.SetLogLevel(level)

	if *flagWriteConfig {
		wrErr := writeDefaultConfig(*flagConfig)
		if wrErr != nil {
			logger.Fatal(wrErr)
		}
		return
	}

	config, err := LoadConfig(*flagConfig)
	if err != nil {
		logger.Fatalf("Could not load config from file %q", *flagConfig)
		return
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		for _, button := range config.Buttons {
			logger.WithFields(logrus.Fields{
				"HWAddr":   button.HWAddr,
				"ButtonID": button.ButtonID,
			}).Debug("configured")
		}
	}

	if *flagInterface != "" {
		config.Listener.Interface = *flagInterface
	}
	if *flagMQTTBroker != "" {
		config.MQTT.BrokerAddr = *flagMQTTBroker
	}

	publisher, err := butt.NewMQTTPublisher(config.MQTT)
	if err != nil {
		logger.Fatal(err)
	}
	handler, err := butt.NewDashButtonEventHandler(config.General, config.Buttons, publisher)
	if err != nil {
		publisher.Close()
		logger.Fatal(err)
	}
	listener, err := butt.NewPCAPListener(config.Listener)
	if err != nil {
		publisher.Close()
		logger.Fatal(err)
	}
	listener.UseEventHandler(handler)

	// Claiming ignorance to all failures.. keep running. Internally,
	// these things will panic where we can't recover and otherwise
	// retry indefinitely.
	ctx := context.TODO()
	runAll(ctx, listener, publisher)
}

type runnable interface {
	Run(ctx context.Context) error
}

func runAll(ctx context.Context, runnables ...runnable) {
	var wg sync.WaitGroup
	for i := range runnables {
		runner := runnables[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			runner.Run(ctx)
		}()
	}
	wg.Wait()
}
