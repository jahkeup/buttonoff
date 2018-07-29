package buttonoff

import (
	"bytes"
	"encoding/json"
	"sync"
	"text/template"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

const (
	DefaultDashButtonTopicTemplate = "/buttonoff/{{.ButtonID}}/pressed"
)

type DashButtonEventHandler struct {
	log           logrus.FieldLogger
	onlyKnown     bool
	topicTemplate *template.Template
	publisher     Publisher

	buttonRegistryLock sync.Mutex
	buttonRegistry     map[string]dashButton
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

	eh := &DashButtonEventHandler{
		log:            logger,
		onlyKnown:      general.DropUnconfigured,
		topicTemplate:  t,
		publisher:      publisher,
		buttonRegistry: buttonRegistry,
	}

	return eh, err
}

func (d *DashButtonEventHandler) HandleEvent(e Event) error {
	if !d.shouldAcceptEvent(e) {
		d.log.Debugf("Dropping unacceptable event: %v", e)
		return nil
	}
	return d.publish(e)
}

func (d *DashButtonEventHandler) publish(e Event) error {
	buttonID := d.getRegistryButtonID(e)

	payload := messagePayload{
		ButtonID:  buttonID,
		Timestamp: e.Timestamp.Format(time.RFC3339),
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

	return d.publisher.Publish(msg)
}

func (d *DashButtonEventHandler) shouldAcceptEvent(e Event) bool {
	// TODO: filter events here based on configuration and limit
	return true
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
		logrus.Debug("Falling back to ButtonID as the message payload: %q", payload.ButtonID)
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
			logrus.Debugf("no ButtonID provided for button %d (%q), using %q", i, hwaddr, id)
		}

		translated := dashButton{
			ButtonID: id,
			HWAddr:   hwaddr,
		}

		dashButtons[hwaddr] = translated
	}

	return dashButtons, nil
}
