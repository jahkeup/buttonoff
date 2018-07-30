package buttonoff

import (
	"time"
)

type Config struct {
	General  GeneralConfig
	Listener ListenerConfig
	MQTT     MQTTConfig
	Buttons  []ButtonConfig
}

type GeneralConfig struct {
	TopicTemplate          string
	DropUnconfigured       bool
	PostPressSupressPeriod time.Duration
}

type ListenerConfig struct {
	Interface string
}

type MQTTConfig struct {
	BrokerAddr string
	Username   string
	Password   string
	// TODO: Support Certificate authentication
	// Certificate string
}

type ButtonConfig struct {
	ButtonID string
	HWAddr   string
}
