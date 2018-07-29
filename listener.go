package buttonoff

import (
	"net"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/pkg/errors"
)

const (
	pcapBOOTPFilter = "(port 67 or port 68)"
)

type Listener interface {
	UseEventHandler(eh EventHandler) error
}

type PCAPListener struct {
	log logrus.FieldLogger

	mu                *sync.Mutex
	pressEventHandler EventHandler
}

func NewPCAPListener(conf ListenerConfig) (*PCAPListener, error) {
	logger := appLogger.WithField("comp", "pcap-listener")

	validateErr := checkListenerConfig(conf, logger)
	if validateErr != nil {
		return nil, validateErr
	}

	filteredHandle, err := filteredActiveHandler(conf.Interface)
	if err != nil {
		logger.WithField("device", conf.Interface).Error(err)
		return nil, err
	}

	filterLog := logger.WithField("comp", "pcap-listener.decoder")

	pl := &PCAPListener{
		log: logger,

		mu:                &sync.Mutex{},
		pressEventHandler: nil,
	}

	go func() {
		packetSource := gopacket.NewPacketSource(filteredHandle, filteredHandle.LinkType())
		for packet := range packetSource.Packets() {
			dhcpReqL := packet.Layer(layers.LayerTypeDHCPv4)
			if dhcpReqL == nil {
				filterLog.Warn("Filtered packet recieved but not decode-able.")
				continue
			}
			dhcpReq := dhcpReqL.(*layers.DHCPv4)
			filterLog.Debugf("Recieved a DHCP packet: %v", dhcpReq)
			filterLog.Debugf("Packet from client %q", dhcpReq.ClientHWAddr)

			event := Event{
				HWAddr:    dhcpReq.ClientHWAddr.String(),
				Timestamp: time.Now(),
			}

			filterLog.Debugf("Submitting event with listener %v", event)
			pl.handleEvent(event)
		}
	}()

	return pl, nil
}

func (pl *PCAPListener) UseEventHandler(eh EventHandler) error {
	pl.mu.Lock()
	pl.log.Debug("Setting event handler")
	pl.pressEventHandler = eh
	pl.mu.Unlock()
	return nil
}

func (pl *PCAPListener) handleEvent(e Event) error {
	pl.mu.Lock()
	if pl.pressEventHandler != nil {
		pl.log.Debug("Invoking EventHandler for event")
		pl.pressEventHandler.HandleEvent(e)
	} else {
		pl.log.Warn("No handler for event")
	}
	pl.mu.Unlock()
	return nil
}

func filteredActiveHandler(device string) (*pcap.Handle, error) {
	inactive, err := pcap.NewInactiveHandle(device)
	if err != nil {
		return nil, errors.Wrapf(err, "could not get inactive handle")
	}
	promiscErr := inactive.SetPromisc(true) // Listening for BOOTP broadcast packets
	if promiscErr != nil {
		return nil, errors.Wrap(err, "could not set promiscuous listen")
	}

	active, err := inactive.Activate()
	if err != nil {
		return nil, errors.Wrap(err, "could not activate handle")
	}

	filterErr := active.SetBPFFilter(pcapBOOTPFilter)
	if filterErr != nil {
		active.Close()
		return nil, errors.Wrapf(err, "could not set filter")
	}

	return active, nil
}

func checkListenerConfig(conf ListenerConfig, log logrus.FieldLogger) error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return errors.Wrap(err, "enumeration of network devices failed")
	}

	for _, iface := range ifaces {
		if iface.Name == conf.Interface {
			log.Debugf("Found device in enumerated network interfaces: %s", iface.Name)
			break
		}
	}

	return nil
}
