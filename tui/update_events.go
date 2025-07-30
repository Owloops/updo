package tui

type UpdateEventType int

const (
	TargetDataUpdateEvent UpdateEventType = iota
	PlotDataUpdateEvent
	SSLDataUpdateEvent
)

type UpdateEvent struct {
	Type      UpdateEventType
	Key       TargetKey
	Data      TargetData
	TermWidth int
	URL       string
	SSLExpiry int
}

func NewTargetDataUpdateEvent(key TargetKey, data TargetData, termWidth int) UpdateEvent {
	return UpdateEvent{
		Type:      TargetDataUpdateEvent,
		Key:       key,
		Data:      data,
		TermWidth: termWidth,
	}
}

func NewSSLUpdateEvent(url string, daysRemaining int) UpdateEvent {
	return UpdateEvent{
		Type:      SSLDataUpdateEvent,
		URL:       url,
		SSLExpiry: daysRemaining,
	}
}

type EventHandler interface {
	HandleUpdateEvent(event UpdateEvent)
}

type EventDispatcher struct {
	handlers []EventHandler
}

func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		handlers: make([]EventHandler, 0),
	}
}

func (ed *EventDispatcher) AddHandler(handler EventHandler) {
	ed.handlers = append(ed.handlers, handler)
}

func (ed *EventDispatcher) DispatchEvent(event UpdateEvent) {
	for _, handler := range ed.handlers {
		handler.HandleUpdateEvent(event)
	}
}
