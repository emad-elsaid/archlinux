package fest

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"time"
)

// TestPrettyHandler_WithAttrs_SliceAliasing demonstrates a critical slice aliasing bug.
// When WithAttrs is called, it uses append(h.attrs, attrs...) which can mutate the
// original handler's slice if it has sufficient capacity.
func TestPrettyHandler_WithAttrs_SliceAliasing(t *testing.T) {
	var buf bytes.Buffer
	
	// Create base handler with capacity in slice
	h1 := &prettyHandler{
		w:     &buf,
		attrs: make([]slog.Attr, 0, 10), // Start with capacity
	}
	
	// Add first attribute
	h2 := h1.WithAttrs([]slog.Attr{slog.String("key1", "value1")}).(*prettyHandler)
	
	// Verify h2 has one attr
	if len(h2.attrs) != 1 {
		t.Fatalf("h2 should have 1 attr, got %d", len(h2.attrs))
	}
	
	// Add second attribute - this creates h3 from h2
	h3 := h2.WithAttrs([]slog.Attr{slog.String("key2", "value2")}).(*prettyHandler)
	
	// BUG: Because append can reuse the underlying array, h2's attrs might be mutated
	// Check if h2 was mutated
	if len(h2.attrs) != 1 {
		t.Errorf("SLICE ALIASING BUG: h2 should still have 1 attr, but has %d", len(h2.attrs))
		t.Errorf("h2.attrs = %v", h2.attrs)
		t.FailNow()
	}
	
	// Verify h3 has two attrs
	if len(h3.attrs) != 2 {
		t.Errorf("h3 should have 2 attrs, got %d", len(h3.attrs))
	}
	
	// The real test: Log with h2 and verify it only has key1
	buf.Reset()
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	_ = h2.Handle(context.Background(), r)
	
	output := buf.String()
	if bytes.Contains(buf.Bytes(), []byte("key2")) {
		t.Errorf("CRITICAL BUG: h2 output contains key2! Output: %q", output)
		t.Errorf("This proves h2's underlying array was mutated by h3's creation")
	}
}

// TestPrettyHandler_Handle_UnknownLevelProducesConfusingOutput demonstrates that
// unknown log levels produce confusing output with a leading space.
func TestPrettyHandler_Handle_UnknownLevelProducesConfusingOutput(t *testing.T) {
	var buf bytes.Buffer
	h := newPrettyHandler(&buf)
	
	// Use a custom level not in the levelEmoji map
	customLevel := slog.Level(99)
	r := slog.NewRecord(time.Now(), customLevel, "important message", 0)
	
	err := h.Handle(context.Background(), r)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	
	output := buf.String()
	
	// BUG: Output starts with space: " important message\n"
	// This makes it hard to distinguish log level visually
	if output[0] == ' ' {
		t.Errorf("BUG: Unknown level produces leading space in output: %q", output)
		t.Errorf("Expected: A default emoji or error indication")
		t.Errorf("Actual: Empty string from levelEmoji[99] causes formatting issue")
	}
}

// TestPrettyHandler_WithGroup_IgnoresGroupName demonstrates that WithGroup
// silently ignores the group parameter, violating slog.Handler contract.
func TestPrettyHandler_WithGroup_IgnoresGroupName(t *testing.T) {
	var buf bytes.Buffer
	h1 := newPrettyHandler(&buf)
	h2 := h1.WithGroup("database")
	
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "query executed", 0)
	r.AddAttrs(slog.String("table", "users"))
	
	err := h2.Handle(context.Background(), r)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	
	output := buf.String()
	
	// BUG: Group name "database" is completely ignored
	// According to slog.Handler spec, groups should nest attributes
	if !bytes.Contains(buf.Bytes(), []byte("database")) {
		t.Errorf("BUG: WithGroup(\"database\") has no effect on output")
		t.Errorf("Output: %q", output)
		t.Errorf("Expected: Attributes to be prefixed or grouped under 'database'")
		t.Errorf("Actual: Group name silently discarded")
	}
}

// TestNewPrettyHandler_NilWriterPanics demonstrates that nil writer validation is missing.
func TestNewPrettyHandler_NilWriterPanics(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			expected := "writer cannot be nil"
			if r != expected {
				t.Errorf("Expected panic message %q, got %q", expected, r)
			}
		} else {
			t.Errorf("Expected panic with nil writer")
		}
	}()
	
	_ = newPrettyHandler(nil)
	t.Errorf("Should have panicked")
}

// TestPrettyHandler_Handle_StringSliceWithNewlines tests that newlines in slice items
// could break the formatting assumption.
func TestPrettyHandler_Handle_StringSliceWithNewlines(t *testing.T) {
	var buf bytes.Buffer
	h := newPrettyHandler(&buf)
	
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "files", 0)
	r.AddAttrs(slog.Any("items", []string{"line1\nline2", "item2\n\nitem3"}))
	
	err := h.Handle(context.Background(), r)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	
	output := buf.String()
	
	// BUG potential: Format assumes items don't contain newlines
	// Each item is printed as "   • %s\n", so embedded newlines break formatting
	lines := bytes.Split(buf.Bytes(), []byte("\n"))
	t.Logf("Output has %d lines (expected 5: header + 2 items + blank)", len(lines))
	t.Logf("Output: %q", output)
	
	// If items have newlines, we'll get more lines than expected
	if len(lines) > 5 {
		t.Errorf("BUG: String items with embedded newlines break line-based formatting")
		t.Errorf("Got %d lines, expected 5 or fewer", len(lines))
	}
}

// TestLogSuccess_UsesCustomLevel verifies that levelSuccess is actually used
func TestLogSuccess_UsesCustomLevel(t *testing.T) {
	var buf bytes.Buffer
	oldLogger := slog.Default()
	defer slog.SetDefault(oldLogger)
	
	slog.SetDefault(slog.New(newPrettyHandler(&buf)))
	
	logSuccess("operation completed")
	
	output := buf.String()
	
	// Should start with green checkmark emoji
	if !bytes.Contains(buf.Bytes(), []byte("✓")) {
		t.Errorf("Expected green checkmark in output, got: %q", output)
	}
	t.Logf("Success output: %q", output)
}
