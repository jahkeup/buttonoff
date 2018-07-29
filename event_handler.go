package buttonoff

type EventHandler interface {
	HandleEvent(e Event) error
}
