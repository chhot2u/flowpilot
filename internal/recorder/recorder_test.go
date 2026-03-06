package recorder

import (
	"context"
	"testing"

	"web-automation/internal/models"
)

func TestRecordStep(t *testing.T) {
	var captured []models.RecordedStep
	handler := func(step models.RecordedStep) {
		captured = append(captured, step)
	}

	// Use a nil context since we won't actually run chromedp
	r := &Recorder{
		handler: handler,
		flowID:  "flow-1",
	}

	r.RecordStep(models.ActionClick, "#btn", "")
	r.RecordStep(models.ActionType, "#input", "hello")
	r.RecordStep(models.ActionNavigate, "", "https://example.com")

	if len(captured) != 3 {
		t.Fatalf("expected 3 captured steps, got %d", len(captured))
	}

	if captured[0].Index != 0 {
		t.Errorf("step[0].Index: got %d, want 0", captured[0].Index)
	}
	if captured[0].Action != models.ActionClick {
		t.Errorf("step[0].Action: got %q, want %q", captured[0].Action, models.ActionClick)
	}
	if captured[0].Selector != "#btn" {
		t.Errorf("step[0].Selector: got %q, want %q", captured[0].Selector, "#btn")
	}

	if captured[1].Index != 1 {
		t.Errorf("step[1].Index: got %d, want 1", captured[1].Index)
	}
	if captured[1].Value != "hello" {
		t.Errorf("step[1].Value: got %q, want %q", captured[1].Value, "hello")
	}

	if captured[2].Index != 2 {
		t.Errorf("step[2].Index: got %d, want 2", captured[2].Index)
	}
	if captured[2].Action != models.ActionNavigate {
		t.Errorf("step[2].Action: got %q, want %q", captured[2].Action, models.ActionNavigate)
	}
	if captured[2].Value != "https://example.com" {
		t.Errorf("step[2].Value: got %q", captured[2].Value)
	}

	if captured[0].Timestamp.IsZero() {
		t.Error("step timestamps should not be zero")
	}
}

func TestRecordStepNilHandler(t *testing.T) {
	r := &Recorder{handler: nil, flowID: "flow-nil"}
	// Should not panic
	r.RecordStep(models.ActionClick, "#btn", "")
}

func TestRecordStepIncrementsIndex(t *testing.T) {
	var captured []models.RecordedStep
	r := &Recorder{
		handler: func(step models.RecordedStep) {
			captured = append(captured, step)
		},
		flowID: "flow-idx",
	}

	for i := 0; i < 10; i++ {
		r.RecordStep(models.ActionClick, "#btn", "")
	}

	for i, step := range captured {
		if step.Index != i {
			t.Errorf("step[%d].Index: got %d, want %d", i, step.Index, i)
		}
	}
}

func TestRankSelectors(t *testing.T) {
	candidates := []models.SelectorCandidate{
		{Selector: ".class", Strategy: models.SelectorCSS, Score: 30},
		{Selector: "#id", Strategy: models.SelectorID, Score: 90},
		{Selector: "[data-testid]", Strategy: models.SelectorDataTestID, Score: 95},
		{Selector: "//div", Strategy: models.SelectorXPath, Score: 10},
	}

	ranked := RankSelectors(candidates)
	if ranked[0].Score != 95 {
		t.Errorf("top rank score: got %d, want 95", ranked[0].Score)
	}
	if ranked[0].Selector != "[data-testid]" {
		t.Errorf("top rank selector: got %q", ranked[0].Selector)
	}
	if ranked[len(ranked)-1].Score != 10 {
		t.Errorf("lowest rank score: got %d, want 10", ranked[len(ranked)-1].Score)
	}
}

func TestRankSelectorsEmpty(t *testing.T) {
	ranked := RankSelectors(nil)
	if len(ranked) != 0 {
		t.Errorf("expected empty result, got %d", len(ranked))
	}
}

func TestRankSelectorsDoesNotMutateOriginal(t *testing.T) {
	candidates := []models.SelectorCandidate{
		{Selector: "c", Score: 30},
		{Selector: "a", Score: 90},
		{Selector: "b", Score: 50},
	}
	origFirst := candidates[0]

	_ = RankSelectors(candidates)

	if candidates[0] != origFirst {
		t.Error("RankSelectors should not mutate the original slice")
	}
}

func TestBestSelector(t *testing.T) {
	candidates := []models.SelectorCandidate{
		{Selector: ".low", Score: 10},
		{Selector: "#high", Score: 99},
		{Selector: ".mid", Score: 50},
	}

	best := BestSelector(candidates)
	if best != "#high" {
		t.Errorf("BestSelector: got %q, want %q", best, "#high")
	}
}

func TestBestSelectorEmpty(t *testing.T) {
	best := BestSelector(nil)
	if best != "" {
		t.Errorf("BestSelector(nil): got %q, want empty", best)
	}
}

func TestSanitize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"normal-id", "normal-id"},
		{"path/traversal", "path_traversal"},
		{"back\\slash", "back_slash"},
		{"dot.dot", "dot_dot"},
		{"null\x00byte", "null_byte"},
	}

	for _, tc := range tests {
		got := sanitize(tc.input)
		if got != tc.want {
			t.Errorf("sanitize(%q): got %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNewRecorder(t *testing.T) {
	ctx := context.Background()
	var called bool
	handler := func(step models.RecordedStep) {
		called = true
	}

	r := New(ctx, "flow-new", handler)
	if r.flowID != "flow-new" {
		t.Errorf("flowID: got %q, want %q", r.flowID, "flow-new")
	}
	if r.parentCtx != ctx {
		t.Error("parentCtx not set correctly")
	}
	if r.BrowserCtx() != nil {
		t.Error("BrowserCtx should be nil before Start")
	}
	if r.FlowID() != "flow-new" {
		t.Errorf("FlowID(): got %q, want %q", r.FlowID(), "flow-new")
	}

	r.RecordStep(models.ActionClick, "#btn", "")
	if !called {
		t.Error("handler was not called")
	}
}

func TestStopIdempotent(t *testing.T) {
	r := &Recorder{flowID: "flow-stop"}

	r.Stop()
	r.Stop()
}

func TestStopCleansUpContexts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := &Recorder{
		flowID:        "flow-cleanup",
		allocCtx:      ctx,
		allocCancel:   cancel,
		browserCtx:    ctx,
		browserCancel: cancel,
	}

	r.Stop()
	if r.browserCtx != nil {
		t.Error("browserCtx should be nil after Stop")
	}
	if r.allocCtx != nil {
		t.Error("allocCtx should be nil after Stop")
	}
	if r.browserCancel != nil {
		t.Error("browserCancel should be nil after Stop")
	}
	if r.allocCancel != nil {
		t.Error("allocCancel should be nil after Stop")
	}
}

func TestBrowserCtxNilBeforeStart(t *testing.T) {
	r := New(context.Background(), "flow-ctx", nil)
	if r.BrowserCtx() != nil {
		t.Error("BrowserCtx() should return nil before Start")
	}
}
