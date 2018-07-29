package buttonoff

type Publisher interface {
	Publish(msg Message) error
	Close() error
}

type Message struct {
	Topic   string
	Payload []byte
}
