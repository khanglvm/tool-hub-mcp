package learning

import (
	"log"
	"sync"
	"time"

	"github.com/khanglvm/tool-hub-mcp/internal/storage"
)

const (
	// eventQueueSize is the buffer size for the event queue.
	// If full, events are dropped (non-blocking).
	eventQueueSize = 1000

	// batchFlushSize is the number of events that triggers an immediate flush.
	// Smaller value means more frequent flushes (better for tests).
	batchFlushSize = 10

	// batchFlushInterval is how often to flush events to disk.
	batchFlushInterval = 5 * time.Minute

	// aggressiveFlushInterval is how often to flush when events are pending.
	// Shorter interval ensures better responsiveness for tests.
	aggressiveFlushInterval = 50 * time.Millisecond
)

// Tracker tracks tool usage in the background with non-blocking writes.
type Tracker struct {
	storage    storage.Storage
	eventQueue chan UsageEvent
	stopChan   chan struct{}
	stopOnce   sync.Once
	wg         sync.WaitGroup
	enabled    bool
	mu         sync.RWMutex
}

// NewTracker creates a new usage tracker with background processing.
func NewTracker(s storage.Storage) *Tracker {
	t := &Tracker{
		storage:    s,
		eventQueue: make(chan UsageEvent, eventQueueSize),
		stopChan:   make(chan struct{}),
		enabled:    true,
	}

	// Initialize storage
	if err := t.storage.Init(); err != nil {
		log.Printf("Warning: learning storage initialization failed: %v", err)
		t.enabled = false
	}

	t.wg.Add(1)
	go t.processEvents()

	return t
}

// Track records a tool usage event (non-blocking).
// If the queue is full, the event is dropped and a warning is logged.
func (t *Tracker) Track(event UsageEvent) {
	if !t.isEnabled() {
		return
	}

	select {
	case t.eventQueue <- event:
		// Event queued successfully
	default:
		log.Printf("Warning: learning queue full, dropping event for tool: %s", event.ToolName)
	}
}

// Stop gracefully shuts down the tracker, flushing remaining events.
func (t *Tracker) Stop() {
	t.stopOnce.Do(func() {
		close(t.stopChan)
		t.wg.Wait()
	})
}

// Disable disables tracking (events are ignored).
func (t *Tracker) Disable() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = false
}

// Enable enables tracking.
func (t *Tracker) Enable() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.enabled = true
}

// IsEnabled returns whether tracking is enabled.
func (t *Tracker) IsEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.enabled
}

// isEnabled is an internal helper (no lock for use within locked methods).
func (t *Tracker) isEnabled() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.enabled && t.storage != nil
}

// processEvents runs in the background, batching and flushing events.
func (t *Tracker) processEvents() {
	defer t.wg.Done()

	ticker := time.NewTicker(aggressiveFlushInterval)
	defer ticker.Stop()

	batch := make([]UsageEvent, 0, batchFlushSize)

	for {
		select {
		case event, ok := <-t.eventQueue:
			if !ok {
				// Channel closed, flush remaining and exit
				t.flush(batch)
				return
			}

			batch = append(batch, event)

			// Flush immediately if batch is full
			if len(batch) >= batchFlushSize {
				t.flush(batch)
				batch = make([]UsageEvent, 0, batchFlushSize)
			}

		case <-ticker.C:
			// Periodic flush (more aggressive)
			if len(batch) > 0 {
				t.flush(batch)
				batch = make([]UsageEvent, 0, batchFlushSize)
			}

		case <-t.stopChan:
			// Stop signal: drain remaining events from channel, flush and exit
			// Keep draining until channel is empty or closed
			for {
				select {
				case event, ok := <-t.eventQueue:
					if !ok {
						// Channel closed, flush and exit
						t.flush(batch)
						return
					}
					batch = append(batch, event)
					// Flush if batch gets full
					if len(batch) >= batchFlushSize {
						t.flush(batch)
						batch = make([]UsageEvent, 0, batchFlushSize)
					}
				default:
					// No more events in channel, flush remaining and exit
					t.flush(batch)
					return
				}
			}
		}
	}
}

// flush writes a batch of events to storage.
func (t *Tracker) flush(events []UsageEvent) {
	if len(events) == 0 {
		return
	}

	for _, event := range events {
		storageEvent := event.ToStorage()
		if err := t.storage.RecordUsage(storageEvent); err != nil {
			log.Printf("Warning: failed to record usage: %v", err)
		}
	}
}

// GetEventQueueSize returns the current number of events in the queue.
// Useful for monitoring queue health.
func (t *Tracker) GetEventQueueSize() int {
	return len(t.eventQueue)
}
