package browser

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"flowpilot/internal/captcha"
	"flowpilot/internal/logs"
	"flowpilot/internal/models"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

const defaultTimeout = 30 * time.Second

const (
	// MaxEvalScriptSize is the maximum allowed length of an eval script in bytes.
	MaxEvalScriptSize = 10_000

	// maxEvalScriptSizeDisplay is used in error messages.
	maxEvalScriptSizeDisplay = "10000"
)

// ErrEvalScriptTooLarge is returned when an eval script exceeds MaxEvalScriptSize.
var ErrEvalScriptTooLarge = fmt.Errorf("eval script exceeds maximum allowed size of %s bytes", maxEvalScriptSizeDisplay)

// ErrEvalScriptEmpty is returned when an eval script is empty.
var ErrEvalScriptEmpty = errors.New("eval script must not be empty")

// dangerousPatterns are blocked in eval scripts to reduce sandbox-escape risk.
var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bchild_process\b`),
	regexp.MustCompile(`(?i)\brequire\s*\(`),
	regexp.MustCompile(`(?i)\bprocess\.exit\b`),
	regexp.MustCompile(`(?i)\bprocess\.env\b`),
	regexp.MustCompile(`(?i)\bfs\s*\.\s*(read|write|unlink|mkdir|rmdir)`),
	regexp.MustCompile(`(?i)\b__dirname\b`),
	regexp.MustCompile(`(?i)\b__filename\b`),
}

// validateEvalScript checks an eval script for size, emptiness, and dangerous patterns.
func validateEvalScript(script string) error {
	if strings.TrimSpace(script) == "" {
		return ErrEvalScriptEmpty
	}
	if len(script) > MaxEvalScriptSize {
		return ErrEvalScriptTooLarge
	}
	for _, pat := range dangerousPatterns {
		if pat.MatchString(script) {
			return fmt.Errorf("eval script contains blocked pattern: %s", pat.String())
		}
	}
	return nil
}

// Runner executes browser automation tasks using chromedp.
type Runner struct {
	screenshotDir string
	allowEval     atomic.Bool
	forceHeadless atomic.Bool
	exec          Executor
	captchaSolver captcha.Solver
}

// NewRunner creates a new browser runner. Eval steps are blocked by default.
func NewRunner(screenshotDir string) (*Runner, error) {
	if err := os.MkdirAll(screenshotDir, 0o700); err != nil {
		return nil, fmt.Errorf("create screenshot dir: %w", err)
	}
	r := &Runner{screenshotDir: screenshotDir, exec: chromeExecutor{}}
	r.allowEval.Store(false)
	return r, nil
}

// SetForceHeadless enforces headless mode on all tasks when enabled.
func (r *Runner) SetForceHeadless(force bool) {
	r.forceHeadless.Store(force)
}

// SetCaptchaSolver sets the CAPTCHA solver used by solve_captcha steps.
func (r *Runner) SetCaptchaSolver(solver captcha.Solver) {
	r.captchaSolver = solver
}

// SetAllowEval configures whether the runner permits eval step execution.
func (r *Runner) SetAllowEval(allow bool) {
	r.allowEval.Store(allow)
}

// RunTask executes a single task with its own browser context and proxy.
func (r *Runner) RunTask(ctx context.Context, task models.Task) (*models.TaskResult, error) {
	start := time.Now()
	result := &models.TaskResult{
		TaskID:        task.ID,
		ExtractedData: make(map[string]string),
	}

	allocCtx, allocCancel := r.createAllocator(ctx, task.Proxy, task.Headless)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	netLogger := logs.NewNetworkLogger(task.ID)
	chromedp.ListenTarget(browserCtx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			netLogger.HandleRequestWillBeSent(e)
		case *network.EventResponseReceived:
			netLogger.HandleResponseReceived(e)
		case *network.EventLoadingFinished:
			netLogger.HandleLoadingFinished(e, nil)
		case *network.EventLoadingFailed:
			netLogger.HandleLoadingFailed(e.RequestID)
		}
	})

	if err := chromedp.Run(browserCtx, network.Enable()); err != nil {
		r.addLog(result, "warn", fmt.Sprintf("enable network logging: %v", err))
	}

	if err := ClearCookies(browserCtx); err != nil {
		r.addLog(result, "warn", fmt.Sprintf("clear cookies: %v", err))
	}

	if task.Proxy.Username != "" {
		if err := r.setupProxyAuth(browserCtx, task.Proxy); err != nil {
			result.Duration = time.Since(start)
			result.Error = fmt.Sprintf("proxy auth setup failed: %v", err)
			r.addLog(result, "error", result.Error)
			return result, err
		}
	}

	if err := r.runSteps(browserCtx, task.Steps, result, netLogger); err != nil {
		result.NetworkLogs = netLogger.Logs()
		result.Duration = time.Since(start)
		return result, err
	}

	result.NetworkLogs = netLogger.Logs()
	result.Success = true
	result.Duration = time.Since(start)
	r.addLog(result, "info", fmt.Sprintf("task completed in %s", result.Duration))
	return result, nil
}

// createAllocator builds a chromedp allocator with safe option copying and optional proxy.
// The headless parameter respects the task's preference unless forceHeadless is enabled.
func (r *Runner) createAllocator(ctx context.Context, proxyConfig models.ProxyConfig, headless bool) (context.Context, context.CancelFunc) {
	// Copy default options to avoid mutating the shared slice.
	opts := make([]chromedp.ExecAllocatorOption, len(chromedp.DefaultExecAllocatorOptions))
	copy(opts, chromedp.DefaultExecAllocatorOptions[:])

	// Respect forceHeadless override; otherwise use the task's headless preference.
	useHeadless := headless
	if r.forceHeadless.Load() {
		useHeadless = true
	}

	opts = append(opts,
		chromedp.Flag("headless", useHeadless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("enable-unsafe-swiftshader", true),
		chromedp.Flag("js-flags", "--max-old-space-size=512"),
	)

	if proxyConfig.Server != "" {
		proxyAddr := proxyConfig.Server
		if proxyConfig.Protocol != "" && proxyConfig.Protocol != models.ProxyHTTP {
			proxyAddr = string(proxyConfig.Protocol) + "://" + proxyConfig.Server
		}
		opts = append(opts, chromedp.ProxyServer(proxyAddr))
	}

	return chromedp.NewExecAllocator(ctx, opts...)
}

// runSteps executes task steps using a program-counter (PC) based approach
// to support conditional logic, loops, and goto jumps.
func (r *Runner) runSteps(browserCtx context.Context, steps []models.TaskStep, result *models.TaskResult, netLogger *logs.NetworkLogger) error {
	stepLogger := logs.NewStepLogger(result.TaskID)
	defer func() { result.StepLogs = stepLogger.Logs() }()

	labelIndex := buildLabelIndex(steps)

	type loopFrame struct {
		startPC     int
		maxIter     int
		currentIter int
	}
	var loopStack []loopFrame

	vars := result.ExtractedData

	pc := 0
	for pc < len(steps) {
		step := steps[pc]
		netLogger.SetStepIndex(pc)
		r.addLog(result, "info", fmt.Sprintf("step %d: %s", pc+1, step.Action))

		switch step.Action {
		case models.ActionLoop:
			maxIter, _ := strconv.Atoi(step.Value)
			if maxIter <= 0 {
				maxIter = 100
			}
			loopStack = append(loopStack, loopFrame{startPC: pc, maxIter: maxIter, currentIter: 0})
			pc++
			continue

		case models.ActionEndLoop:
			if len(loopStack) == 0 {
				return fmt.Errorf("step %d: end_loop without matching loop", pc+1)
			}
			top := &loopStack[len(loopStack)-1]
			top.currentIter++
			if top.currentIter < top.maxIter {
				pc = top.startPC + 1
				continue
			}
			loopStack = loopStack[:len(loopStack)-1]
			pc++
			continue

		case models.ActionBreakLoop:
			if len(loopStack) == 0 {
				return fmt.Errorf("step %d: break_loop without matching loop", pc+1)
			}
			endPC := findEndLoop(steps, loopStack[len(loopStack)-1].startPC)
			if endPC < 0 {
				return fmt.Errorf("step %d: no matching end_loop found", pc+1)
			}
			loopStack = loopStack[:len(loopStack)-1]
			pc = endPC + 1
			continue

		case models.ActionGoto:
			target, ok := labelIndex[step.JumpTo]
			if !ok {
				return fmt.Errorf("step %d: goto label %q not found", pc+1, step.JumpTo)
			}
			pc = target
			continue

		case models.ActionIfElement, models.ActionIfText, models.ActionIfURL:
			condMet, err := r.evaluateCondition(browserCtx, step, vars)
			if err != nil {
				r.addLog(result, "warn", fmt.Sprintf("step %d condition error: %v", pc+1, err))
				condMet = false
			}
			if condMet && step.JumpTo != "" {
				target, ok := labelIndex[step.JumpTo]
				if !ok {
					return fmt.Errorf("step %d: jumpTo label %q not found", pc+1, step.JumpTo)
				}
				pc = target
				continue
			}
			pc++
			continue
		}

		timeout := defaultTimeout
		if step.Timeout > 0 {
			timeout = time.Duration(step.Timeout) * time.Millisecond
		}
		start := stepLogger.StartStep(pc, step.Action, step.Selector, step.Value, "")
		stepCtx, stepCancel := context.WithTimeout(browserCtx, timeout)
		err := r.executeStep(stepCtx, step, result)
		stepCancel()

		var code models.ErrorCode
		if err != nil {
			code = models.ClassifyError(err)
		}
		stepLogger.EndStep(pc, step.Action, step.Selector, step.Value, "", start, err, code)

		if err != nil {
			r.addLog(result, "error", fmt.Sprintf("step %d failed: %v", pc+1, err))
			result.Error = fmt.Sprintf("step %d (%s) failed: %v", pc+1, step.Action, err)
			return err
		}

		if step.Action == models.ActionExtract && step.VarName != "" {
			if val, ok := result.ExtractedData[step.Value]; ok {
				vars[step.VarName] = val
			} else if val, ok := result.ExtractedData[step.Selector]; ok {
				vars[step.VarName] = val
			}
		}

		r.addLog(result, "info", fmt.Sprintf("step %d completed", pc+1))
		pc++
	}
	return nil
}

func (r *Runner) setupProxyAuth(ctx context.Context, proxyConfig models.ProxyConfig) error {
	chromedp.ListenTarget(ctx, func(ev any) {
		switch e := ev.(type) {
		case *fetch.EventAuthRequired:
			go func() {
				execCtx := chromedp.FromContext(ctx)
				if execCtx == nil || execCtx.Target == nil {
					return
				}
				c := cdp.WithExecutor(ctx, execCtx.Target)
				if err := fetch.ContinueWithAuth(e.RequestID, &fetch.AuthChallengeResponse{
					Response: fetch.AuthChallengeResponseResponseProvideCredentials,
					Username: proxyConfig.Username,
					Password: proxyConfig.Password,
				}).Do(c); err != nil {
					log.Printf("proxy auth continue failed: %v", err)
				}
			}()
		case *fetch.EventRequestPaused:
			go func() {
				execCtx := chromedp.FromContext(ctx)
				if execCtx == nil || execCtx.Target == nil {
					return
				}
				c := cdp.WithExecutor(ctx, execCtx.Target)
				if err := fetch.ContinueRequest(e.RequestID).Do(c); err != nil {
					log.Printf("proxy request continue failed: %v", err)
				}
			}()
		}
	})

	if err := r.exec.Run(ctx, fetch.Enable().WithHandleAuthRequests(true)); err != nil {
		return fmt.Errorf("enable fetch for proxy auth: %w", err)
	}
	return nil
}

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
	return r.exec.Run(ctx, chromedp.Navigate(step.Value))
}

func (r *Runner) execClick(ctx context.Context, step models.TaskStep) error {
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.Click(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execType(ctx context.Context, step models.TaskStep) error {
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
	// Respect context cancellation during wait.
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
	if err := r.exec.Run(ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
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
	// Validate the scroll value is a number to prevent JS injection.
	if _, err := strconv.Atoi(step.Value); err != nil {
		return fmt.Errorf("invalid scroll value %q: must be an integer", step.Value)
	}
	return r.exec.Run(ctx,
		chromedp.Evaluate(`window.scrollBy(0, `+step.Value+`)`, nil),
	)
}

func (r *Runner) execSelect(ctx context.Context, step models.TaskStep) error {
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.SetValue(step.Selector, step.Value, chromedp.ByQuery),
	)
}

var ErrEvalNotAllowed = errors.New("eval action is not allowed: runner has allowEval=false")

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
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.DoubleClick(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execFileUpload(ctx context.Context, step models.TaskStep) error {
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
	return r.exec.Run(ctx,
		chromedp.ScrollIntoView(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execSubmitForm(ctx context.Context, step models.TaskStep) error {
	return r.exec.Run(ctx,
		chromedp.WaitVisible(step.Selector, chromedp.ByQuery),
		chromedp.Submit(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execWaitNotPresent(ctx context.Context, step models.TaskStep) error {
	return r.exec.Run(ctx,
		chromedp.WaitNotPresent(step.Selector, chromedp.ByQuery),
	)
}

func (r *Runner) execWaitEnabled(ctx context.Context, step models.TaskStep) error {
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

func (r *Runner) addLog(result *models.TaskResult, level, message string) {
	result.Logs = append(result.Logs, models.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
	})
}

// ClearCookies clears cookies in a browser context.
func ClearCookies(ctx context.Context) error {
	return chromedp.Run(ctx, network.ClearBrowserCookies())
}

func buildLabelIndex(steps []models.TaskStep) map[string]int {
	idx := make(map[string]int, len(steps))
	for i, s := range steps {
		if s.Label != "" {
			idx[s.Label] = i
		}
	}
	return idx
}

func findEndLoop(steps []models.TaskStep, loopPC int) int {
	depth := 0
	for i := loopPC; i < len(steps); i++ {
		if steps[i].Action == models.ActionLoop {
			depth++
		}
		if steps[i].Action == models.ActionEndLoop {
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func (r *Runner) evaluateCondition(ctx context.Context, step models.TaskStep, vars map[string]string) (bool, error) {
	switch step.Action {
	case models.ActionIfElement:
		var nodes []*cdp.Node
		err := r.exec.Run(ctx, chromedp.Nodes(step.Selector, &nodes, chromedp.ByQuery, chromedp.AtLeast(0)))
		if err != nil {
			return false, fmt.Errorf("if_element check: %w", err)
		}
		switch step.Condition {
		case "not_exists":
			return len(nodes) == 0, nil
		default:
			return len(nodes) > 0, nil
		}

	case models.ActionIfText:
		var text string
		if err := r.exec.Run(ctx,
			chromedp.Text(step.Selector, &text, chromedp.ByQuery),
		); err != nil {
			return false, nil
		}
		return evaluateTextCondition(step.Condition, text, vars)

	case models.ActionIfURL:
		var currentURL string
		if err := r.exec.Run(ctx, chromedp.Location(&currentURL)); err != nil {
			return false, fmt.Errorf("if_url get location: %w", err)
		}
		return evaluateTextCondition(step.Condition, currentURL, vars)

	default:
		return false, fmt.Errorf("unknown condition action: %s", step.Action)
	}
}

func evaluateTextCondition(condition, text string, vars map[string]string) (bool, error) {
	for k, v := range vars {
		condition = strings.ReplaceAll(condition, "{{"+k+"}}", v)
	}

	parts := strings.SplitN(condition, ":", 2)
	if len(parts) != 2 {
		return strings.Contains(text, condition), nil
	}

	op, val := parts[0], parts[1]
	switch op {
	case "contains":
		return strings.Contains(text, val), nil
	case "not_contains":
		return !strings.Contains(text, val), nil
	case "equals":
		return text == val, nil
	case "not_equals":
		return text != val, nil
	case "starts_with":
		return strings.HasPrefix(text, val), nil
	case "ends_with":
		return strings.HasSuffix(text, val), nil
	case "matches":
		re, err := regexp.Compile(val)
		if err != nil {
			return false, fmt.Errorf("invalid regex in condition: %w", err)
		}
		return re.MatchString(text), nil
	default:
		return false, fmt.Errorf("unknown condition operator: %s", op)
	}
}
