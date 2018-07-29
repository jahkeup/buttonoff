package buttonoff

import (
	"net/url"
	"time"

	"github.com/Sirupsen/logrus"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

const (
	mqttClientID             = "buttonoff"
	mqttPublishTimeout       = time.Second * 5
	mqttConnectTimeout       = time.Second * 30
	mqttMaxReconnectInterval = time.Minute * 5
)

var (
	MQTTPublishTimeoutErr = errors.Errorf("MQTT Publish timeout after %s", mqttPublishTimeout)
	MQTTConnectTimeoutErr = errors.Errorf("MQTT Connect timeout after %s", mqttConnectTimeout)
)

type mqttPublisher interface {
	Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token
}

type MQTTPublisher struct {
	log  logrus.FieldLogger
	mqtt mqttPublisher
}

func NewMQTTPublisher(conf MQTTConfig) (*MQTTPublisher, error) {
	logger := appLogger.WithField("comp", "mqtt-publisher")

	options, err := clientOptionsFromConfig(conf)
	if err != nil {
		return nil, err
	}
	client := mqtt.NewClient(options)
	connectToken := client.Connect()
	logger.Debug("Connecting to MQTT broker")
	complete := connectToken.WaitTimeout(mqttConnectTimeout)
	if !complete {
		return nil, MQTTConnectTimeoutErr
	}
	connectErr := connectToken.Error()
	if connectErr != nil {
		return nil, connectErr
	}
	logger.Debug("Connected to MQTT broker")

	pub := &MQTTPublisher{
		log:  logger,
		mqtt: client,
	}

	return pub, nil
}

func (mp *MQTTPublisher) Publish(msg Message) error {
	token := mp.mqtt.Publish(msg.Topic, 0, false, msg.Payload)
	complete := token.WaitTimeout(mqttPublishTimeout)
	if !complete {
		return MQTTPublishTimeoutErr
	}
	publishErr := token.Error()
	if publishErr != nil {
		mp.log.Error(errors.Wrapf(publishErr, "could not publish message to %s", msg.Topic))
	}
	return publishErr
}

func clientOptionsFromConfig(conf MQTTConfig) (*mqtt.ClientOptions, error) {
	opts := mqtt.NewClientOptions().
		SetAutoReconnect(true).
		SetClientID(mqttClientID).
		SetMaxReconnectInterval(mqttMaxReconnectInterval)

	opts.SetUsername(conf.Username)
	opts.SetPassword(conf.Password)

	_, parseErr := url.Parse(conf.BrokerAddr)
	if parseErr != nil {
		return nil, errors.Wrap(parseErr, "BrokerAddr must be in the form tcp://127.0.0.1:1883")
	}

	opts.AddBroker(conf.BrokerAddr)

	return opts, nil
}
