package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"flowpilot/internal/models"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

// executeStep dispatches to the appropriate action handler.
func (r *Runner) executeStep(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	switch step.Action {
	case models.ActionNavigate:
		return r.execNavigate(ctx, step)
	case models.ActionClick:
		return r.execClick(ctx, step)
	case models.ActionType:
		return r.execType(ctx, step)
	case models.ActionWait:
		return r.execWait(ctx, step)
	case models.ActionScreenshot:
		return r.execScreenshot(ctx, result)
	case models.ActionExtract:
		return r.execExtract(ctx, step, result)
	case models.ActionScroll:
		return r.execScroll(ctx, step)
	case models.ActionSelect:
		return r.execSelect(ctx, step)
	case models.ActionEval:
		return r.execEval(ctx, step)
	case models.ActionTabSwitch:
		return r.execTabSwitch(ctx, step)
	case models.ActionSolveCaptcha:
		return r.execSolveCaptcha(ctx, step, result)
	case models.ActionDoubleClick:
		return r.execDoubleClick(ctx, step)
	case models.ActionFileUpload:
		return r.execFileUpload(ctx, step)
	case models.ActionNavigateBack:
		return r.execNavigateBack(ctx)
	case models.ActionNavigateForward:
		return r.execNavigateForward(ctx)
	case models.ActionReload:
		return r.execReload(ctx)
	case models.ActionScrollIntoView:
		return r.execScrollIntoView(ctx, step)
	case models.ActionSubmitForm:
		return r.execSubmitForm(ctx, step)
	case models.ActionWaitNotPresent:
		return r.execWaitNotPresent(ctx, step)
	case models.ActionWaitEnabled:
		return r.execWaitEnabled(ctx, step)
	case models.ActionWaitFunction:
		return r.execWaitFunction(ctx, step)
	case models.ActionEmulateDevice:
		return r.execEmulateDevice(ctx, step)
	case models.ActionGetTitle:
		return r.execGetTitle(ctx, step, result)
	case models.ActionGetAttributes:
		return r.execGetAttributes(ctx, step, result)
	default:
		return fmt.Errorf("unknown action: %s", step.Action)
	}
}

func (r *Runner) execNavigate(ctx context.Context, step models.TaskStep) error {
	resp, err := r.exec.RunResponse(ctx, chromedp.Navigate(step.Value))
	if err != nil {
		return err
	}
	if resp != nil && resp.Status >= 400 {
		return fmt.Errorf("navigation to %s returned HTTP %d", step.Value, resp.Status)
	}
	return nil
}

func requireSelector(action, selector string) error {
	if strings.TrimSpace(selector) == "" {
		return fmt.Errorf("%s: selector is required", action)
	}
	return nil
}

func requireValue(action, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s: value is required", action)
	}
	return nil
}

func (r *Runner) execClick(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("click", step.Selector); err != nil {
		return err
	}
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.Click(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execType(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("type", step.Selector); err != nil {
		return err
	}
	if err := requireValue("type", step.Value); err != nil {
		return err
	}
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.Clear(step.Selector, chromedp.ByQuery),
		chromedp.SendKeys(step.Selector, step.Value, chromedp.ByQuery),
	)
}

func (r *Runner) execWait(ctx context.Context, step models.TaskStep) error {
	if step.Selector != "" {
		return r.exec.Run(ctx,
			chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		)
	}
	dur, err := time.ParseDuration(step.Value + "ms")
	if err != nil {
		dur = 1 * time.Second
	}
	timer := time.NewTimer(dur)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runner) execScreenshot(ctx context.Context, result *models.TaskResult) error {
	var buf []byte
	if err := r.exec.Run(ctx, chromedp.FullScreenshot(&buf, 100)); err != nil {
		return fmt.Errorf("capture screenshot: %w", err)
	}
	sanitizedID := sanitizeFilename(result.TaskID)
	filename := fmt.Sprintf("%s_%d.png", sanitizedID, time.Now().UnixMilli())
	path := filepath.Join(r.screenshotDir, filename)
	if !strings.HasPrefix(path, filepath.Clean(r.screenshotDir)+string(os.PathSeparator)) {
		return fmt.Errorf("screenshot path escapes screenshot directory")
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return fmt.Errorf("save screenshot: %w", err)
	}
	result.Screenshots = append(result.Screenshots, path)
	return nil
}

func sanitizeFilename(name string) string {
	safe := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == '.' || r == '\x00' {
			return '_'
		}
		return r
	}, name)
	return filepath.Base(safe)
}

func (r *Runner) execExtract(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if err := requireSelector("extract", step.Selector); err != nil {
		return err
	}
	var text string
	if err := r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.Text(step.Selector, &text, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("extract text: %w", err)
	}
	key := step.Value
	if key == "" {
		key = step.Selector
	}
	result.ExtractedData[key] = text
	return nil
}

func (r *Runner) execScroll(ctx context.Context, step models.TaskStep) error {
	if _, err := strconv.Atoi(step.Value); err != nil {
		return fmt.Errorf("invalid scroll value %q: must be an integer", step.Value)
	}
	return r.exec.Run(ctx,
		chromedp.Evaluate(`window.scrollBy(0, `+step.Value+`)`, nil),
	)
}

func (r *Runner) execSelect(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("select", step.Selector); err != nil {
		return err
	}
	if err := requireValue("select", step.Value); err != nil {
		return err
	}
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.SetValue(step.Selector, step.Value, chromedp.ByQuery),
	)
}

func (r *Runner) execEval(ctx context.Context, step models.TaskStep) error {
	if !r.allowEval.Load() {
		return ErrEvalNotAllowed
	}
	if err := validateEvalScript(step.Value); err != nil {
		return fmt.Errorf("eval validation failed: %w", err)
	}
	var res any
	return r.exec.Run(ctx,
		chromedp.Evaluate(step.Value, &res),
	)
}

func (r *Runner) execTabSwitch(ctx context.Context, step models.TaskStep) error {
	if err := requireValue("tab_switch", step.Value); err != nil {
		return err
	}
	targets, err := r.exec.Targets(ctx)
	if err != nil {
		return fmt.Errorf("list targets: %w", err)
	}
	for _, t := range targets {
		if t.Type == "page" && t.URL == step.Value {
			return r.exec.Run(ctx, chromedp.ActionFunc(func(c context.Context) error {
				return target.ActivateTarget(t.TargetID).Do(c)
			}))
		}
	}
	return fmt.Errorf("tab with URL %q not found", step.Value)
}

func (r *Runner) execSolveCaptcha(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if r.captchaSolver == nil {
		return fmt.Errorf("captcha solver not configured")
	}

	var pageURL string
	if err := r.exec.Run(ctx, chromedp.Location(&pageURL)); err != nil {
		return fmt.Errorf("get page url for captcha: %w", err)
	}

	req := models.CaptchaSolveRequest{
		Type:    models.CaptchaType(step.Value),
		SiteKey: step.Selector,
		PageURL: pageURL,
	}

	solveResult, err := r.captchaSolver.Solve(ctx, req)
	if err != nil {
		return fmt.Errorf("solve captcha: %w", err)
	}

	key := "captcha_token"
	if step.VarName != "" {
		key = step.VarName
	}
	result.ExtractedData[key] = solveResult.Token

	if step.Value == string(models.CaptchaTypeRecaptchaV2) || step.Value == string(models.CaptchaTypeRecaptchaV3) {
		js := fmt.Sprintf(`document.getElementById("g-recaptcha-response").innerHTML = %q;`, solveResult.Token)
		var res interface{}
		if err := r.exec.Run(ctx, chromedp.Evaluate(js, &res)); err != nil {
			r.addLog(result, "warn", fmt.Sprintf("inject captcha token: %v", err))
		}
	}

	return nil
}

func (r *Runner) execDoubleClick(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("double_click", step.Selector); err != nil {
		return err
	}
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.DoubleClick(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execFileUpload(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("file_upload", step.Selector); err != nil {
		return err
	}
	if err := requireValue("file_upload", step.Value); err != nil {
		return err
	}
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.SetUploadFiles(step.Selector, []string{step.Value}, chromedp.ByQuery),
	)
}

func (r *Runner) execNavigateBack(ctx context.Context) error {
	return r.exec.Run(ctx, chromedp.NavigateBack())
}

func (r *Runner) execNavigateForward(ctx context.Context) error {
	return r.exec.Run(ctx, chromedp.NavigateForward())
}

func (r *Runner) execReload(ctx context.Context) error {
	return r.exec.Run(ctx, chromedp.Reload())
}

func (r *Runner) execScrollIntoView(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("scroll_into_view", step.Selector); err != nil {
		return err
	}
	return r.exec.Run(ctx,
		chromedp.ScrollIntoView(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execSubmitForm(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("submit_form", step.Selector); err != nil {
		return err
	}
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.Submit(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execWaitNotPresent(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("wait_not_present", step.Selector); err != nil {
		return err
	}
	return r.exec.Run(ctx,
		chromedp.WaitNotPresent(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execWaitEnabled(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("wait_enabled", step.Selector); err != nil {
		return err
	}
	return r.exec.Run(ctx,
		chromedp.WaitEnabled(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execWaitFunction(ctx context.Context, step models.TaskStep) error {
	if !r.allowEval.Load() {
		return ErrEvalNotAllowed
	}
	if err := validateEvalScript(step.Value); err != nil {
		return fmt.Errorf("wait_function validation failed: %w", err)
	}
	return r.exec.Run(ctx,
		chromedp.Poll(step.Value, nil),
	)
}

func (r *Runner) execEmulateDevice(ctx context.Context, step models.TaskStep) error {
	width, height, err := parseViewportSize(step.Value)
	if err != nil {
		return fmt.Errorf("invalid viewport size %q: %w", step.Value, err)
	}
	return r.exec.Run(ctx,
		chromedp.EmulateViewport(int64(width), int64(height)),
	)
}

func parseViewportSize(val string) (int, int, error) {
	parts := strings.SplitN(val, "x", 2)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected WIDTHxHEIGHT format, got %q", val)
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid width: %w", err)
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid height: %w", err)
	}
	if w <= 0 || h <= 0 {
		return 0, 0, fmt.Errorf("width and height must be positive")
	}
	return w, h, nil
}

func (r *Runner) execGetTitle(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	var title string
	if err := r.exec.Run(ctx, chromedp.Title(&title)); err != nil {
		return fmt.Errorf("get title: %w", err)
	}
	key := step.Value
	if key == "" {
		key = "page_title"
	}
	result.ExtractedData[key] = title
	return nil
}

func (r *Runner) execGetAttributes(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if err := requireSelector("get_attributes", step.Selector); err != nil {
		return err
	}
	var attrs map[string]string
	if err := r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.Attributes(step.Selector, &attrs, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("get attributes: %w", err)
	}
	key := step.Value
	if key == "" {
		key = step.Selector
	}
	for k, v := range attrs {
		result.ExtractedData[key+"_"+k] = v
	}
	return nil
}
