package buttonoff

import (
	"context"
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
	log     logrus.FieldLogger
	capture *pcapCapture

	mu                *sync.Mutex
	pressEventHandler EventHandler
}

func NewPCAPListener(conf ListenerConfig) (*PCAPListener, error) {
	logger := appLogger.WithField("comp", "pcap-listener")

	validateErr := checkListenerConfig(conf, logger)
	if validateErr != nil {
		return nil, validateErr
	}

	pl := &PCAPListener{
		log:               logger,
		capture:           nil,
		mu:                &sync.Mutex{},
		pressEventHandler: nil,
	}

	capture, captureErr := newPCAPCapture(conf, pl.handleEvent)
	if captureErr != nil {
		return nil, captureErr
	}
	pl.capture = capture

	return pl, nil
}

func (pl *PCAPListener) Run(ctx context.Context) error {
	return pl.capture.Run(ctx)
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

type pcapCapture struct {
	log      logrus.FieldLogger
	handle   *pcap.Handle
	callback func(e Event) error
}

func newPCAPCapture(config ListenerConfig, callback func(e Event) error) (*pcapCapture, error) {
	logger := appLogger.WithFields(logrus.Fields{
		"comp":   "pcap-listener.capture",
		"device": config.Interface,
	})

	filteredHandle, err := filteredActiveHandler(config.Interface)
	if err != nil {
		logger.WithField("device", config.Interface).Error(err)
		return nil, err
	}

	return &pcapCapture{
		log:      logger,
		handle:   filteredHandle,
		callback: callback,
	}, nil
}

func (cap *pcapCapture) Run(ctx context.Context) error {
	packetSource := gopacket.NewPacketSource(cap.handle, cap.handle.LinkType())
	packets := packetSource.Packets()

	cap.log.Debug("running capture processor")
	for {
		select {
		case <-ctx.Done():
			return cap.shutdown()
		case packet := <-packets:
			cap.log.Debug("handling captured packet")
			cap.processPacket(packet)
		}
	}
}

func (cap *pcapCapture) shutdown() error {
	cap.log.Debug("closing pcap handle")
	cap.handle.Close()
	return nil
}

func (cap *pcapCapture) processPacket(packet gopacket.Packet) {
	dhcpReqL := packet.Layer(layers.LayerTypeDHCPv4)
	if dhcpReqL == nil {
		cap.log.Warn("Filtered packet recieved but not decode-able.")
		return
	}
	dhcpReq := dhcpReqL.(*layers.DHCPv4)
	cap.log.Debugf("Recieved a DHCP packet: %v", dhcpReq)
	cap.log.Debugf("Packet from client %q", dhcpReq.ClientHWAddr)

	event := Event{
		HWAddr:    dhcpReq.ClientHWAddr.String(),
		Timestamp: time.Now(),
	}

	cap.log.Debugf("Submitting event with listener %v", event)
	cap.callback(event)
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
