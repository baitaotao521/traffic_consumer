package urlmanager

import (
	"testing"

	"github.com/baitaotao521/traffic_consumer/internal/model"
)

func TestMarkInvalidEmitsEventAndRemovesFromSelection(t *testing.T) {
	var event model.Event
	manager := New([]string{"https://example.com/a"}, "round_robin", 5, func(payload model.Event) {
		event = payload
	})

	if got := manager.GetURLForThread(1); got != "https://example.com/a" {
		t.Fatalf("unexpected selected URL: %s", got)
	}
	allInvalid := manager.MarkInvalid("https://example.com/a", "boom")
	if !allInvalid {
		t.Fatalf("single URL should make all_invalid true")
	}
	if event.Type != "download_failure" || !event.AllInvalid || event.Error != "boom" {
		t.Fatalf("unexpected event: %+v", event)
	}
	if got := manager.GetURLForThread(1); got != "" {
		t.Fatalf("invalid URL should no longer be selected: %s", got)
	}
}

func TestRemoveURLUpdatesThreadStatus(t *testing.T) {
	manager := New([]string{"a", "b"}, "round_robin", 5, nil)
	if got := manager.GetURLForThread(1); got != "a" {
		t.Fatalf("unexpected selected URL: %s", got)
	}
	if !manager.RemoveURL("a") {
		t.Fatalf("expected URL removal")
	}
	snapshot := manager.ThreadSnapshot(1)
	if snapshot[1] != "a (已移除)" {
		t.Fatalf("thread status not updated: %+v", snapshot)
	}
}
