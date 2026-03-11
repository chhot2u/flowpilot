package browser

import (
	"context"
	"errors"
	"strings"
	"testing"

	"flowpilot/internal/models"
)

func TestExecDoubleClickWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execDoubleClick(context.Background(), models.TaskStep{Selector: "#btn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.callCount() != 1 {
		t.Fatalf("expected 1 call, got %d", mock.callCount())
	}
}

func TestExecDoubleClickError(t *testing.T) {
	mock := &mockExecutor{runErr: errors.New("double click failed")}
	r := newMockRunner(t, mock)

	err := r.execDoubleClick(context.Background(), models.TaskStep{Selector: "#btn"})
	if err == nil || err.Error() != "double click failed" {
		t.Fatalf("expected 'double click failed', got: %v", err)
	}
}

func TestExecFileUploadWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execFileUpload(context.Background(), models.TaskStep{Selector: "#file", Value: "/tmp/test.txt"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.callCount() != 1 {
		t.Fatalf("expected 1 call, got %d", mock.callCount())
	}
}

func TestExecFileUploadError(t *testing.T) {
	mock := &mockExecutor{runErr: errors.New("upload failed")}
	r := newMockRunner(t, mock)

	err := r.execFileUpload(context.Background(), models.TaskStep{Selector: "#file", Value: "/tmp/test.txt"})
	if err == nil || err.Error() != "upload failed" {
		t.Fatalf("expected 'upload failed', got: %v", err)
	}
}

func TestExecNavigateBackWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execNavigateBack(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecNavigateForwardWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execNavigateForward(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecReloadWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execReload(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecScrollIntoViewWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execScrollIntoView(context.Background(), models.TaskStep{Selector: "#elem"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecSubmitFormWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execSubmitForm(context.Background(), models.TaskStep{Selector: "#form"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecWaitNotPresentWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execWaitNotPresent(context.Background(), models.TaskStep{Selector: "#spinner"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecWaitEnabledWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execWaitEnabled(context.Background(), models.TaskStep{Selector: "#btn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecWaitFunctionBlockedByDefault(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execWaitFunction(context.Background(), models.TaskStep{Value: "document.title === 'ready'"})
	if err != ErrEvalNotAllowed {
		t.Fatalf("expected ErrEvalNotAllowed, got: %v", err)
	}
}

func TestExecWaitFunctionValidation(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)
	r.allowEval.Store(true)

	err := r.execWaitFunction(context.Background(), models.TaskStep{Value: "require('child_process')"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "wait_function validation failed") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestExecWaitFunctionWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)
	r.allowEval.Store(true)

	err := r.execWaitFunction(context.Background(), models.TaskStep{Value: "document.title === 'ready'"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecEmulateDeviceValid(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	err := r.execEmulateDevice(context.Background(), models.TaskStep{Value: "375x812"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecEmulateDeviceInvalid(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)

	tests := []struct {
		name  string
		value string
	}{
		{"no separator", "375812"},
		{"invalid width", "abcx812"},
		{"invalid height", "375xabc"},
		{"zero width", "0x812"},
		{"negative height", "375x-1"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := r.execEmulateDevice(context.Background(), models.TaskStep{Value: tc.value})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestExecGetTitleWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)
	result := &models.TaskResult{TaskID: "title-test", ExtractedData: make(map[string]string)}

	err := r.execGetTitle(context.Background(), models.TaskStep{Value: "my_title"}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.ExtractedData["my_title"]; !ok {
		t.Error("expected 'my_title' key in extracted data")
	}
}

func TestExecGetTitleDefaultKey(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)
	result := &models.TaskResult{TaskID: "title-test", ExtractedData: make(map[string]string)}

	err := r.execGetTitle(context.Background(), models.TaskStep{}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := result.ExtractedData["page_title"]; !ok {
		t.Error("expected 'page_title' key in extracted data")
	}
}

func TestExecGetAttributesWithMock(t *testing.T) {
	mock := &mockExecutor{}
	r := newMockRunner(t, mock)
	result := &models.TaskResult{TaskID: "attrs-test", ExtractedData: make(map[string]string)}

	err := r.execGetAttributes(context.Background(), models.TaskStep{Selector: "#elem", Value: "el"}, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseViewportSize(t *testing.T) {
	tests := []struct {
		input string
		w, h  int
		err   bool
	}{
		{"1920x1080", 1920, 1080, false},
		{"375x812", 375, 812, false},
		{"invalid", 0, 0, true},
		{"0x0", 0, 0, true},
		{"-1x100", 0, 0, true},
		{"100x-1", 0, 0, true},
		{"abcx100", 0, 0, true},
		{"100xabc", 0, 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			w, h, err := parseViewportSize(tc.input)
			if tc.err {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if w != tc.w || h != tc.h {
				t.Errorf("got %dx%d, want %dx%d", w, h, tc.w, tc.h)
			}
		})
	}
}

func TestExecuteStepNewActionsDispatch(t *testing.T) {
	actions := []struct {
		action   models.StepAction
		selector string
		value    string
	}{
		{models.ActionDoubleClick, "#btn", ""},
		{models.ActionFileUpload, "#file", "/tmp/test.txt"},
		{models.ActionNavigateBack, "", ""},
		{models.ActionNavigateForward, "", ""},
		{models.ActionReload, "", ""},
		{models.ActionScrollIntoView, "#elem", ""},
		{models.ActionSubmitForm, "#form", ""},
		{models.ActionWaitNotPresent, "#spinner", ""},
		{models.ActionWaitEnabled, "#btn", ""},
		{models.ActionEmulateDevice, "", "1920x1080"},
		{models.ActionGetTitle, "", "title_key"},
	}

	for _, tc := range actions {
		t.Run(string(tc.action), func(t *testing.T) {
			mock := &mockExecutor{}
			r := newMockRunner(t, mock)
			result := &models.TaskResult{TaskID: "dispatch", ExtractedData: make(map[string]string)}

			step := models.TaskStep{Action: tc.action, Selector: tc.selector, Value: tc.value}
			err := r.executeStep(context.Background(), step, result)
			if err != nil && err.Error() == "unknown action: "+string(tc.action) {
				t.Fatalf("action %s was not dispatched correctly", tc.action)
			}
		})
	}
}
