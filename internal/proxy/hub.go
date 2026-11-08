package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// RequestEvent represents a logged proxy transaction for real-time visualization.
type RequestEvent struct {
	ID          uint64            `json:"id"`
	Timestamp   string            `json:"timestamp"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	ClientIP    string            `json:"client_ip"`
	Simulated   bool              `json:"simulated"`
	CountryCode string            `json:"country_code"`
	CountryName string            `json:"country_name"`
	City        string            `json:"city"`
	StatusCode  int               `json:"status_code"`
	DurationMs  float64           `json:"duration_ms"`
	Headers     map[string]string `json:"headers"`
}

// EventHub manages live SSE subscribers.
type EventHub struct {
	mu          sync.RWMutex
	subscribers map[chan *RequestEvent]struct{}
	counter     uint64
	history     []*RequestEvent
	maxHistory  int
}

// NewEventHub creates an event broadcaster.
func NewEventHub(maxHistory int) *EventHub {
	if maxHistory <= 0 {
		maxHistory = 100
	}
	return &EventHub{
		subscribers: make(map[chan *RequestEvent]struct{}),
		history:     make([]*RequestEvent, 0, maxHistory),
		maxHistory:  maxHistory,
	}
}

// Publish broadcasts an event to all connected listeners.
func (h *EventHub) Publish(evt *RequestEvent) {
	h.mu.Lock()
	h.counter++
	evt.ID = h.counter
	evt.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)

	// Keep rolling history buffer
	if len(h.history) >= h.maxHistory {
		h.history = h.history[1:]
	}
	h.history = append(h.history, evt)

	for ch := range h.subscribers {
		select {
		case ch <- evt:
		default:
			// Non-blocking drop if channel full
		}
	}
	h.mu.Unlock()
}

// RecentHistory returns a copy of recently captured events.
func (h *EventHub) RecentHistory() []*RequestEvent {
	h.mu.RLock()
	defer h.mu.RUnlock()

	res := make([]*RequestEvent, len(h.history))
	copy(res, h.history)
	return res
}

// ServeSSE handles the Server-Sent Events HTTP connection.
func (h *EventHub) ServeSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan *RequestEvent, 64)

	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	history := make([]*RequestEvent, len(h.history))
	copy(history, h.history)
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.subscribers, ch)
		close(ch)
		h.mu.Unlock()
	}()

	// Flush initial history events
	for _, evt := range history {
		data, _ := json.Marshal(evt)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}
	flusher.Flush()

	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			// Keep-alive ping
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case evt, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err == nil {
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}
