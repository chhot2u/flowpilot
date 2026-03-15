# Prompt: Optimize FlowPilot for 100+ Concurrent Tasks — Variant C (Maximum Precision)

<system>
You are a Go concurrency optimization specialist. You will receive a series of precisely-defined code changes to apply to the FlowPilot codebase. Each change includes the exact file, function, current state, target state, and acceptance criteria.

## Mandatory Tool Usage Protocol

Before modifying ANY code, you MUST run `cocoindex-code_search` with a query targeting the specific function/struct you plan to change. This is non-negotiable. Example:

```
cocoindex-code_search(query="func New dbPath sql.Open MaxOpenConns WAL", limit=5)
```

If the search results don't match the expected current state described below, STOP and report the discrepancy before proceeding.

## Global Constraints — Violations Are Failures

- ❌ Do NOT create new files
- ❌ Do NOT add external Go dependencies
- ❌ Do NOT change exported method signatures on `App` struct
- ❌ Do NOT delete or rename any existing test
- ❌ Do NOT use `go:generate` or code generation
- ❌ Do NOT change the chromedp or SQLite libraries
- ❌ Do NOT use `panic()` or `log.Fatal()` in non-main packages
- ✅ DO preserve error wrapping pattern: `fmt.Errorf("context: %w", err)`
- ✅ DO preserve import grouping: stdlib → external → internal
- ✅ DO run `go test -tags=dev -race ./...` after each change set
- ✅ DO use `cocoindex-code_search` before every edit
</system>

---

## CHANGE 1: SQLite High-Concurrency Configuration

### Discovery
```
cocoindex-code_search(query="database DB struct conn sql.DB New dbPath MaxOpenConns", limit=5)
```

### File: `internal/database/sqlite.go`

### Current State
```go
type DB struct {
    conn *sql.DB
}

func New(dbPath string) (*DB, error) {
    conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
    // ...
    conn.SetMaxOpenConns(1)
    conn.SetMaxIdleConns(1)
```

### Target State
```go
type DB struct {
    conn     *sql.DB // write-only connection
    readConn *sql.DB // read-only connection for concurrent queries
}

func New(dbPath string) (*DB, error) {
    dsn := dbPath + "?_journal_mode=WAL&_busy_timeout=30000&_txlock=immediate"
    conn, err := sql.Open("sqlite3", dsn)
    // ...
    conn.SetMaxOpenConns(1)
    conn.SetMaxIdleConns(1)

    readConn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&mode=ro")
    // ...
    readConn.SetMaxOpenConns(4)
    readConn.SetMaxIdleConns(4)

    db := &DB{conn: conn, readConn: readConn}

    // Performance PRAGMAs (apply to both connections)
    for _, c := range []*sql.DB{conn, readConn} {
        c.Exec("PRAGMA synchronous=NORMAL")
        c.Exec("PRAGMA cache_size=-64000")
        c.Exec("PRAGMA mmap_size=268435456")
        c.Exec("PRAGMA temp_store=MEMORY")
    }
```

### Also Required
- Update `Close()` to close both connections
- Add `func (db *DB) Reader() *sql.DB { return db.readConn }`
- Migrate all read-only methods (`List*`, `Get*`, `Count*`, `GetBatchProgress`) in `db_tasks.go`, `db_batch.go`, `db_logs.go` to use `db.readConn.QueryContext` instead of `db.conn.QueryContext`

### Acceptance Criteria
- [ ] Two separate `*sql.DB` connections exist
- [ ] Write conn: MaxOpenConns=1, busy_timeout=30000
- [ ] Read conn: MaxOpenConns=4, mode=ro
- [ ] PRAGMAs applied to both
- [ ] `Close()` closes both
- [ ] All `List*`/`Get*` use readConn
- [ ] `go test -tags=dev ./internal/database/...` passes

---

## CHANGE 2: Composite Database Indexes

### Discovery
```
cocoindex-code_search(query="CREATE INDEX idx_tasks_status idx_tasks_priority migrate schema", limit=5)
```

### File: `internal/database/sqlite.go`, function `migrate()`

### Add After Existing Indexes
```sql
CREATE INDEX IF NOT EXISTS idx_tasks_status_priority ON tasks(status, priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_tasks_batch_status ON tasks(batch_id, status);
CREATE INDEX IF NOT EXISTS idx_network_logs_task_step ON network_logs(task_id, step_index);
CREATE INDEX IF NOT EXISTS idx_tasks_created ON tasks(created_at);
CREATE INDEX IF NOT EXISTS idx_step_logs_task_step ON step_logs(task_id, step_index);
```

### Acceptance Criteria
- [ ] 5 new composite indexes added
- [ ] `go test -tags=dev ./internal/database/...` passes

---

## CHANGE 3: Queue O(1) Task Lookup

### Discovery
```
cocoindex-code_search(query="isTaskInHeap isTaskEnqueued running map cancelled map heap", limit=5)
```

### File: `internal/queue/queue.go`

### Current State
```go
type Queue struct {
    // ... existing fields ...
    pq       taskHeap
    pausedPQ taskHeap
}

func (q *Queue) isTaskInHeap(taskID string) bool {
    for _, item := range q.pq { // O(n)
        if item.task.ID == taskID { return true }
    }
    for _, item := range q.pausedPQ { // O(n)
        if item.task.ID == taskID { return true }
    }
    return false
}
```

### Target State
```go
type Queue struct {
    // ... existing fields ...
    pq          taskHeap
    pausedPQ    taskHeap
    heapSet     map[string]struct{} // O(1) lookup for pq
    pausedSet   map[string]struct{} // O(1) lookup for pausedPQ
}

func (q *Queue) isTaskInHeap(taskID string) bool {
    _, inMain := q.heapSet[taskID]
    _, inPaused := q.pausedSet[taskID]
    return inMain || inPaused
}
```

### Also Required
- Initialize `heapSet` and `pausedSet` in `New()`: `make(map[string]struct{}, maxConcurrency*10)`
- Update ALL `heap.Push(&q.pq, item)` calls to also do `q.heapSet[item.task.ID] = struct{}{}`
- Update ALL `heap.Pop(&q.pq)` calls to also do `delete(q.heapSet, item.task.ID)`
- Update ALL `heap.Remove(&q.pq, i)` calls to also do `delete(q.heapSet, item.task.ID)`
- Same pattern for `pausedPQ` ↔ `pausedSet`
- Update `removeFromHeap` and `removeFromPausedHeap` accordingly
- Update `ResumeBatch` when moving items between heaps

### Acceptance Criteria
- [ ] `isTaskInHeap` is O(1)
- [ ] All heap mutations keep sets in sync
- [ ] `go test -tags=dev ./internal/queue/...` passes

---

## CHANGE 4: Bulk SubmitBatch

### Discovery
```
cocoindex-code_search(query="SubmitBatch Submit loop sequential tasks error", limit=5)
```

### File: `internal/queue/queue.go`

### Current State
```go
func (q *Queue) SubmitBatch(ctx context.Context, tasks []models.Task) error {
    for _, task := range tasks {
        if err := q.Submit(ctx, task); err != nil {
            return fmt.Errorf("submit task %s: %w", task.ID, err)
        }
    }
    return nil
}
```

### Target State
```go
func (q *Queue) SubmitBatch(ctx context.Context, tasks []models.Task) error {
    q.mu.Lock()
    if q.stopped {
        q.mu.Unlock()
        return fmt.Errorf("queue is stopped")
    }

    // Check capacity once
    if q.maxPending > 0 && q.pq.Len()+q.pausedPQ.Len()+len(tasks) > q.maxPending {
        q.mu.Unlock()
        return ErrQueueFull
    }

    // Bulk enqueue
    added := make([]models.Task, 0, len(tasks))
    for _, task := range tasks {
        if q.isTaskEnqueued(task.ID) || q.isTaskInHeap(task.ID) {
            continue // skip duplicates silently in batch mode
        }
        taskCtx, cancel := context.WithCancel(ctx)
        item := &heapItem{
            task:    task,
            ctx:     taskCtx,
            cancel:  cancel,
            addedAt: time.Now(),
        }
        heap.Push(&q.pq, item)
        q.heapSet[item.task.ID] = struct{}{}
        q.metrics.TotalSubmitted++
        added = append(added, task)
    }
    q.mu.Unlock()

    // Batch DB update outside lock
    taskIDs := make([]string, len(added))
    for i, t := range added {
        taskIDs[i] = t.ID
    }
    if err := q.db.BatchUpdateTaskStatus(ctx, taskIDs, models.TaskStatusQueued, ""); err != nil {
        // Rollback: remove from heap
        q.mu.Lock()
        for _, t := range added {
            q.removeFromHeap(t.ID)
        }
        q.mu.Unlock()
        return fmt.Errorf("batch update task status: %w", err)
    }

    // Emit events and wake workers
    for _, t := range added {
        q.emitEvent(t.ID, models.TaskStatusQueued, "")
    }
    q.cond.Broadcast()
    return nil
}
```

### Also Required
- Add `BatchUpdateTaskStatus(ctx, taskIDs []string, status, msg)` to `internal/database/db_tasks.go` using a single transaction with prepared statement
- Existing `SubmitBatch` tests must still pass (the function signature is unchanged)

### Acceptance Criteria
- [ ] Single lock acquisition for all tasks
- [ ] Single DB transaction for all status updates
- [ ] Single Broadcast instead of N Signals
- [ ] Duplicate tasks skipped (not errored)
- [ ] `go test -tags=dev ./internal/queue/...` passes

---

## CHANGE 5: BrowserPool Wait Queue

### Discovery
```
cocoindex-code_search(query="BrowserPool Acquire exhausted all browsers at max tab capacity", limit=5)
```

### File: `internal/browser/pool.go`

### Changes
- Add `cond *sync.Cond` and `acquireTimeout time.Duration` to `BrowserPool` struct
- Initialize `cond` in `NewBrowserPool`: `p.cond = sync.NewCond(&p.mu)`
- In `Acquire()`: replace the final `return nil, nil, fmt.Errorf("browser pool exhausted...")` with a timed wait loop using `cond.Wait()` + context deadline check
- In the `release` closure: add `p.cond.Signal()` after decrementing `inUse`
- Increase `MaxPoolSize` from 50 to 200
- Add `AcquireTimeout` to `PoolConfig` (default 60s)

### Acceptance Criteria
- [ ] `Acquire()` blocks (up to timeout) instead of failing immediately
- [ ] `release()` signals waiting acquirers
- [ ] MaxPoolSize = 200
- [ ] `go test -tags=dev ./internal/browser/...` passes

---

## CHANGE 6: RunTask Uses BrowserPool

### Discovery
```
cocoindex-code_search(query="RunTask createAllocator allocCtx allocCancel NewContext browserCtx", limit=5)
```

### File: `internal/browser/browser.go`

### Changes
- Add `pool *BrowserPool` field to `Runner` struct
- Add `func (r *Runner) SetPool(p *BrowserPool) { r.pool = p }`
- In `RunTask`: if `r.pool != nil`, acquire browser context from pool instead of `createAllocator`
- Ensure `release()` is deferred correctly

### Also Required
- In `app.go` `startup()`: create BrowserPool, call `runner.SetPool(pool)`

### Acceptance Criteria
- [ ] RunTask uses pool when available
- [ ] Falls back to createAllocator when pool is nil
- [ ] No Chrome process leak (release always called)
- [ ] `go test -tags=dev ./internal/browser/...` passes

---

## CHANGE 7: Chrome High-Concurrency Flags

### Discovery
```
cocoindex-code_search(query="createAllocator chromedp.Flag disable-dev-shm DisableGPU opts append", limit=5)
```

### File: `internal/browser/browser.go`, function `createAllocator`

### Add After Existing Flags
```go
chromedp.Flag("disable-background-networking", true),
chromedp.Flag("disable-default-apps", true),
chromedp.Flag("disable-extensions", true),
chromedp.Flag("disable-sync", true),
chromedp.Flag("disable-translate", true),
chromedp.Flag("no-first-run", true),
chromedp.Flag("disable-background-timer-throttling", true),
chromedp.Flag("disable-renderer-backgrounding", true),
chromedp.Flag("disable-backgrounding-occluded-windows", true),
```

### Acceptance Criteria
- [ ] 9 new Chrome flags added
- [ ] Existing flags preserved
- [ ] `go test -tags=dev ./internal/browser/...` passes

---

## CHANGE 8: NetworkLogger Pre-allocation

### Discovery
```
cocoindex-code_search(query="NewNetworkLogger make map startTimes requests responses logs", limit=5)
```

### File: `internal/logs/network.go`

### Changes
```go
func NewNetworkLogger(taskID string) *NetworkLogger {
    return &NetworkLogger{
        taskID:     taskID,
        startTimes: make(map[network.RequestID]time.Time, 64),
        requests:   make(map[network.RequestID]*network.Request, 64),
        responses:  make(map[network.RequestID]*network.Response, 64),
        logs:       make([]models.NetworkLog, 0, 128),
    }
}
```

### Acceptance Criteria
- [ ] All maps pre-allocated with capacity 64
- [ ] Logs slice pre-allocated with capacity 128
- [ ] `go test -tags=dev ./internal/logs/...` passes

---

## CHANGE 9: Worker Stagger Cap

### Discovery
```
cocoindex-code_search(query="stagger startup 100ms per worker workerID Chrome launch", limit=5)
```

### File: `internal/queue/queue.go`, function `New`

### Current
```go
time.Sleep(time.Duration(workerID) * 100 * time.Millisecond)
```

### Target
```go
stagger := time.Duration(workerID) * 50 * time.Millisecond
if stagger > 2*time.Second {
    stagger = 2 * time.Second
}
time.Sleep(stagger)
```

### Acceptance Criteria
- [ ] Max stagger is 2 seconds regardless of worker count
- [ ] Per-worker stagger reduced from 100ms to 50ms
- [ ] `go test -tags=dev ./internal/queue/...` passes

---

## CHANGE 10: AppConfig Defaults

### Discovery
```
cocoindex-code_search(query="DefaultAppConfig QueueConcurrency RetentionDays AppConfig struct", limit=5)
```

### File: `app.go`

### Changes
```go
type AppConfig struct {
    QueueConcurrency    int
    BrowserPoolSize     int  // NEW
    BrowserMaxTabs      int  // NEW
    RetentionDays       int
    HealthCheckInterval int
    MaxProxyFailures    int
}

func DefaultAppConfig() AppConfig {
    return AppConfig{
        QueueConcurrency:    50,  // was 10
        BrowserPoolSize:     20,  // NEW
        BrowserMaxTabs:      15,  // NEW
        RetentionDays:       90,
        HealthCheckInterval: 300,
        MaxProxyFailures:    3,
    }
}
```

Wire in `startup()`:
```go
pool := browser.NewBrowserPool(browser.PoolConfig{
    Size:    a.config.BrowserPoolSize,
    MaxTabs: a.config.BrowserMaxTabs,
}, chromedp.DefaultExecAllocatorOptions[:])
a.runner.SetPool(pool)
```

### Acceptance Criteria
- [ ] QueueConcurrency default = 50
- [ ] BrowserPool created and wired
- [ ] `go test -tags=dev ./...` passes

---

## Final Gate

```sh
go vet -tags=dev ./...
go test -tags=dev -race -count=1 ./...
```

**ALL tests MUST pass with `-race` flag.** If any test fails, fix the implementation — never delete tests.
