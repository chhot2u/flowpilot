package queue

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"flowpilot/internal/browser"
	"flowpilot/internal/database"
	"flowpilot/internal/models"
	"flowpilot/internal/proxy"
)

// EventCallback is invoked when a task's status changes.
type EventCallback func(event models.TaskEvent)

// ErrQueueFull is returned when the pending queue has reached its maximum size.
var ErrQueueFull = errors.New("queue is full: too many pending tasks")

// ErrBatchPaused is returned when attempting to submit to a paused batch.
var ErrBatchPaused = errors.New("batch is paused")

// Queue manages task scheduling, concurrency limiting, and execution using
// a fixed worker pool with a priority heap. Higher-priority tasks are
// dequeued first; within the same priority level, tasks are processed FIFO.
type Queue struct {
	db           *database.DB
	runner       *browser.Runner
	proxyManager *proxy.Manager
	workerCount  int
	maxPending   int
	onEvent      EventCallback
	metrics      models.QueueMetrics

	mu        sync.Mutex
	cond      *sync.Cond
	pq        taskHeap            // main priority queue
	pausedPQ  taskHeap            // tasks from paused batches
	heapSet   map[string]struct{} // O(1) lookup for pq
	pausedSet map[string]struct{} // O(1) lookup for pausedPQ
	running   map[string]context.CancelFunc
	cancelled map[string]bool
	paused    map[string]bool // batchID → paused
	stopped   bool
	stopOnce  sync.Once
	stopCh    chan struct{}
	workerWg  sync.WaitGroup
}

// New creates a Queue with the given concurrency limit and event callback.
// It spawns workerCount fixed workers with a staggered 100ms warm-up delay.
func New(db *database.DB, runner *browser.Runner, maxConcurrency int, onEvent EventCallback) *Queue {
	q := &Queue{
		db:          db,
		runner:      runner,
		workerCount: maxConcurrency,
		maxPending:  maxConcurrency * 10, // default: 10x concurrency limit
		onEvent:     onEvent,
		metrics:     models.QueueMetrics{},
		pq:          make(taskHeap, 0),
		pausedPQ:    make(taskHeap, 0),
		heapSet:     make(map[string]struct{}, maxConcurrency*10),
		pausedSet:   make(map[string]struct{}, maxConcurrency*10),
		running:     make(map[string]context.CancelFunc),
		cancelled:   make(map[string]bool),
		paused:      make(map[string]bool),
		stopCh:      make(chan struct{}),
	}
	q.cond = sync.NewCond(&q.mu)
	heap.Init(&q.pq)
	heap.Init(&q.pausedPQ)

	// Start fixed worker pool with staggered warm-up.
	for i := 0; i < maxConcurrency; i++ {
		q.workerWg.Add(1)
		workerID := i
		go func() {
			if workerID > 0 {
				stagger := time.Duration(workerID) * 50 * time.Millisecond
				if stagger > 2*time.Second {
					stagger = 2 * time.Second
				}
				time.Sleep(stagger)
			}
			q.worker(workerID)
		}()
	}

	return q
}

// SetProxyManager attaches a proxy manager for automatic proxy selection.
func (q *Queue) SetProxyManager(pm *proxy.Manager) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.proxyManager = pm
}

func (q *Queue) getProxyManager() *proxy.Manager {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.proxyManager
}

// Submit enqueues a task for execution. Returns ErrQueueFull if at capacity.
func (q *Queue) Submit(ctx context.Context, task models.Task) error {
	q.mu.Lock()
	if q.stopped {
		q.mu.Unlock()
		return fmt.Errorf("queue is stopped")
	}
	if q.isTaskEnqueued(task.ID) {
		q.mu.Unlock()
		return fmt.Errorf("task %s is already running", task.ID)
	}
	if q.isTaskInHeap(task.ID) {
		q.mu.Unlock()
		return fmt.Errorf("task %s is already pending", task.ID)
	}
	if q.maxPending > 0 && q.pq.Len()+q.pausedPQ.Len() >= q.maxPending {
		q.mu.Unlock()
		return ErrQueueFull
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
	q.mu.Unlock()

	if err := q.db.UpdateTaskStatus(context.Background(), task.ID, models.TaskStatusQueued, ""); err != nil {
		q.mu.Lock()
		q.removeFromHeap(task.ID)
		q.mu.Unlock()
		cancel()
		return fmt.Errorf("update task status to queued: %w", err)
	}
	q.emitEvent(task.ID, models.TaskStatusQueued, "")

	// Signal one worker that a task is available.
	q.cond.Signal()
	return nil
}

// SubmitBatch enqueues multiple tasks with a single lock acquisition and DB transaction.
func (q *Queue) SubmitBatch(ctx context.Context, tasks []models.Task) error {
	if len(tasks) == 0 {
		return nil
	}

	q.mu.Lock()
	if q.stopped {
		q.mu.Unlock()
		return fmt.Errorf("queue is stopped")
	}

	if q.maxPending > 0 && q.pq.Len()+q.pausedPQ.Len()+len(tasks) > q.maxPending {
		q.mu.Unlock()
		return ErrQueueFull
	}

	added := make([]models.Task, 0, len(tasks))
	for _, task := range tasks {
		if q.isTaskEnqueued(task.ID) || q.isTaskInHeap(task.ID) {
			continue
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

	if len(added) == 0 {
		return nil
	}

	taskIDs := make([]string, len(added))
	for i, t := range added {
		taskIDs[i] = t.ID
	}
	if err := q.db.BatchUpdateTaskStatus(ctx, taskIDs, models.TaskStatusQueued, ""); err != nil {
		q.mu.Lock()
		for _, t := range added {
			q.removeFromHeap(t.ID)
		}
		q.mu.Unlock()
		return fmt.Errorf("batch update task status: %w", err)
	}

	for _, t := range added {
		q.emitEvent(t.ID, models.TaskStatusQueued, "")
	}
	q.cond.Broadcast()
	return nil
}

// Cancel stops a running or pending task and marks it as cancelled.
func (q *Queue) Cancel(taskID string) error {
	q.mu.Lock()

	if cancel, ok := q.running[taskID]; ok {
		q.cancelled[taskID] = true
		cancel()
		delete(q.running, taskID)
		q.mu.Unlock()
	} else if q.removeFromHeap(taskID) {
		q.cancelled[taskID] = true
		q.mu.Unlock()
	} else if q.removeFromPausedHeap(taskID) {
		q.cancelled[taskID] = true
		q.mu.Unlock()
	} else {
		q.mu.Unlock()
	}

	if err := q.db.UpdateTaskStatus(context.Background(), taskID, models.TaskStatusCancelled, "cancelled by user"); err != nil {
		return fmt.Errorf("update task status to cancelled: %w", err)
	}
	q.emitEvent(taskID, models.TaskStatusCancelled, "cancelled by user")
	return nil
}

// PauseBatch pauses all pending tasks for the given batch. Running tasks
// continue to completion but new tasks from this batch won't be picked up.
func (q *Queue) PauseBatch(batchID string) {
	q.mu.Lock()
	q.paused[batchID] = true
	// Move items from main heap to paused heap for this batch.
	var remaining []*heapItem
	for q.pq.Len() > 0 {
		item := heap.Pop(&q.pq).(*heapItem)
		delete(q.heapSet, item.task.ID)
		if item.task.BatchID == batchID {
			heap.Push(&q.pausedPQ, item)
			q.pausedSet[item.task.ID] = struct{}{}
		} else {
			remaining = append(remaining, item)
		}
	}
	for _, item := range remaining {
		heap.Push(&q.pq, item)
		q.heapSet[item.task.ID] = struct{}{}
	}
	q.mu.Unlock()
}

// ResumeBatch resumes a paused batch. Paused tasks are moved back into the
// main priority queue and workers are signaled.
func (q *Queue) ResumeBatch(batchID string) {
	q.mu.Lock()
	delete(q.paused, batchID)
	// Move items back from paused heap to main heap.
	var remaining []*heapItem
	movedCount := 0
	for q.pausedPQ.Len() > 0 {
		item := heap.Pop(&q.pausedPQ).(*heapItem)
		delete(q.pausedSet, item.task.ID)
		if item.task.BatchID == batchID {
			heap.Push(&q.pq, item)
			q.heapSet[item.task.ID] = struct{}{}
			movedCount++
		} else {
			remaining = append(remaining, item)
		}
	}
	for _, item := range remaining {
		heap.Push(&q.pausedPQ, item)
		q.pausedSet[item.task.ID] = struct{}{}
	}
	q.mu.Unlock()

	// Wake workers for the resumed tasks.
	if movedCount > 0 {
		q.cond.Broadcast()
	}
}

// Stop cancels all running and pending tasks and prevents new submissions.
func (q *Queue) Stop() {
	q.stopOnce.Do(func() {
		q.mu.Lock()
		q.stopped = true

		// Cancel all running tasks.
		for id, cancel := range q.running {
			cancel()
			delete(q.running, id)
		}

		// Cancel all tasks in the main heap.
		for q.pq.Len() > 0 {
			item := heap.Pop(&q.pq).(*heapItem)
			delete(q.heapSet, item.task.ID)
			item.cancel()
		}

		// Cancel all tasks in the paused heap.
		for q.pausedPQ.Len() > 0 {
			item := heap.Pop(&q.pausedPQ).(*heapItem)
			delete(q.pausedSet, item.task.ID)
			item.cancel()
		}
		q.mu.Unlock()

		// Wake all workers so they can exit.
		q.cond.Broadcast()
		close(q.stopCh)

		// Wait for all workers to finish.
		q.workerWg.Wait()
	})
}

// RunningCount returns the number of currently executing tasks.
func (q *Queue) RunningCount() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.running)
}

// Metrics returns a snapshot of queue metrics.
// Queued = tasks waiting in the priority heap (main + paused).
// Running = tasks currently executing.
// Pending = Queued + Running (total in-flight).
func (q *Queue) Metrics() models.QueueMetrics {
	q.mu.Lock()
	defer q.mu.Unlock()
	metrics := q.metrics
	metrics.Running = len(q.running)
	metrics.Queued = q.pq.Len() + q.pausedPQ.Len()
	metrics.Pending = metrics.Queued + metrics.Running
	return metrics
}

// RecoverStaleTasks finds tasks stuck in "running" or "queued" status
// (from a previous crash), resets them to "pending", and re-submits them.
func (q *Queue) RecoverStaleTasks(ctx context.Context) error {
	stale, err := q.db.ListStaleTasks(ctx)
	if err != nil {
		return fmt.Errorf("list stale tasks: %w", err)
	}
	for _, task := range stale {
		if err := q.db.UpdateTaskStatus(ctx, task.ID, models.TaskStatusPending, "recovered after restart"); err != nil {
			return fmt.Errorf("reset stale task %s: %w", task.ID, err)
		}
		task.Status = models.TaskStatusPending
		if err := q.Submit(ctx, task); err != nil {
			return fmt.Errorf("re-submit stale task %s: %w", task.ID, err)
		}
	}
	return nil
}

// worker is the main loop for a fixed pool worker. It blocks on the
// condition variable until a task is available, then executes it.
func (q *Queue) worker(_ int) {
	defer q.workerWg.Done()

	for {
		q.mu.Lock()
		// Wait until there's work or the queue is stopped.
		for q.pq.Len() == 0 && !q.stopped {
			q.cond.Wait()
		}
		if q.stopped {
			q.mu.Unlock()
			return
		}

		item := heap.Pop(&q.pq).(*heapItem)
		delete(q.heapSet, item.task.ID)

		// Skip cancelled tasks.
		if q.cancelled[item.task.ID] {
			delete(q.cancelled, item.task.ID)
			item.cancel()
			q.mu.Unlock()
			continue
		}

		// Move paused-batch items to the paused heap.
		if item.task.BatchID != "" && q.paused[item.task.BatchID] {
			heap.Push(&q.pausedPQ, item)
			q.pausedSet[item.task.ID] = struct{}{}
			q.mu.Unlock()
			continue
		}

		q.running[item.task.ID] = item.cancel
		q.mu.Unlock()

		q.executeTask(item.ctx, item.task)
	}
}

type retryInfo struct {
	shouldRetry bool
	task        models.Task
	backoff     time.Duration
	parentCtx   context.Context
}

func (q *Queue) executeTask(ctx context.Context, task models.Task) {
	defer func() {
		q.mu.Lock()
		delete(q.running, task.ID)
		delete(q.cancelled, task.ID)
		q.mu.Unlock()
	}()

	q.mu.Lock()
	if q.stopped || q.cancelled[task.ID] {
		delete(q.cancelled, task.ID)
		q.mu.Unlock()
		return
	}
	q.mu.Unlock()

	taskTimeout := 5 * time.Minute
	if task.Timeout > 0 {
		taskTimeout = time.Duration(task.Timeout) * time.Second
	}
	taskCtx, cancel := context.WithTimeout(ctx, taskTimeout)
	defer cancel()

	// Update the cancel func so Cancel() uses the timeout-aware one.
	q.mu.Lock()
	if q.stopped || q.cancelled[task.ID] {
		delete(q.cancelled, task.ID)
		q.mu.Unlock()
		return
	}
	q.running[task.ID] = cancel
	q.mu.Unlock()

	var selectedProxyID string
	pm := q.getProxyManager()
	if task.Proxy.Server == "" && pm != nil {
		if p, err := pm.SelectProxy(task.Proxy.Geo); err == nil {
			selectedProxyID = p.ID
			task.Proxy = p.ToProxyConfig()
		}
	}

	if err := q.db.UpdateTaskStatus(taskCtx, task.ID, models.TaskStatusRunning, ""); err != nil {
		q.emitEvent(task.ID, models.TaskStatusFailed, fmt.Sprintf("update status: %v", err))
		return
	}
	q.emitEvent(task.ID, models.TaskStatusRunning, "")

	result, err := q.runner.RunTask(taskCtx, task)

	if result != nil && len(result.StepLogs) > 0 {
		if slErr := q.db.InsertStepLogs(ctx, task.ID, result.StepLogs); slErr != nil {
			q.emitEvent(task.ID, task.Status, fmt.Sprintf("persist step logs: %v", slErr))
		}
	}

	if result != nil && len(result.NetworkLogs) > 0 {
		if nlErr := q.db.InsertNetworkLogs(ctx, task.ID, result.NetworkLogs); nlErr != nil {
			q.emitEvent(task.ID, task.Status, fmt.Sprintf("persist network logs: %v", nlErr))
		}
	}

	if selectedProxyID != "" {
		if pm := q.getProxyManager(); pm != nil {
			if recordErr := pm.RecordUsage(selectedProxyID, err == nil); recordErr != nil {
				q.emitEvent(task.ID, task.Status, fmt.Sprintf("proxy usage recording failed: %v", recordErr))
			}
		}
	}

	var retry retryInfo
	if err != nil {
		retry = q.handleFailure(ctx, task, err)
	} else {
		q.handleSuccess(task, result)
	}

	if retry.shouldRetry {
		go q.scheduleRetry(retry)
	}
}

func (q *Queue) handleFailure(parentCtx context.Context, task models.Task, execErr error) retryInfo {
	if task.RetryCount < task.MaxRetries {
		if err := q.db.IncrementRetry(context.Background(), task.ID); err != nil {
			q.emitEvent(task.ID, models.TaskStatusFailed, fmt.Sprintf("increment retry: %v", err))
			return retryInfo{}
		}
		q.emitEvent(task.ID, models.TaskStatusRetrying, execErr.Error())

		backoffSec := math.Pow(2, float64(task.RetryCount))
		if backoffSec > 60 {
			backoffSec = 60
		}
		backoff := time.Duration(backoffSec) * time.Second

		retryTask := task
		retryTask.RetryCount++
		retryTask.Status = models.TaskStatusPending
		retryTask.Steps = make([]models.TaskStep, len(task.Steps))
		copy(retryTask.Steps, task.Steps)

		return retryInfo{
			shouldRetry: true,
			task:        retryTask,
			backoff:     backoff,
			parentCtx:   parentCtx,
		}
	}

	if err := q.db.UpdateTaskStatus(context.Background(), task.ID, models.TaskStatusFailed, execErr.Error()); err != nil {
		q.emitEvent(task.ID, models.TaskStatusFailed, fmt.Sprintf("update status: %v", err))
		return retryInfo{}
	}
	q.mu.Lock()
	q.metrics.TotalFailed++
	q.mu.Unlock()
	q.emitEvent(task.ID, models.TaskStatusFailed, execErr.Error())
	return retryInfo{}
}

func (q *Queue) scheduleRetry(ri retryInfo) {
	timer := time.NewTimer(ri.backoff)
	select {
	case <-timer.C:
	case <-q.stopCh:
		timer.Stop()
		if err := q.db.UpdateTaskStatus(context.Background(), ri.task.ID, models.TaskStatusCancelled, "cancelled during retry backoff (queue stopped)"); err != nil {
			q.emitEvent(ri.task.ID, models.TaskStatusFailed, fmt.Sprintf("update status: %v", err))
		}
		return
	case <-ri.parentCtx.Done():
		timer.Stop()
		if err := q.db.UpdateTaskStatus(context.Background(), ri.task.ID, models.TaskStatusCancelled, "cancelled during retry backoff"); err != nil {
			q.emitEvent(ri.task.ID, models.TaskStatusFailed, fmt.Sprintf("update status: %v", err))
		}
		return
	}

	// Re-submit via the heap instead of spawning another goroutine.
	if err := q.Submit(ri.parentCtx, ri.task); err != nil {
		if err2 := q.db.UpdateTaskStatus(context.Background(), ri.task.ID, models.TaskStatusFailed, fmt.Sprintf("retry re-submit: %v", err)); err2 != nil {
			q.emitEvent(ri.task.ID, models.TaskStatusFailed, fmt.Sprintf("update status: %v", err2))
		}
		q.emitEvent(ri.task.ID, models.TaskStatusFailed, fmt.Sprintf("retry re-submit: %v", err))
	}
}

func (q *Queue) handleSuccess(task models.Task, result *models.TaskResult) {
	if err := q.db.UpdateTaskResult(context.Background(), task.ID, *result); err != nil {
		q.emitEvent(task.ID, models.TaskStatusFailed, fmt.Sprintf("save result: %v", err))
		return
	}
	if err := q.db.UpdateTaskStatus(context.Background(), task.ID, models.TaskStatusCompleted, ""); err != nil {
		q.emitEvent(task.ID, models.TaskStatusFailed, fmt.Sprintf("update status: %v", err))
		return
	}
	q.mu.Lock()
	q.metrics.TotalCompleted++
	q.mu.Unlock()
	q.emitEvent(task.ID, models.TaskStatusCompleted, "")
}

func (q *Queue) emitEvent(taskID string, status models.TaskStatus, errMsg string) {
	if q.onEvent != nil {
		q.onEvent(models.TaskEvent{
			TaskID: taskID,
			Status: status,
			Error:  errMsg,
		})
	}
}

// isTaskEnqueued checks if a task is currently running. Must be called with mu held.
func (q *Queue) isTaskEnqueued(taskID string) bool {
	_, ok := q.running[taskID]
	return ok
}

// isTaskInHeap checks if a task is in the main or paused heap. Must be called with mu held.
func (q *Queue) isTaskInHeap(taskID string) bool {
	_, inMain := q.heapSet[taskID]
	_, inPaused := q.pausedSet[taskID]
	return inMain || inPaused
}

// removeFromHeap removes a task from the main heap. Returns true if found.
// Must be called with mu held.
func (q *Queue) removeFromHeap(taskID string) bool {
	for i, item := range q.pq {
		if item.task.ID == taskID {
			item.cancel()
			heap.Remove(&q.pq, i)
			delete(q.heapSet, taskID)
			return true
		}
	}
	return false
}

// removeFromPausedHeap removes a task from the paused heap. Returns true if found.
// Must be called with mu held.
func (q *Queue) removeFromPausedHeap(taskID string) bool {
	for i, item := range q.pausedPQ {
		if item.task.ID == taskID {
			item.cancel()
			heap.Remove(&q.pausedPQ, i)
			delete(q.pausedSet, taskID)
			return true
		}
	}
	return false
}
