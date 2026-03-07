package logs

import (
	"testing"
	"time"

	"github.com/chromedp/cdproto/network"
)

func TestNewNetworkLogger(t *testing.T) {
	nl := NewNetworkLogger("task-net")
	if nl.taskID != "task-net" {
		t.Errorf("taskID: got %q, want %q", nl.taskID, "task-net")
	}
	if len(nl.Logs()) != 0 {
		t.Errorf("initial Logs: got %d, want 0", len(nl.Logs()))
	}
}

func TestSetStepIndex(t *testing.T) {
	nl := NewNetworkLogger("task-step")
	nl.SetStepIndex(3)
	if nl.stepIndex != 3 {
		t.Errorf("stepIndex: got %d, want 3", nl.stepIndex)
	}
}

func TestHandleFullRequestResponseCycle(t *testing.T) {
	nl := NewNetworkLogger("task-cycle")
	nl.SetStepIndex(1)
	reqID := network.RequestID("req-1")

	// Simulate request sent
	nl.HandleRequestWillBeSent(&network.EventRequestWillBeSent{
		RequestID: reqID,
		Request:   &network.Request{Method: "GET", Headers: network.Headers{"Accept": "text/html"}},
	})

	// Simulate response received
	nl.HandleResponseReceived(&network.EventResponseReceived{
		RequestID: reqID,
		Response: &network.Response{
			URL:      "https://example.com/page",
			Status:   200,
			MimeType: "text/html",
			Headers:  network.Headers{"Content-Type": "text/html"},
		},
	})

	// Simulate loading finished
	nl.HandleLoadingFinished(&network.EventLoadingFinished{
		RequestID:         reqID,
		EncodedDataLength: 1234,
	}, nil)

	logs := nl.Logs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	log := logs[0]
	if log.TaskID != "task-cycle" {
		t.Errorf("TaskID: got %q, want %q", log.TaskID, "task-cycle")
	}
	if log.StepIndex != 1 {
		t.Errorf("StepIndex: got %d, want 1", log.StepIndex)
	}
	if log.RequestURL != "https://example.com/page" {
		t.Errorf("RequestURL: got %q", log.RequestURL)
	}
	if log.Method != "GET" {
		t.Errorf("Method: got %q, want GET", log.Method)
	}
	if log.StatusCode != 200 {
		t.Errorf("StatusCode: got %d, want 200", log.StatusCode)
	}
	if log.MimeType != "text/html" {
		t.Errorf("MimeType: got %q", log.MimeType)
	}
	if log.ResponseSize != 1234 {
		t.Errorf("ResponseSize: got %d, want 1234", log.ResponseSize)
	}
	if log.DurationMs < 0 {
		t.Errorf("DurationMs: got %d, should be >= 0", log.DurationMs)
	}
	if log.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestHandleLoadingFinishedWithResponseOverride(t *testing.T) {
	nl := NewNetworkLogger("task-override")
	reqID := network.RequestID("req-2")

	nl.HandleRequestWillBeSent(&network.EventRequestWillBeSent{
		RequestID: reqID,
		Request:   &network.Request{Method: "POST", Headers: network.Headers{}},
	})

	// Response passed directly in HandleLoadingFinished (override)
	resp := &network.Response{
		URL:      "https://api.example.com",
		Status:   201,
		MimeType: "application/json",
		Headers:  network.Headers{},
	}
	nl.HandleLoadingFinished(&network.EventLoadingFinished{
		RequestID:         reqID,
		EncodedDataLength: 500,
	}, resp)

	logs := nl.Logs()
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}
	if logs[0].StatusCode != 201 {
		t.Errorf("StatusCode: got %d, want 201", logs[0].StatusCode)
	}
}

func TestHandleLoadingFinishedWithoutRequest(t *testing.T) {
	nl := NewNetworkLogger("task-nodata")
	reqID := network.RequestID("req-orphan")

	// Loading finished without prior request/response — should be skipped
	nl.HandleLoadingFinished(&network.EventLoadingFinished{
		RequestID:         reqID,
		EncodedDataLength: 100,
	}, nil)

	if len(nl.Logs()) != 0 {
		t.Errorf("expected 0 logs for orphan request, got %d", len(nl.Logs()))
	}
}

func TestHandleLoadingFinishedWithoutResponse(t *testing.T) {
	nl := NewNetworkLogger("task-noresp")
	reqID := network.RequestID("req-noresp")

	// Only request, no response
	nl.HandleRequestWillBeSent(&network.EventRequestWillBeSent{
		RequestID: reqID,
		Request:   &network.Request{Method: "GET", Headers: network.Headers{}},
	})

	// Loading finished without response — should be skipped (resp == nil check)
	nl.HandleLoadingFinished(&network.EventLoadingFinished{
		RequestID:         reqID,
		EncodedDataLength: 100,
	}, nil)

	if len(nl.Logs()) != 0 {
		t.Errorf("expected 0 logs when response is missing, got %d", len(nl.Logs()))
	}
}

func TestMultipleRequestsCaptured(t *testing.T) {
	nl := NewNetworkLogger("task-multi")

	for i := 0; i < 3; i++ {
		reqID := network.RequestID(time.Now().String() + string(rune(i)))
		nl.HandleRequestWillBeSent(&network.EventRequestWillBeSent{
			RequestID: reqID,
			Request:   &network.Request{Method: "GET", Headers: network.Headers{}},
		})
		nl.HandleResponseReceived(&network.EventResponseReceived{
			RequestID: reqID,
			Response: &network.Response{
				URL:      "https://example.com",
				Status:   200,
				MimeType: "text/html",
				Headers:  network.Headers{},
			},
		})
		nl.HandleLoadingFinished(&network.EventLoadingFinished{
			RequestID:         reqID,
			EncodedDataLength: 100,
		}, nil)
	}

	if len(nl.Logs()) != 3 {
		t.Errorf("expected 3 logs, got %d", len(nl.Logs()))
	}
}

func TestStartTimeClearedAfterFinish(t *testing.T) {
	nl := NewNetworkLogger("task-cleanup")
	reqID := network.RequestID("req-clean")

	nl.HandleRequestWillBeSent(&network.EventRequestWillBeSent{
		RequestID: reqID,
		Request:   &network.Request{Method: "GET", Headers: network.Headers{}},
	})
	nl.HandleResponseReceived(&network.EventResponseReceived{
		RequestID: reqID,
		Response:  &network.Response{URL: "https://example.com", Status: 200, Headers: network.Headers{}},
	})
	nl.HandleLoadingFinished(&network.EventLoadingFinished{
		RequestID: reqID,
	}, nil)

	// Internal maps should be cleaned up
	if _, ok := nl.startTimes[reqID]; ok {
		t.Error("startTimes should be cleaned up after finish")
	}
	if _, ok := nl.requests[reqID]; ok {
		t.Error("requests should be cleaned up after finish")
	}
	if _, ok := nl.responses[reqID]; ok {
		t.Error("responses should be cleaned up after finish")
	}
}

func TestHandleLoadingFailedCleansUpMaps(t *testing.T) {
	logger := NewNetworkLogger("task-fail-1")

	// Simulate a request that starts but never finishes
	logger.HandleRequestWillBeSent(&network.EventRequestWillBeSent{
		RequestID: "req-failed-1",
		Request:   &network.Request{URL: "https://example.com/timeout", Method: "GET"},
	})
	logger.HandleResponseReceived(&network.EventResponseReceived{
		RequestID: "req-failed-1",
		Response:  &network.Response{URL: "https://example.com/timeout", Status: 0},
	})

	// Request fails — this should clean up
	logger.HandleLoadingFailed("req-failed-1")

	// Verify maps are empty (no leak)
	logger.mu.Lock()
	startLen := len(logger.startTimes)
	reqLen := len(logger.requests)
	respLen := len(logger.responses)
	logger.mu.Unlock()

	if startLen != 0 {
		t.Errorf("startTimes should be empty, got %d entries", startLen)
	}
	if reqLen != 0 {
		t.Errorf("requests should be empty, got %d entries", reqLen)
	}
	if respLen != 0 {
		t.Errorf("responses should be empty, got %d entries", respLen)
	}

	// Logs should be empty (failed request doesn't produce a log)
	logs := logger.Logs()
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

func TestHandleLoadingFailedIdempotent(t *testing.T) {
	logger := NewNetworkLogger("task-fail-2")

	// Call HandleLoadingFailed for a request that was never started
	logger.HandleLoadingFailed("nonexistent-req")

	// Should not panic or produce logs
	logs := logger.Logs()
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}
