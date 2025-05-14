package timeline

import (
	"reflect"
	"testing"
	"time"

	"github.com/cardinalhq/flutter/pkg/config"
)

func TestDuplicateSpans_Simple(t *testing.T) {
	span := Span{
		Name: "root",
		ResourceAttributes: map[string]any{
			"service.name": "svc",
		},
		Attributes: map[string]any{
			"foo": "bar",
		},
	}
	variant := TraceVariant{Name: "v1"}

	dup := duplicateSpans(span, variant)

	if &dup == &span {
		t.Errorf("duplicateSpans returned the same pointer")
	}
	if !reflect.DeepEqual(dup.ResourceAttributes, span.ResourceAttributes) {
		t.Errorf("ResourceAttributes not equal: got %+v, want %+v", dup.ResourceAttributes, span.ResourceAttributes)
	}
	if !reflect.DeepEqual(dup.Attributes, span.Attributes) {
		t.Errorf("Attributes not equal: got %+v, want %+v", dup.Attributes, span.Attributes)
	}
	// Mutate original, dup should not change
	span.ResourceAttributes["service.name"] = "changed"
	span.Attributes["foo"] = "baz"
	if dup.ResourceAttributes["service.name"] != "svc" {
		t.Errorf("ResourceAttributes not deep copied")
	}
	if dup.Attributes["foo"] != "bar" {
		t.Errorf("Attributes not deep copied")
	}
}

func TestDuplicateSpans_WithChildren(t *testing.T) {
	span := Span{
		Name:               "root",
		ResourceAttributes: map[string]any{"k": "v"},
		Attributes:         map[string]any{"a": 1},
		Children: []Span{
			{
				Name:               "child1",
				ResourceAttributes: map[string]any{"ck": "cv"},
				Attributes:         map[string]any{"ca": 2},
			},
			{
				Name:               "child2",
				ResourceAttributes: map[string]any{"ck2": "cv2"},
				Attributes:         map[string]any{"ca2": 3},
			},
		},
	}
	variant := TraceVariant{Name: "v2"}

	dup := duplicateSpans(span, variant)

	if len(dup.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(dup.Children))
	}
	for i := range dup.Children {
		if &dup.Children[i] == &span.Children[i] {
			t.Errorf("child %d not deep copied", i)
		}
		if !reflect.DeepEqual(dup.Children[i].ResourceAttributes, span.Children[i].ResourceAttributes) {
			t.Errorf("child %d ResourceAttributes not equal", i)
		}
		if !reflect.DeepEqual(dup.Children[i].Attributes, span.Children[i].Attributes) {
			t.Errorf("child %d Attributes not equal", i)
		}
	}
	// Mutate original child, dup should not change
	span.Children[0].ResourceAttributes["ck"] = "changed"
	if dup.Children[0].ResourceAttributes["ck"] != "cv" {
		t.Errorf("child ResourceAttributes not deep copied")
	}
}

func TestApplySpanOverride(t *testing.T) {
	t.Run("Duration", func(t *testing.T) {
		origDuration := config.DurationFromDuration(100 * time.Millisecond)
		overrideDuration := config.DurationFromDuration(200 * time.Millisecond)
		span := &Span{Duration: origDuration}
		override := SpanOverride{Duration: &overrideDuration}

		applySpanOverride(span, override)

		if span.Duration != overrideDuration {
			t.Errorf("expected Duration %d, got %d", overrideDuration, span.Duration)
		}
	})

	t.Run("Error", func(t *testing.T) {
		origError := false
		overrideError := true
		span := &Span{Error: origError}
		override := SpanOverride{Error: &overrideError}

		applySpanOverride(span, override)

		if span.Error != overrideError {
			t.Errorf("expected Error %v, got %v", overrideError, span.Error)
		}
	})

	t.Run("Attributes", func(t *testing.T) {
		span := &Span{Attributes: map[string]any{"foo": "bar", "keep": 1}}
		overrideAttrs := map[string]any{"foo": "baz", "new": 2}
		override := SpanOverride{Attributes: overrideAttrs}

		applySpanOverride(span, override)

		want := map[string]any{"foo": "baz", "keep": 1, "new": 2}
		if !reflect.DeepEqual(span.Attributes, want) {
			t.Errorf("expected Attributes %+v, got %+v", want, span.Attributes)
		}
	})

	t.Run("NilFields", func(t *testing.T) {
		span := &Span{
			Duration:   config.DurationFromDuration(5 * time.Second),
			Error:      false,
			Attributes: map[string]any{"x": 1},
		}
		override := SpanOverride{}

		applySpanOverride(span, override)

		if span.Duration != config.DurationFromDuration(5*time.Second) {
			t.Errorf("Duration changed unexpectedly")
		}
		if span.Error != false {
			t.Errorf("Error changed unexpectedly")
		}
		wantAttrs := map[string]any{"x": 1}
		if !reflect.DeepEqual(span.Attributes, wantAttrs) {
			t.Errorf("Attributes changed unexpectedly")
		}
	})
}
