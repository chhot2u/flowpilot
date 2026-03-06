package recorder

import (
	"context"
	"fmt"
	"time"

	"web-automation/internal/models"

	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/chromedp"
)

// EventHandler receives recorded steps as they happen.
type EventHandler func(step models.RecordedStep)

// Recorder captures user interactions in a live browser session.
type Recorder struct {
	ctx       context.Context
	handler   EventHandler
	flowID    string
	stepIndex int
}

// New creates a recorder bound to a chromedp context.
func New(ctx context.Context, flowID string, handler EventHandler) *Recorder {
	return &Recorder{ctx: ctx, handler: handler, flowID: flowID, stepIndex: 0}
}

// Start registers listeners and begins capturing interactions.
func (r *Recorder) Start() error {
	chromedp.ListenTarget(r.ctx, func(ev any) {
		switch e := ev.(type) {
		case *dom.EventDocumentUpdated:
			// no-op: reserved for future DOM change tracking
		case *dom.EventAttributeModified:
			// no-op
		case *dom.EventCharacterDataModified:
			// no-op
		default:
			_ = e
		}
	})

	if err := chromedp.Run(r.ctx, dom.Enable()); err != nil {
		return fmt.Errorf("enable dom domain: %w", err)
	}
	return nil
}

// RecordStep emits a recorded step to the handler.
func (r *Recorder) RecordStep(action models.StepAction, selector, value string) {
	if r.handler == nil {
		return
	}
	step := models.RecordedStep{
		Index:     r.stepIndex,
		Action:    action,
		Selector:  selector,
		Value:     value,
		Timestamp: time.Now(),
	}
	r.stepIndex++
	r.handler(step)
}
