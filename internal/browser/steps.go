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
	case models.ActionClickAd:
		return r.execClickAd(ctx, step, result)
	case models.ActionWhile:
		return r.execWhile(ctx, step, result)
	case models.ActionEndWhile:
		return r.execEndWhile(ctx, step, result)
	case models.ActionIfExists:
		return r.execIfExists(ctx, step, result)
	case models.ActionIfNotExists:
		return r.execIfNotExists(ctx, step, result)
	case models.ActionIfVisible:
		return r.execIfVisible(ctx, step, result)
	case models.ActionIfEnabled:
		return r.execIfEnabled(ctx, step, result)
	case models.ActionTryBlock:
		return r.execTryBlock(ctx, step, result)
	case models.ActionVariableSet:
		return r.execVariableSet(ctx, step, result)
	case models.ActionVariableMath:
		return r.execVariableMath(ctx, step, result)
	case models.ActionVariableString:
		return r.execVariableString(ctx, step, result)
	case models.ActionHover:
		return r.execHover(ctx, step)
	case models.ActionDragDrop:
		return r.execDragDrop(ctx, step)
	case models.ActionContextClick:
		return r.execContextClick(ctx, step)
	case models.ActionHighlight:
		return r.execHighlight(ctx, step, result)
	case models.ActionGetCookies:
		return r.execGetCookies(ctx, step, result)
	case models.ActionSetCookie:
		return r.execSetCookie(ctx, step, result)
	case models.ActionDeleteCookies:
		return r.execDeleteCookies(ctx, step, result)
	case models.ActionGetStorage:
		return r.execGetStorage(ctx, step, result)
	case models.ActionSetStorage:
		return r.execSetStorage(ctx, step, result)
	case models.ActionDeleteStorage:
		return r.execDeleteStorage(ctx, step, result)
	case models.ActionDownload:
		return r.execDownload(ctx, step, result)
	case models.ActionSelectRandom:
		return r.execSelectRandom(ctx, step, result)
	case models.ActionDebugPause:
		return r.execDebugPause(ctx, step, result)
	case models.ActionDebugResume:
		return r.execDebugResume(ctx, step, result)
	case models.ActionDebugStep:
		return r.execDebugStep(ctx, step, result)
	case models.ActionHeadless:
		return r.execHeadless(ctx, step)
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

// adDiscoveryScript is injected into the page to find a visible ad element.
// It returns a JSON-serialisable object with the ad's bounding rect and metadata.
const adDiscoveryScript = `(function() {
  var selectors = [
    'ins.adsbygoogle',
    'iframe[id*="google_ads"]',
    'iframe[id*="aswift"]',
    'div[id*="google_ads"]',
    'div[id*="ad-container"]',
    'div[class*="ad-slot"]',
    'div[class*="ad-wrapper"]',
    'div[data-ad]',
    'iframe[data-google-container-id]',
    'a[href*="googleads"]',
    'a[href*="doubleclick"]'
  ];
  for (var i = 0; i < selectors.length; i++) {
    var el = document.querySelector(selectors[i]);
    if (el) {
      var rect = el.getBoundingClientRect();
      if (rect.width > 0 && rect.height > 0) {
        return {
          found: true,
          selector: selectors[i],
          tag: el.tagName.toLowerCase(),
          href: el.href || el.src || '',
          x: Math.round(rect.x + rect.width / 2),
          y: Math.round(rect.y + rect.height / 2)
        };
      }
    }
  }
  return { found: false };
})()`

// adClickAtScript dispatches a mouse click at the given page coordinates.
const adClickAtScript = `(function(x, y) {
  var el = document.elementFromPoint(x, y);
  if (el) {
    el.dispatchEvent(new MouseEvent('click', {
      bubbles: true, cancelable: true, view: window,
      clientX: x, clientY: y
    }));
    return true;
  }
  return false;
})`

type adDiscoveryResult struct {
	Found    bool    `json:"found"`
	Selector string  `json:"selector"`
	Tag      string  `json:"tag"`
	Href     string  `json:"href"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
}

// captureAdScreenshot takes a full-page screenshot labelled with the given tag
// (e.g. "before", "after") and stores the path in both result.Screenshots and
// result.ExtractedData. Errors are non-fatal and returned for the caller to log.
func (r *Runner) captureAdScreenshot(ctx context.Context, result *models.TaskResult, keyPrefix, label string) (string, error) {
	var buf []byte
	if err := r.exec.Run(ctx, chromedp.FullScreenshot(&buf, 100)); err != nil {
		return "", fmt.Errorf("capture ad screenshot (%s): %w", label, err)
	}
	sanitizedID := sanitizeFilename(result.TaskID)
	filename := fmt.Sprintf("%s_ad_%s_%d.png", sanitizedID, label, time.Now().UnixMilli())
	path := filepath.Join(r.screenshotDir, filename)
	if !strings.HasPrefix(path, filepath.Clean(r.screenshotDir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("ad screenshot path escapes screenshot directory")
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return "", fmt.Errorf("save ad screenshot (%s): %w", label, err)
	}
	result.Screenshots = append(result.Screenshots, path)
	result.ExtractedData[keyPrefix+"_screenshot_"+label] = path
	return path, nil
}

func (r *Runner) execClickAd(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	keyPrefix := "ad"
	if step.VarName != "" {
		keyPrefix = step.VarName
	}

	// If a selector is explicitly provided, use it directly.
	if strings.TrimSpace(step.Selector) != "" {
		// Try to extract metadata before clicking.
		metaJS := fmt.Sprintf(`(function() {
  var el = document.querySelector(%q);
  if (!el) return { found: false };
  var rect = el.getBoundingClientRect();
  return {
    found: true,
    selector: %q,
    tag: el.tagName.toLowerCase(),
    href: el.href || el.src || '',
    x: Math.round(rect.x + rect.width / 2),
    y: Math.round(rect.y + rect.height / 2)
  };
})()`, step.Selector, step.Selector)

		var info adDiscoveryResult
		if err := r.exec.Run(ctx, chromedp.Evaluate(metaJS, &info)); err != nil {
			return fmt.Errorf("click_ad: evaluate selector metadata: %w", err)
		}
		if !info.Found {
			return fmt.Errorf("click_ad: element not found for selector %q", step.Selector)
		}

		result.ExtractedData[keyPrefix+"_selector"] = info.Selector
		result.ExtractedData[keyPrefix+"_tag"] = info.Tag
		result.ExtractedData[keyPrefix+"_href"] = info.Href

		// Capture before-click screenshot.
		if _, err := r.captureAdScreenshot(ctx, result, keyPrefix, "before"); err != nil {
			r.addLog(result, "warn", fmt.Sprintf("click_ad: %v", err))
		}

		// For iframes, dispatch a coordinate-based click since we can't enter cross-origin frames.
		if info.Tag == "iframe" {
			clickJS := fmt.Sprintf(`(%s)(%v, %v)`, adClickAtScript, info.X, info.Y)
			var clicked bool
			if err := r.exec.Run(ctx, chromedp.Evaluate(clickJS, &clicked)); err != nil {
				return fmt.Errorf("click_ad: dispatch click on iframe: %w", err)
			}
			if !clicked {
				return fmt.Errorf("click_ad: no element at iframe center (%v, %v)", info.X, info.Y)
			}
		} else {
			// Regular element — use standard chromedp click.
			if err := r.exec.Run(ctx,
				chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
				chromedp.Click(step.Selector, chromedp.ByQuery),
			); err != nil {
				return err
			}
		}

		// Capture after-click screenshot.
		if _, err := r.captureAdScreenshot(ctx, result, keyPrefix, "after"); err != nil {
			r.addLog(result, "warn", fmt.Sprintf("click_ad: %v", err))
		}
		return nil
	}

	// No selector provided — auto-discover an ad element.
	var info adDiscoveryResult
	if err := r.exec.Run(ctx, chromedp.Evaluate(adDiscoveryScript, &info)); err != nil {
		return fmt.Errorf("click_ad: ad discovery failed: %w", err)
	}
	if !info.Found {
		return fmt.Errorf("click_ad: no ad element found on page")
	}

	result.ExtractedData[keyPrefix+"_selector"] = info.Selector
	result.ExtractedData[keyPrefix+"_tag"] = info.Tag
	result.ExtractedData[keyPrefix+"_href"] = info.Href

	// Capture before-click screenshot.
	if _, err := r.captureAdScreenshot(ctx, result, keyPrefix, "before"); err != nil {
		r.addLog(result, "warn", fmt.Sprintf("click_ad: %v", err))
	}

	// Dispatch a coordinate-based click (works for both iframes and regular elements).
	clickJS := fmt.Sprintf(`(%s)(%v, %v)`, adClickAtScript, info.X, info.Y)
	var clicked bool
	if err := r.exec.Run(ctx, chromedp.Evaluate(clickJS, &clicked)); err != nil {
		return fmt.Errorf("click_ad: dispatch click: %w", err)
	}
	if !clicked {
		return fmt.Errorf("click_ad: no element at coordinates (%v, %v)", info.X, info.Y)
	}

	// Capture after-click screenshot.
	if _, err := r.captureAdScreenshot(ctx, result, keyPrefix, "after"); err != nil {
		r.addLog(result, "warn", fmt.Sprintf("click_ad: %v", err))
	}
	return nil
}

func (r *Runner) execWhile(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if step.Condition == "" {
		return fmt.Errorf("while_condition: condition is required")
	}
	maxLoops := step.MaxLoops
	if maxLoops <= 0 {
		maxLoops = 1000
	}
	result.ExtractedData["_while_max_loops"] = strconv.Itoa(maxLoops)
	result.ExtractedData["_while_condition"] = step.Condition
	return nil
}

func (r *Runner) execEndWhile(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	return nil
}

func (r *Runner) execIfExists(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if err := requireSelector("if_exists", step.Selector); err != nil {
		return err
	}
	var exists bool
	checkJS := fmt.Sprintf(`(function() { return document.querySelector(%q) !== null; })()`, step.Selector)
	if err := r.exec.Run(ctx, chromedp.Evaluate(checkJS, &exists)); err != nil {
		return fmt.Errorf("if_exists: check element: %w", err)
	}
	result.ExtractedData["_if_exists_result"] = strconv.FormatBool(exists)
	if !exists && step.JumpTo != "" {
		return r.jumpToLabel(step.JumpTo, result)
	}
	return nil
}

func (r *Runner) execIfNotExists(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if err := requireSelector("if_not_exists", step.Selector); err != nil {
		return err
	}
	var exists bool
	checkJS := fmt.Sprintf(`(function() { return document.querySelector(%q) !== null; })()`, step.Selector)
	if err := r.exec.Run(ctx, chromedp.Evaluate(checkJS, &exists)); err != nil {
		return fmt.Errorf("if_not_exists: check element: %w", err)
	}
	result.ExtractedData["_if_not_exists_result"] = strconv.FormatBool(!exists)
	if exists && step.JumpTo != "" {
		return r.jumpToLabel(step.JumpTo, result)
	}
	return nil
}

func (r *Runner) execIfVisible(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if err := requireSelector("if_visible", step.Selector); err != nil {
		return err
	}
	var visible bool
	checkJS := fmt.Sprintf(`(function() {
		var el = document.querySelector(%q);
		if (!el) return false;
		var style = window.getComputedStyle(el);
		return el.offsetWidth > 0 && el.offsetHeight > 0 && style.display !== 'none' && style.visibility !== 'hidden';
	})()`, step.Selector)
	if err := r.exec.Run(ctx, chromedp.Evaluate(checkJS, &visible)); err != nil {
		return fmt.Errorf("if_visible: check visibility: %w", err)
	}
	result.ExtractedData["_if_visible_result"] = strconv.FormatBool(visible)
	if !visible && step.JumpTo != "" {
		return r.jumpToLabel(step.JumpTo, result)
	}
	return nil
}

func (r *Runner) execIfEnabled(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if err := requireSelector("if_enabled", step.Selector); err != nil {
		return err
	}
	var enabled bool
	checkJS := fmt.Sprintf(`(function() {
		var el = document.querySelector(%q);
		if (!el) return false;
		return !el.disabled && !el.readOnly;
	})()`, step.Selector)
	if err := r.exec.Run(ctx, chromedp.Evaluate(checkJS, &enabled)); err != nil {
		return fmt.Errorf("if_enabled: check enabled: %w", err)
	}
	result.ExtractedData["_if_enabled_result"] = strconv.FormatBool(enabled)
	if !enabled && step.JumpTo != "" {
		return r.jumpToLabel(step.JumpTo, result)
	}
	return nil
}

func (r *Runner) execTryBlock(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	return nil
}

func (r *Runner) jumpToLabel(label string, result *models.TaskResult) error {
	result.ExtractedData["_jump_to_label"] = label
	return nil
}

func (r *Runner) execVariableSet(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if step.VarName == "" {
		return fmt.Errorf("variable_set: varName is required")
	}
	key := "var_" + step.VarName
	result.ExtractedData[key] = step.Value
	return nil
}

func (r *Runner) execVariableMath(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if step.VarName == "" {
		return fmt.Errorf("variable_math: varName is required")
	}
	key := "var_" + step.VarName
	currentVal := result.ExtractedData[key]
	if currentVal == "" {
		currentVal = "0"
	}
	current, err := strconv.ParseFloat(currentVal, 64)
	if err != nil {
		return fmt.Errorf("variable_math: invalid number %q: %w", currentVal, err)
	}
	var newVal float64
	switch step.Operator {
	case "+", "add":
		newVal = current + mustParseFloat(step.Value)
	case "-", "sub", "subtract":
		newVal = current - mustParseFloat(step.Value)
	case "*", "mul", "multiply":
		newVal = current * mustParseFloat(step.Value)
	case "/", "div", "divide":
		divisor := mustParseFloat(step.Value)
		if divisor == 0 {
			return fmt.Errorf("variable_math: division by zero")
		}
		newVal = current / divisor
	default:
		return fmt.Errorf("variable_math: unknown operator %q", step.Operator)
	}
	result.ExtractedData[key] = strconv.FormatFloat(newVal, 'f', -1, 64)
	return nil
}

func mustParseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func (r *Runner) execVariableString(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if step.VarName == "" {
		return fmt.Errorf("variable_string: varName is required")
	}
	key := "var_" + step.VarName
	currentVal := result.ExtractedData[key]
	switch step.Operator {
	case "concat", "append":
		result.ExtractedData[key] = currentVal + step.Value
	case "prepend":
		result.ExtractedData[key] = step.Value + currentVal
	case "replace":
		if len(step.Condition) > 0 {
			result.ExtractedData[key] = strings.ReplaceAll(currentVal, step.Condition, step.Value)
		}
	case "upper":
		result.ExtractedData[key] = strings.ToUpper(currentVal)
	case "lower":
		result.ExtractedData[key] = strings.ToLower(currentVal)
	case "trim":
		result.ExtractedData[key] = strings.TrimSpace(currentVal)
	case "length":
		result.ExtractedData[key] = strconv.Itoa(len(currentVal))
	case "substring":
		start, _ := strconv.Atoi(step.Condition)
		end := start + len(step.Value)
		if start < 0 {
			start = 0
		}
		if end > len(currentVal) {
			end = len(currentVal)
		}
		result.ExtractedData[key] = currentVal[start:end]
	default:
		result.ExtractedData[key] = currentVal
	}
	return nil
}

func (r *Runner) execHover(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("hover", step.Selector); err != nil {
		return err
	}
	hoverJS := fmt.Sprintf(`(function() {
		var el = document.querySelector(%q);
		if (!el) return false;
		var rect = el.getBoundingClientRect();
		var event = new MouseEvent('mouseover', {
			bubbles: true, cancelable: true, view: window,
			clientX: rect.left + rect.width / 2,
			clientY: rect.top + rect.height / 2
		});
		el.dispatchEvent(event);
		return true;
	})()`, step.Selector)
	var success bool
	if err := r.exec.Run(ctx, chromedp.Evaluate(hoverJS, &success)); err != nil {
		return fmt.Errorf("hover: execute hover: %w", err)
	}
	return nil
}

func (r *Runner) execDragDrop(ctx context.Context, step models.TaskStep) error {
	if step.Selector == "" {
		return fmt.Errorf("drag_drop: source selector is required")
	}
	if step.Target == "" {
		return fmt.Errorf("drag_drop: target selector is required")
	}
	dragJS := fmt.Sprintf(`(function() {
		var source = document.querySelector(%q);
		var target = document.querySelector(%q);
		if (!source || !target) return false;
		var sourceRect = source.getBoundingClientRect();
		var targetRect = target.getBoundingClientRect();
		var sourceX = sourceRect.left + sourceRect.width / 2;
		var sourceY = sourceRect.top + sourceRect.height / 2;
		var targetX = targetRect.left + targetRect.width / 2;
		var targetY = targetRect.top + targetRect.height / 2;
		var dispatchDragEvent = function(el, type, x, y) {
			var event = new MouseEvent(type, {
				bubbles: true, cancelable: true, view: window,
				clientX: x, clientY: y,
				detail: 0, button: 0, buttons: 1
			});
			el.dispatchEvent(event);
		};
		dispatchDragEvent(source, 'mousedown', sourceX, sourceY);
		dispatchDragEvent(source, 'mousemove', targetX, targetY);
		dispatchDragEvent(target, 'mouseover', targetX, targetY);
		dispatchDragEvent(target, 'mouseenter', targetX, targetY);
		dispatchDragEvent(target, 'dragenter', targetX, targetY);
		dispatchDragEvent(target, 'drop', targetX, targetY);
		dispatchDragEvent(source, 'mouseup', targetX, targetY);
		return true;
	})()`, step.Selector, step.Target)
	var success bool
	if err := r.exec.Run(ctx, chromedp.Evaluate(dragJS, &success)); err != nil {
		return fmt.Errorf("drag_drop: execute drag: %w", err)
	}
	if !success {
		return fmt.Errorf("drag_drop: drag operation failed")
	}
	return nil
}

func (r *Runner) execContextClick(ctx context.Context, step models.TaskStep) error {
	if err := requireSelector("context_click", step.Selector); err != nil {
		return err
	}
	contextClickJS := fmt.Sprintf(`(function() {
		var el = document.querySelector(%q);
		if (!el) return false;
		var rect = el.getBoundingClientRect();
		var event = new MouseEvent('contextmenu', {
			bubbles: true, cancelable: true, view: window,
			clientX: rect.left + rect.width / 2,
			clientY: rect.top + rect.height / 2,
			button: 2
		});
		el.dispatchEvent(event);
		return true;
	})()`, step.Selector)
	var success bool
	if err := r.exec.Run(ctx, chromedp.Evaluate(contextClickJS, &success)); err != nil {
		return fmt.Errorf("context_click: execute context click: %w", err)
	}
	return nil
}

func (r *Runner) execHighlight(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if err := requireSelector("highlight", step.Selector); err != nil {
		return err
	}
	color := step.Value
	if color == "" {
		color = "yellow"
	}
	duration := 2000
	if step.Duration > 0 {
		duration = step.Duration
	}
	highlightJS := fmt.Sprintf(`(function() {
		var el = document.querySelector(%q);
		if (!el) return false;
		var originalOutline = el.style.outline;
		var originalBackground = el.style.backgroundColor;
		el.style.outline = '3px solid %s';
		el.style.backgroundColor = 'rgba(255, 255, 0, 0.3)';
		setTimeout(function() {
			el.style.outline = originalOutline;
			el.style.backgroundColor = originalBackground;
		}, %d);
		return true;
	})()`, step.Selector, color, duration)
	var success bool
	if err := r.exec.Run(ctx, chromedp.Evaluate(highlightJS, &success)); err != nil {
		return fmt.Errorf("highlight: execute highlight: %w", err)
	}
	result.ExtractedData["_highlight_result"] = strconv.FormatBool(success)
	return nil
}

func (r *Runner) execGetCookies(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	getCookiesJS := `document.cookie.split(';').map(function(c) {
		var parts = c.trim().split('=');
		return {name: parts[0], value: parts.slice(1).join('=')};
	})`
	var cookies []map[string]string
	if err := r.exec.Run(ctx, chromedp.Evaluate(getCookiesJS, &cookies)); err != nil {
		return fmt.Errorf("get_cookies: retrieve cookies: %w", err)
	}
	keyPrefix := step.VarName
	if keyPrefix == "" {
		keyPrefix = "cookie"
	}
	for i, c := range cookies {
		result.ExtractedData[fmt.Sprintf("%s_%d_name", keyPrefix, i)] = c["name"]
		result.ExtractedData[fmt.Sprintf("%s_%d_value", keyPrefix, i)] = c["value"]
	}
	result.ExtractedData[keyPrefix+"_count"] = strconv.Itoa(len(cookies))
	return nil
}

func (r *Runner) execSetCookie(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if step.Name == "" {
		return fmt.Errorf("set_cookie: name is required")
	}
	cookieStr := step.Name + "=" + step.Value
	if step.Path != "" {
		cookieStr += "; path=" + step.Path
	}
	if step.Domain != "" {
		cookieStr += "; domain=" + step.Domain
	}
	setCookieJS := fmt.Sprintf(`(function() { document.cookie = %q; return true; })()`, cookieStr)
	var success bool
	if err := r.exec.Run(ctx, chromedp.Evaluate(setCookieJS, &success)); err != nil {
		return fmt.Errorf("set_cookie: set cookie: %w", err)
	}
	result.ExtractedData["_set_cookie_"+step.Name] = strconv.FormatBool(success)
	return nil
}

func (r *Runner) execDeleteCookies(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	domain := step.Domain
	deleteCookieJS := fmt.Sprintf(`(function() {
		var cookies = document.cookie.split(';');
		var domain = %q;
		for (var i = 0; i < cookies.length; i++) {
			var name = cookies[i].trim().split('=')[0];
			if (domain === '' || name.indexOf(domain) !== -1) {
				document.cookie = name + '=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/';
			}
		}
		return true;
	})()`, domain)
	var success bool
	if err := r.exec.Run(ctx, chromedp.Evaluate(deleteCookieJS, &success)); err != nil {
		return fmt.Errorf("delete_cookies: delete cookies: %w", err)
	}
	result.ExtractedData["_delete_cookies_domain"] = domain
	return nil
}

func (r *Runner) execGetStorage(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	storageType := "localStorage"
	if step.Data == "session" {
		storageType = "sessionStorage"
	}
	key := step.VarName
	if key == "" {
		key = step.Selector
	}
	getJS := fmt.Sprintf(`(function() {
		var data = %s.getItem(%q);
		return data;
	})()`, storageType, key)
	var value string
	if err := r.exec.Run(ctx, chromedp.Evaluate(getJS, &value)); err != nil {
		return fmt.Errorf("get_storage: get storage item: %w", err)
	}
	result.ExtractedData["storage_"+key] = value
	return nil
}

func (r *Runner) execSetStorage(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if step.Selector == "" {
		return fmt.Errorf("set_storage: key is required")
	}
	storageType := "localStorage"
	if step.Data == "session" {
		storageType = "sessionStorage"
	}
	setJS := fmt.Sprintf(`(function() {
		%s.setItem(%q, %q);
		return true;
	})()`, storageType, step.Selector, step.Value)
	var success bool
	if err := r.exec.Run(ctx, chromedp.Evaluate(setJS, &success)); err != nil {
		return fmt.Errorf("set_storage: set storage item: %w", err)
	}
	result.ExtractedData["_set_storage_"+step.Selector] = strconv.FormatBool(success)
	return nil
}

func (r *Runner) execDeleteStorage(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if step.Selector == "" {
		return fmt.Errorf("delete_storage: key is required")
	}
	storageType := "localStorage"
	if step.Data == "session" {
		storageType = "sessionStorage"
	}
	deleteJS := fmt.Sprintf(`(function() {
		%s.removeItem(%q);
		return true;
	})()`, storageType, step.Selector)
	var success bool
	if err := r.exec.Run(ctx, chromedp.Evaluate(deleteJS, &success)); err != nil {
		return fmt.Errorf("delete_storage: delete storage item: %w", err)
	}
	result.ExtractedData["_delete_storage_"+step.Selector] = strconv.FormatBool(success)
	return nil
}

func (r *Runner) execDownload(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if err := requireSelector("download", step.Selector); err != nil {
		return err
	}
	downloadPath := step.Path
	if downloadPath == "" {
		downloadPath = r.screenshotDir
	}
	setDownloadJS := fmt.Sprintf(`(function() {
		var link = document.querySelector(%q);
		if (!link || !link.href) return false;
		return link.href;
	})()`, step.Selector)
	var url string
	if err := r.exec.Run(ctx, chromedp.Evaluate(setDownloadJS, &url)); err != nil {
		return fmt.Errorf("download: get download URL: %w", err)
	}
	result.ExtractedData["_download_url"] = url
	result.ExtractedData["_download_path"] = downloadPath
	return nil
}

func (r *Runner) execSelectRandom(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	if err := requireSelector("select_random", step.Selector); err != nil {
		return err
	}
	getOptionsJS := fmt.Sprintf(`(function() {
		var select = document.querySelector(%q);
		if (!select || select.tagName !== 'SELECT') return [];
		var options = [];
		for (var i = 0; i < select.options.length; i++) {
			options.push({index: i, value: select.options[i].value, text: select.options[i].text});
		}
		return options;
	})()`, step.Selector)
	var options []map[string]string
	if err := r.exec.Run(ctx, chromedp.Evaluate(getOptionsJS, &options)); err != nil {
		return fmt.Errorf("select_random: get options: %w", err)
	}
	if len(options) == 0 {
		return fmt.Errorf("select_random: no options found")
	}
	randomIndex := time.Now().UnixNano() % int64(len(options))
	selectedValue := options[randomIndex]["value"]
	if err := r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.SetValue(step.Selector, selectedValue, chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("select_random: select option: %w", err)
	}
	result.ExtractedData["_select_random_value"] = selectedValue
	result.ExtractedData["_select_random_index"] = strconv.FormatInt(randomIndex, 10)
	return nil
}

func (r *Runner) execDebugPause(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	result.ExtractedData["_debug_paused"] = "true"
	result.ExtractedData["_debug_step_index"] = strconv.Itoa(len(result.StepLogs))
	return nil
}

func (r *Runner) execDebugStep(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	result.ExtractedData["_debug_step"] = "executed"
	result.ExtractedData["_debug_step_index"] = strconv.Itoa(len(result.StepLogs))
	return nil
}

func (r *Runner) execDebugResume(ctx context.Context, step models.TaskStep, result *models.TaskResult) error {
	result.ExtractedData["_debug_paused"] = "false"
	return nil
}

func (r *Runner) execHeadless(ctx context.Context, step models.TaskStep) error {
	return nil
}
