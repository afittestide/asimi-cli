package main

import (
	"testing"
)

func TestHelpViewerCreation(t *testing.T) {
	viewer := NewHelpViewer()
	if viewer == nil {
		t.Fatal("NewHelpViewer returned nil")
	}

	if viewer.IsVisible() {
		t.Error("New help viewer should not be visible")
	}
}

func TestHelpViewerShowHide(t *testing.T) {
	viewer := NewHelpViewer()

	viewer.Show("index")
	if !viewer.IsVisible() {
		t.Error("Help viewer should be visible after Show()")
	}

	viewer.Hide()
	if viewer.IsVisible() {
		t.Error("Help viewer should not be visible after Hide()")
	}
}

func TestHelpViewerTopics(t *testing.T) {
	viewer := NewHelpViewer()

	topics := []string{
		"index",
		"modes",
		"commands",
		"navigation",
		"editing",
		"files",
		"sessions",
		"context",
		"config",
		"quickref",
	}

	for _, topic := range topics {
		viewer.Show(topic)
		content := viewer.renderHelpContent(topic)
		if content == "" {
			t.Errorf("Help content for topic '%s' is empty", topic)
		}
		if len(content) < 50 {
			t.Errorf("Help content for topic '%s' seems too short: %d chars", topic, len(content))
		}
	}
}

func TestHelpViewerUnknownTopic(t *testing.T) {
	viewer := NewHelpViewer()
	viewer.Show("nonexistent-topic")
	content := viewer.renderHelpContent("nonexistent-topic")

	if content == "" {
		t.Error("Help viewer should return content for unknown topics")
	}

	// Should contain "not found" message
	if len(content) < 50 {
		t.Error("Unknown topic help should provide guidance")
	}
}
