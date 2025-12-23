package watchdog

import (
	"time"
)

// Event 는 워치독 상태 변경과 관리 동작을 기록한다.
type Event struct {
	At          time.Time `json:"at"`
	Action      string    `json:"action"`
	Container   string    `json:"container,omitempty"`
	By          string    `json:"by,omitempty"`          // auto | manual
	RequestedBy string    `json:"requestedBy,omitempty"` // Cloudflare Access email
	Reason      string    `json:"reason,omitempty"`
	Result      string    `json:"result,omitempty"` // initiated | ok | failed | skipped
	Error       string    `json:"error,omitempty"`
}

const watchdogEventBufferSize = 200

func (w *Watchdog) appendEvent(evt Event) {
	if evt.At.IsZero() {
		evt.At = time.Now()
	}

	w.eventsMu.Lock()
	defer w.eventsMu.Unlock()

	w.events = append(w.events, evt)
	if len(w.events) <= watchdogEventBufferSize {
		return
	}

	excess := len(w.events) - watchdogEventBufferSize
	copy(w.events, w.events[excess:])
	w.events = w.events[:watchdogEventBufferSize]
}

// SnapshotEvents 는 동작을 수행한다.
func (w *Watchdog) SnapshotEvents(limit int) []Event {
	if limit <= 0 || limit > watchdogEventBufferSize {
		limit = watchdogEventBufferSize
	}

	w.eventsMu.Lock()
	defer w.eventsMu.Unlock()

	if len(w.events) == 0 {
		return nil
	}

	if limit > len(w.events) {
		limit = len(w.events)
	}

	out := make([]Event, limit)
	start := len(w.events) - limit
	copy(out, w.events[start:])
	return out
}
