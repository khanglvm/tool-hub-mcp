package learning

import (
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	mockStore := newMockStorage()
	tracker := NewTracker(mockStore)

	if tracker == nil {
		t.Fatal("NewTracker returned nil")
	}

	if !tracker.IsEnabled() {
		t.Error("expected tracker to be enabled")
	}

	// Clean up
	tracker.Stop()
}

func TestTracker_Track(t *testing.T) {
	mockStore := newMockStorage()
	tracker := NewTracker(mockStore)
	defer tracker.Stop()

	event := UsageEvent{
		ToolName:  "test_tool",
		Timestamp: time.Now(),
	}

	// Track should not block
	tracker.Track(event)

	// Give time for background processing
	time.Sleep(100 * time.Millisecond)

	// Verify event was recorded
	history, err := mockStore.GetUsageHistory("test_tool", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) == 0 {
		t.Error("expected event to be recorded")
	}
}

func TestTracker_TrackMultiple(t *testing.T) {
	mockStore := newMockStorage()
	tracker := NewTracker(mockStore)
	defer tracker.Stop()

	// Track multiple events
	for i := 0; i < 10; i++ {
		event := UsageEvent{
			ToolName:  "test_tool",
			Timestamp: time.Now(),
		}
		tracker.Track(event)
	}

	// Give time for background processing
	time.Sleep(200 * time.Millisecond)

	// Verify all events were recorded
	history, err := mockStore.GetUsageHistory("test_tool", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 10 {
		t.Errorf("expected 10 events, got %d", len(history))
	}
}

func TestTracker_Disable(t *testing.T) {
	mockStore := newMockStorage()
	tracker := NewTracker(mockStore)
	defer tracker.Stop()

	tracker.Disable()

	if tracker.IsEnabled() {
		t.Error("expected tracker to be disabled")
	}

	event := UsageEvent{
		ToolName:  "test_tool",
		Timestamp: time.Now(),
	}

	tracker.Track(event)

	// Give time for background processing
	time.Sleep(100 * time.Millisecond)

	// Verify event was NOT recorded
	history, err := mockStore.GetUsageHistory("test_tool", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 0 {
		t.Error("expected no events when disabled")
	}
}

func TestTracker_Enable(t *testing.T) {
	mockStore := newMockStorage()
	tracker := NewTracker(mockStore)
	defer tracker.Stop()

	tracker.Disable()
	tracker.Enable()

	if !tracker.IsEnabled() {
		t.Error("expected tracker to be enabled")
	}

	event := UsageEvent{
		ToolName:  "test_tool",
		Timestamp: time.Now(),
	}

	tracker.Track(event)

	// Give time for background processing
	time.Sleep(100 * time.Millisecond)

	// Verify event was recorded
	history, err := mockStore.GetUsageHistory("test_tool", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) == 0 {
		t.Error("expected event to be recorded when enabled")
	}
}

func TestTracker_Stop(t *testing.T) {
	mockStore := newMockStorage()
	tracker := NewTracker(mockStore)

	// Track some events
	for i := 0; i < 5; i++ {
		event := UsageEvent{
			ToolName:  "test_tool",
			Timestamp: time.Now(),
		}
		tracker.Track(event)
	}

	// Stop should flush remaining events
	tracker.Stop()

	// Verify all events were recorded
	history, err := mockStore.GetUsageHistory("test_tool", time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("failed to get history: %v", err)
	}

	if len(history) != 5 {
		t.Errorf("expected 5 events after stop, got %d", len(history))
	}
}

func TestTracker_TrackNonBlocking(t *testing.T) {
	mockStore := newMockStorage()
	tracker := NewTracker(mockStore)
	defer tracker.Stop()

	start := time.Now()

	// Track many events quickly (should not block)
	for i := 0; i < 1000; i++ {
		event := UsageEvent{
			ToolName:  "test_tool",
			Timestamp: time.Now(),
		}
		tracker.Track(event)
	}

	elapsed := time.Since(start)

	// Should complete in <100ms (non-blocking)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Track is blocking: took %v, expected <100ms", elapsed)
	}
}

func TestTracker_TrackQueueFull(t *testing.T) {
	mockStore := newMockStorage()
	tracker := NewTracker(mockStore)
	defer tracker.Stop()

	// Disable storage to cause queue backup
	tracker.Disable()

	// Fill the queue
	for i := 0; i < eventQueueSize+100; i++ {
		event := UsageEvent{
			ToolName:  "test_tool",
			Timestamp: time.Now(),
		}
		tracker.Track(event) // Should not block even when full
	}

	// Queue size should be at capacity
	size := tracker.GetEventQueueSize()
	if size > eventQueueSize {
		t.Errorf("queue size %d exceeds capacity %d", size, eventQueueSize)
	}
}

func TestTracker_GetEventQueueSize(t *testing.T) {
	mockStore := newMockStorage()
	tracker := NewTracker(mockStore)
	defer tracker.Stop()

	// Initially empty
	size := tracker.GetEventQueueSize()
	if size != 0 {
		t.Errorf("expected queue size 0, got %d", size)
	}

	// Add some events
	for i := 0; i < 10; i++ {
		event := UsageEvent{
			ToolName:  "test_tool",
			Timestamp: time.Now(),
		}
		tracker.Track(event)
	}

	// Size should increase (but may have started processing)
	size = tracker.GetEventQueueSize()
	if size == 0 {
		// Events may have been processed already, which is ok
		t.Log("Events processed immediately (acceptable)")
	}
}

func TestTracker_StorageError(t *testing.T) {
	errorStore := &errorMockStorage{}
	tracker := NewTracker(errorStore)
	defer tracker.Stop()

	event := UsageEvent{
		ToolName:  "test_tool",
		Timestamp: time.Now(),
	}

	// Should not panic even if storage fails
	tracker.Track(event)

	// Give time for background processing
	time.Sleep(100 * time.Millisecond)

	// Tracker should still be running
	if !tracker.IsEnabled() {
		t.Error("expected tracker to remain enabled after storage error")
	}
}

func TestUsageEvent_ToStorage(t *testing.T) {
	event := UsageEvent{
		ToolName:        "test_tool",
		ContextHash:     "test-context-hash",
		SearchID:        "test-search-id",
		Selected:        true,
		Timestamp:       time.Now(),
		Rating:          5,
		WasRecommended:  true,
	}

	storageEvent := event.ToStorage()

	if storageEvent.ToolName != event.ToolName {
		t.Errorf("expected ToolName %s, got %s", event.ToolName, storageEvent.ToolName)
	}

	if storageEvent.ContextHash != event.ContextHash {
		t.Errorf("expected ContextHash %s, got %s", event.ContextHash, storageEvent.ContextHash)
	}

	if storageEvent.Selected != event.Selected {
		t.Errorf("expected Selected %v, got %v", event.Selected, storageEvent.Selected)
	}

	if storageEvent.Rating != event.Rating {
		t.Errorf("expected Rating %d, got %d", event.Rating, storageEvent.Rating)
	}

	if storageEvent.WasRecommended != event.WasRecommended {
		t.Errorf("expected WasRecommended %v, got %v", event.WasRecommended, storageEvent.WasRecommended)
	}
}
