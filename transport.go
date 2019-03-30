package witness

type notifier struct {
}

func (n *notifier) Notify(rtl RoundTripLog) {
}

func NewTransport(chan bool) *notifier {
	return &notifier{}
}
