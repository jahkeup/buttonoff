package buttonoff

import (
	"bytes"
	"encoding/json"
	"sync"
	"text/template"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	DefaultDashButtonTopicTemplate          = "/buttonoff/{{.ButtonID}}/pressed"
	DefaultDashButtonPostPressSupressPeriod = time.Millisecond * 600
)

type DashButtonEventHandler struct {
	log           logrus.FieldLogger
	onlyKnown     bool
	topicTemplate *template.Template
	publisher     Publisher

	buttonRegistryLock *sync.Mutex
	buttonRegistry     map[string]dashButton

	limiter Accepter
}

func NewDashButtonEventHandler(general GeneralConfig, buttons []ButtonConfig, publisher Publisher) (*DashButtonEventHandler, error) {
	logger := appLogger.WithField("comp", "event-handler")

	if general.TopicTemplate == "" {
		general.TopicTemplate = DefaultDashButtonTopicTemplate
	}
	t, err := template.New("topic-format").Parse(general.TopicTemplate)
	if err != nil {
		panic(err)
	}

	buttonRegistry, buildErr := buildButtonMap(buttons)
	if buildErr != nil {
		return nil, buildErr
	}

	if general.PostPressSupressPeriod == 0 {
		fallback := DefaultDashButtonPostPressSupressPeriod
		logger.Warnf("No PostPressSupressPeriod defined, falling back to %s",
			fallback)
		general.PostPressSupressPeriod = fallback
	}

	limiter := NewPressRateLimiter(general.PostPressSupressPeriod)
	logger.Debugf("Suppressing duplicate events for %s after press", general.PostPressSupressPeriod)

	eh := &DashButtonEventHandler{
		log:                logger,
		onlyKnown:          general.DropUnconfigured,
		topicTemplate:      t,
		publisher:          publisher,
		buttonRegistryLock: &sync.Mutex{},
		buttonRegistry:     buttonRegistry,
		limiter:            limiter,
	}

	return eh, err
}

func (d *DashButtonEventHandler) HandleEvent(e Event) error {
	if d.shouldAcceptEvent(e) {
		d.log.Debugf("Handling accepted event: %v", e)
		return d.publish(e)
	}
	d.log.Debugf("Dropping unacceptable event: %v", e)
	return nil
}

func (d *DashButtonEventHandler) publish(e Event) error {
	buttonID := d.getRegistryButtonID(e)

	payload := messagePayload{
		ButtonID:  buttonID,
		Timestamp: e.Timestamp.Format(time.RFC3339Nano),
	}

	topic := bytes.NewBuffer(nil)
	err := d.topicTemplate.Execute(topic, payload)
	if err != nil {
		return errors.Wrapf(err, "could not format topic for event: %v", e)
	}

	msg := Message{
		Topic:   topic.String(),
		Payload: payload.ToJSONBytes(),
	}
	d.log.WithFields(logrus.Fields{
		"button-id": buttonID,
		"topic":     topic,
	}).Debug("Publishing message for event")
	return d.publisher.Publish(msg)
}

func (d *DashButtonEventHandler) shouldAcceptEvent(e Event) bool {
	log := d.log.WithField("limit-key", e.HWAddr)
	shouldAccept := d.limiter.Accept(e.HWAddr)
	if shouldAccept {
		log.Debug("Accepting event")
	} else {
		log.Debug("Rejecting event")
	}
	return shouldAccept
}

func (d *DashButtonEventHandler) getRegistryButtonID(e Event) string {
	d.buttonRegistryLock.Lock()
	var id string

	hwaddr := e.HWAddr
	if butt, ok := d.buttonRegistry[hwaddr]; ok {
		id = butt.ButtonID
	} else {
		id = e.HWAddr
		d.buttonRegistry[hwaddr] = dashButton{
			ButtonID: id,
			HWAddr:   hwaddr,
		}
	}
	d.buttonRegistryLock.Unlock()
	return id
}

type dashButton struct {
	ButtonID string
	HWAddr   string
}

type messagePayload struct {
	ButtonID  string `json:"button_id"`
	Timestamp string `json:"timestamp"`
}

func (payload *messagePayload) ToJSONBytes() []byte {
	payloadJSON, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		logrus.Error(errors.Wrapf(marshalErr, "could not marshal message payload %v", payload))
		logrus.Debugf("Falling back to ButtonID as the message payload: %q", payload.ButtonID)
		return []byte(payload.ButtonID)
	}
	return payloadJSON
}

func buildButtonMap(buttons []ButtonConfig) (map[string]dashButton, error) {
	dashButtons := make(map[string]dashButton)
	for i, butt := range buttons {
		hwaddr := butt.HWAddr
		if hwaddr == "" {
			return nil, errors.Errorf("config for button %d is missing HWAddr, add it to continue", i)
		}

		id := butt.ButtonID
		if id == "" {
			id = hwaddr
			logrus.Debugf("No ButtonID provided for button %d (%q), using %q", i, hwaddr, id)
		}

		translated := dashButton{
			ButtonID: id,
			HWAddr:   hwaddr,
		}

		dashButtons[hwaddr] = translated
	}

	return dashButtons, nil
}
