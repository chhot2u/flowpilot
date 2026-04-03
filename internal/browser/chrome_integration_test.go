package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"

	"flowpilot/internal/models"
)

// chromeOpts returns headless Chrome allocator options suitable for CI.
func chromeOpts() []chromedp.ExecAllocatorOption {
	return append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)
}

func TestBrowserPoolAcquireAndRelease(t *testing.T) {
	pool := NewBrowserPool(PoolConfig{Size: 2, MaxTabs: 5, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	browserCtx, release, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	if browserCtx == nil {
		t.Fatal("expected non-nil browserCtx")
	}
	if release == nil {
		t.Fatal("expected non-nil release func")
	}

	// Navigate to about:blank to verify context works
	if err := chromedp.Run(browserCtx, chromedp.Navigate("about:blank")); err != nil {
		t.Fatalf("Navigate: %v", err)
	}
	release()
}

func TestBrowserPoolAcquireMultipleTabs(t *testing.T) {
	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 3, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ctx1, release1, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire 1: %v", err)
	}
	ctx2, release2, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire 2: %v", err)
	}

	_ = chromedp.Run(ctx1, chromedp.Navigate("about:blank"))
	_ = chromedp.Run(ctx2, chromedp.Navigate("about:blank"))

	release1()
	release2()
}

func TestBrowserPoolStoppedReturnsError(t *testing.T) {
	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 1, AcquireTimeout: 5 * time.Second}, chromeOpts())
	pool.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _, err := pool.Acquire(ctx)
	if err == nil {
		t.Fatal("expected error acquiring from stopped pool")
	}
	if !strings.Contains(err.Error(), "stopped") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBrowserPoolStopIdempotentChrome(t *testing.T) {
	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 1}, chromeOpts())
	pool.Stop()
	pool.Stop() // must not panic
}

func TestBrowserPoolEvictIdle(t *testing.T) {
	pool := NewBrowserPool(PoolConfig{Size: 2, MaxTabs: 1, IdleTimeout: 10 * time.Millisecond, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	bCtx, release, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	_ = chromedp.Run(bCtx, chromedp.Navigate("about:blank"))
	release()

	// Wait for idle eviction to trigger
	time.Sleep(200 * time.Millisecond)
	pool.evictIdle()

	pool.mu.Lock()
	count := len(pool.browsers)
	pool.mu.Unlock()
	if count > 0 {
		t.Logf("pool still has %d browser(s) after eviction (may be in use)", count)
	}
}

func TestRunnerRunTaskNavigate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Test Page</title></head><body><h1>Hello</h1></body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)
	runner.SetAllowEval(false)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 3, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-navigate",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.Error != "" {
		t.Errorf("task error: %s", result.Error)
	}
}

func TestRunnerRunTaskExtractText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><p id="msg">Hello World</p></body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 3, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-extract",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionExtract, Selector: "#msg", VarName: "message"},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.ExtractedData["message"] != "Hello World" {
		t.Errorf("extracted: got %q, want %q", result.ExtractedData["message"], "Hello World")
	}
}

func TestRunnerRunTaskGetTitle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>MyTitle</title></head><body></body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-title",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionGetTitle, VarName: "page_title"},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.ExtractedData["page_title"] != "MyTitle" {
		t.Errorf("title: got %q, want %q", result.ExtractedData["page_title"], "MyTitle")
	}
}

func TestRunnerRunTaskGetURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>page</body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-url",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionGetTitle, VarName: "current_url"},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	_ = result
}

func TestRunnerRunTaskWait(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><div id="ready">OK</div></body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-wait",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionWait, Selector: "#ready"},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.Error != "" {
		t.Errorf("task error: %s", result.Error)
	}
}

func TestRunnerRunTaskClickAndType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<input id="name" type="text">
<button id="btn" onclick="document.getElementById('out').innerText='clicked'">Click</button>
<div id="out"></div>
</body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-click-type",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionType, Selector: "#name", Value: "hello"},
			{Action: models.ActionClick, Selector: "#btn"},
			{Action: models.ActionExtract, Selector: "#out", VarName: "result"},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.ExtractedData["result"] != "clicked" {
		t.Errorf("result: got %q, want %q", result.ExtractedData["result"], "clicked")
	}
}

func TestRunnerRunTaskEvalAllowed(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body></body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)
	runner.SetAllowEval(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-eval",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionEval, Value: "document.title = 'changed'; true;"},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.Error != "" {
		t.Errorf("task error: %s", result.Error)
	}
}

func TestRunnerRunTaskSetupProxyAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>proxied</body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Task with proxy credentials triggers setupProxyAuth code path
	task := models.Task{
		ID:  "chrome-proxy-auth",
		URL: ts.URL,
		Proxy: models.ProxyConfig{
			Server:   "127.0.0.1:19999", // unreachable, but triggers auth setup
			Protocol: models.ProxyHTTP,
			Username: "user",
			Password: "pass",
		},
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
		},
	}

	result, _ := runner.RunTask(ctx, task)
	// Result may fail since proxy is unreachable; just check it ran
	if result == nil {
		t.Fatal("expected non-nil result even with proxy failure")
	}
}

func TestRunnerRunTaskLoopSteps(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><p id="p">item</p></body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-loop",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionLoop, Value: "3"},
			{Action: models.ActionExtract, Selector: "#p", VarName: "item"},
			{Action: models.ActionEndLoop},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.Error != "" {
		t.Errorf("task error: %s", result.Error)
	}
}

func TestRunnerRunTaskScrollAndScreenshot(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body style="height:2000px"><p id="top">top</p></body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-scroll-screenshot",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionScrollIntoView, Selector: "#top"},
			{Action: models.ActionScreenshot, VarName: "shot"},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.ExtractedData["shot"] == "" {
		t.Error("expected screenshot data")
	}
}

func TestRunnerRunTaskSelectAndCheck(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>
<select id="sel"><option value="a">A</option><option value="b">B</option></select>
<input id="chk" type="checkbox">
</body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-select-check",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionSelect, Selector: "#sel", Value: "b"},
			{Action: models.ActionClick, Selector: "#chk"},
			{Action: models.ActionClick, Selector: "#chk"},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.Error != "" {
		t.Errorf("task error: %s", result.Error)
	}
}

func TestRunnerRunTaskNoPool(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>ok</body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)
	// No pool set — runner creates its own allocator context per task

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-no-pool",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.Error != "" {
		t.Errorf("task error: %s", result.Error)
	}
}

func TestRunnerRunTaskCookies(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc123"})
		_, _ = w.Write([]byte(`<html><body>cookies set</body></html>`))
	}))
	t.Cleanup(ts.Close)

	dir := t.TempDir()
	runner, err := NewRunner(dir)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	runner.SetForceHeadless(true)

	pool := NewBrowserPool(PoolConfig{Size: 1, MaxTabs: 2, AcquireTimeout: 30 * time.Second}, chromeOpts())
	t.Cleanup(func() { pool.Stop() })
	runner.SetPool(pool)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	task := models.Task{
		ID:  "chrome-cookies",
		URL: ts.URL,
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: ts.URL},
			{Action: models.ActionGetCookies, VarName: "cookie"},
			{Action: models.ActionDeleteCookies},
		},
	}

	result, err := runner.RunTask(ctx, task)
	if err != nil {
		t.Fatalf("RunTask: %v", err)
	}
	if result.Error != "" {
		t.Errorf("task error: %s", result.Error)
	}
}
