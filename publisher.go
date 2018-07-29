package buttonoff

type Publisher interface {
	Publish(msg Message) error
}

type Message struct {
	Topic   string
	Payload []byte
}
