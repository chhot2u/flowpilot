package browser

import (
	"context"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type Executor interface {
	Run(ctx context.Context, actions ...chromedp.Action) error
	Targets(ctx context.Context) ([]*target.Info, error)
}

type chromeExecutor struct{}

func (chromeExecutor) Run(ctx context.Context, actions ...chromedp.Action) error {
	return chromedp.Run(ctx, actions...)
}

func (chromeExecutor) Targets(ctx context.Context) ([]*target.Info, error) {
	return chromedp.Targets(ctx)
}
