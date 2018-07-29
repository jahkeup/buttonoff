package main

import (
	"flag"
	"time"

	butt "code.jahkeup.com/vcastle/buttonoff"
	"github.com/Sirupsen/logrus"
)

var (
	flagInterface  = flag.String("interface", "eth0", "Interface to listen on")
	flagMQTTBroker = flag.String("broker", "tcp://127.0.0.1:1883", "MQTT Broker to publish to")
	flagLogLevel   = flag.String("log", "INFO", "Log level to write out at")
)

func main() {
	flag.Parse()

	config := butt.Config{
		General: butt.GeneralConfig{
			DropUnconfigured: true,
		},
		Listener: butt.ListenerConfig{
			Interface: *flagInterface,
		},
		MQTT: butt.MQTTConfig{
			BrokerAddr: *flagMQTTBroker,
		},
		Buttons: []butt.ButtonConfig{
			{
				ButtonID: "test-button",
				HWAddr:   "fc:a6:67:b1:24:41",
			},
		},
	}

	level := logrus.InfoLevel
	if parsedLevel, err := logrus.ParseLevel(*flagLogLevel); err == nil {
		level = parsedLevel
	} else {
		logrus.Warn("Could not parse provided log level %q, falling back to %s", level)
	}
	butt.SetLogLevel(level)

	publisher, err := butt.NewMQTTPublisher(config.MQTT)
	if err != nil {
		logrus.Fatal(err)
	}
	handler, err := butt.NewDashButtonEventHandler(config.General, config.Buttons, publisher)
	if err != nil {
		logrus.Fatal(err)
	}
	listener, err := butt.NewPCAPListener(config.Listener)
	if err != nil {
		logrus.Fatal(err)
	}

	listener.UseEventHandler(handler)
	time.Sleep(time.Second * 60)
}
