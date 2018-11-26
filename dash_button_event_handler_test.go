package buttonoff

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuplicateButtonIDs(t *testing.T) {
	buttonID := "DUP"
	buttons := []ButtonConfig{
		{ButtonID: buttonID, HWAddr: "aa:aa:aa:aa:aa:aa"},
		{ButtonID: buttonID, HWAddr: "bb:bb:bb:bb:bb:bb"},
		{ButtonID: buttonID, HWAddr: "ff:ff:ff:ff:ff:ff"},
	}
	events := []Event{
		{
			HWAddr:    buttons[0].HWAddr,
			Timestamp: time.Now(),
		},
		{
			HWAddr:    buttons[1].HWAddr,
			Timestamp: time.Now(),
		},
		{
			HWAddr:    buttons[2].HWAddr,
			Timestamp: time.Now(),
		},
	}
	handler, err := NewDashButtonEventHandler(GeneralConfig{
		DropUnconfigured:       true,
		PostPressSupressPeriod: time.Second * 1,
		TopicTemplate:          DefaultDashButtonTopicTemplate,
	}, buttons, (Publisher)(nil))
	require.NoError(t, err)
	for _, event := range events {
		id := handler.getRegistryButtonID(event)
		assert.Equal(t, buttonID, id, "expected the button id to be the same for each event")
	}
}
