package database

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"web-automation/internal/models"
)

func setupTestDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func makeTask(id, name string) models.Task {
	return models.Task{
		ID:   id,
		Name: name,
		URL:  "https://example.com",
		Steps: []models.TaskStep{
			{Action: models.ActionNavigate, Value: "https://example.com"},
			{Action: models.ActionClick, Selector: "#btn"},
		},
		Proxy: models.ProxyConfig{
			Server:   "proxy.example.com:8080",
			Username: "user",
			Password: "pass",
			Geo:      "US",
		},
		Priority:   models.PriorityNormal,
		Status:     models.TaskStatusPending,
		MaxRetries: 3,
		Tags:       []string{"test", "demo"},
		CreatedAt:  time.Now().Truncate(time.Second),
	}
}

func makeProxy(id, server, geo string) models.Proxy {
	return models.Proxy{
		ID:        id,
		Server:    server,
		Protocol:  models.ProxyHTTP,
		Username:  "user",
		Password:  "pass",
		Geo:       geo,
		Status:    models.ProxyStatusUnknown,
		CreatedAt: time.Now().Truncate(time.Second),
	}
}

// --- Task CRUD Tests ---

func TestCreateAndGetTask(t *testing.T) {
	db := setupTestDB(t)
	task := makeTask("task-1", "Test Task")

	if err := db.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	got, err := db.GetTask("task-1")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}

	if got.ID != task.ID {
		t.Errorf("ID: got %q, want %q", got.ID, task.ID)
	}
	if got.Name != task.Name {
		t.Errorf("Name: got %q, want %q", got.Name, task.Name)
	}
	if got.URL != task.URL {
		t.Errorf("URL: got %q, want %q", got.URL, task.URL)
	}
	if len(got.Steps) != len(task.Steps) {
		t.Errorf("Steps length: got %d, want %d", len(got.Steps), len(task.Steps))
	}
	if got.Proxy.Server != task.Proxy.Server {
		t.Errorf("Proxy.Server: got %q, want %q", got.Proxy.Server, task.Proxy.Server)
	}
	if got.Priority != task.Priority {
		t.Errorf("Priority: got %d, want %d", got.Priority, task.Priority)
	}
	if got.Status != task.Status {
		t.Errorf("Status: got %q, want %q", got.Status, task.Status)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "test" {
		t.Errorf("Tags: got %v, want %v", got.Tags, task.Tags)
	}
}

func TestGetTaskNotFound(t *testing.T) {
	db := setupTestDB(t)
	_, err := db.GetTask("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent task, got nil")
	}
}

func TestCreateTaskDuplicateID(t *testing.T) {
	db := setupTestDB(t)
	task := makeTask("dup-1", "First")
	if err := db.CreateTask(task); err != nil {
		t.Fatalf("first CreateTask: %v", err)
	}
	task2 := makeTask("dup-1", "Second")
	err := db.CreateTask(task2)
	if err == nil {
		t.Fatal("expected error for duplicate ID, got nil")
	}
}

func TestListTasks(t *testing.T) {
	db := setupTestDB(t)

	for i := 0; i < 5; i++ {
		task := makeTask(
			"list-"+string(rune('a'+i)),
			"Task "+string(rune('A'+i)),
		)
		task.Priority = models.TaskPriority(i + 1)
		if err := db.CreateTask(task); err != nil {
			t.Fatalf("CreateTask %d: %v", i, err)
		}
	}

	tasks, err := db.ListTasks()
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 5 {
		t.Fatalf("ListTasks count: got %d, want 5", len(tasks))
	}

	// Should be ordered by priority DESC
	for i := 1; i < len(tasks); i++ {
		if tasks[i].Priority > tasks[i-1].Priority {
			t.Errorf("tasks not sorted by priority DESC: %d > %d at index %d",
				tasks[i].Priority, tasks[i-1].Priority, i)
		}
	}
}

func TestListTasksByStatus(t *testing.T) {
	db := setupTestDB(t)

	task1 := makeTask("status-1", "Pending Task")
	task1.Status = models.TaskStatusPending
	task2 := makeTask("status-2", "Running Task")
	task2.Status = models.TaskStatusRunning
	task3 := makeTask("status-3", "Also Pending")
	task3.Status = models.TaskStatusPending

	for _, task := range []models.Task{task1, task2, task3} {
		if err := db.CreateTask(task); err != nil {
			t.Fatalf("CreateTask %s: %v", task.ID, err)
		}
	}

	pending, err := db.ListTasksByStatus(models.TaskStatusPending)
	if err != nil {
		t.Fatalf("ListTasksByStatus(pending): %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("pending count: got %d, want 2", len(pending))
	}

	running, err := db.ListTasksByStatus(models.TaskStatusRunning)
	if err != nil {
		t.Fatalf("ListTasksByStatus(running): %v", err)
	}
	if len(running) != 1 {
		t.Errorf("running count: got %d, want 1", len(running))
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	db := setupTestDB(t)
	task := makeTask("upd-status-1", "Update Status Test")
	if err := db.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	tests := []struct {
		name   string
		status models.TaskStatus
		errMsg string
	}{
		{"to running", models.TaskStatusRunning, ""},
		{"to failed", models.TaskStatusFailed, "something broke"},
		{"to queued", models.TaskStatusQueued, ""},
		{"to completed", models.TaskStatusCompleted, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := db.UpdateTaskStatus(task.ID, tc.status, tc.errMsg); err != nil {
				t.Fatalf("UpdateTaskStatus: %v", err)
			}
			got, err := db.GetTask(task.ID)
			if err != nil {
				t.Fatalf("GetTask: %v", err)
			}
			if got.Status != tc.status {
				t.Errorf("status: got %q, want %q", got.Status, tc.status)
			}
		})
	}
}

func TestUpdateTaskStatusSetsTimestamps(t *testing.T) {
	db := setupTestDB(t)
	task := makeTask("ts-1", "Timestamp Test")
	if err := db.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	// Running should set started_at
	if err := db.UpdateTaskStatus(task.ID, models.TaskStatusRunning, ""); err != nil {
		t.Fatalf("UpdateTaskStatus(running): %v", err)
	}
	got, _ := db.GetTask(task.ID)
	if got.StartedAt == nil {
		t.Error("StartedAt should be set after running")
	}

	// Completed should set completed_at
	if err := db.UpdateTaskStatus(task.ID, models.TaskStatusCompleted, ""); err != nil {
		t.Fatalf("UpdateTaskStatus(completed): %v", err)
	}
	got, _ = db.GetTask(task.ID)
	if got.CompletedAt == nil {
		t.Error("CompletedAt should be set after completed")
	}
}

func TestUpdateTaskResult(t *testing.T) {
	db := setupTestDB(t)
	task := makeTask("result-1", "Result Test")
	if err := db.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	result := models.TaskResult{
		TaskID:  task.ID,
		Success: true,
		ExtractedData: map[string]string{
			"title": "Example",
		},
		Screenshots: []string{"/tmp/shot1.png"},
		Duration:    5 * time.Second,
	}

	if err := db.UpdateTaskResult(task.ID, result); err != nil {
		t.Fatalf("UpdateTaskResult: %v", err)
	}

	got, err := db.GetTask(task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.Result == nil {
		t.Fatal("Result is nil after update")
	}
	if !got.Result.Success {
		t.Error("Result.Success should be true")
	}
	if got.Result.ExtractedData["title"] != "Example" {
		t.Errorf("ExtractedData[title]: got %q, want %q", got.Result.ExtractedData["title"], "Example")
	}
}

func TestIncrementRetry(t *testing.T) {
	db := setupTestDB(t)
	task := makeTask("retry-1", "Retry Test")
	if err := db.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	for i := 0; i < 3; i++ {
		if err := db.IncrementRetry(task.ID); err != nil {
			t.Fatalf("IncrementRetry %d: %v", i, err)
		}
	}

	got, err := db.GetTask(task.ID)
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if got.RetryCount != 3 {
		t.Errorf("RetryCount: got %d, want 3", got.RetryCount)
	}
	if got.Status != models.TaskStatusRetrying {
		t.Errorf("Status: got %q, want %q", got.Status, models.TaskStatusRetrying)
	}
}

func TestDeleteTask(t *testing.T) {
	db := setupTestDB(t)
	task := makeTask("del-1", "Delete Test")
	if err := db.CreateTask(task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	if err := db.DeleteTask(task.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	_, err := db.GetTask(task.ID)
	if err == nil {
		t.Error("GetTask should return error after deletion")
	}
}

func TestGetTaskStats(t *testing.T) {
	db := setupTestDB(t)

	tasks := []models.Task{
		makeTask("stats-1", "S1"),
		makeTask("stats-2", "S2"),
		makeTask("stats-3", "S3"),
	}
	tasks[0].Status = models.TaskStatusPending
	tasks[1].Status = models.TaskStatusPending
	tasks[2].Status = models.TaskStatusCompleted

	for _, task := range tasks {
		if err := db.CreateTask(task); err != nil {
			t.Fatalf("CreateTask: %v", err)
		}
	}

	stats, err := db.GetTaskStats()
	if err != nil {
		t.Fatalf("GetTaskStats: %v", err)
	}

	if stats["pending"] != 2 {
		t.Errorf("pending count: got %d, want 2", stats["pending"])
	}
	if stats["completed"] != 1 {
		t.Errorf("completed count: got %d, want 1", stats["completed"])
	}
}

// --- Proxy CRUD Tests ---

func TestCreateAndListProxies(t *testing.T) {
	db := setupTestDB(t)

	p1 := makeProxy("proxy-1", "proxy1.example.com:8080", "US")
	p2 := makeProxy("proxy-2", "proxy2.example.com:8080", "UK")

	if err := db.CreateProxy(p1); err != nil {
		t.Fatalf("CreateProxy 1: %v", err)
	}
	if err := db.CreateProxy(p2); err != nil {
		t.Fatalf("CreateProxy 2: %v", err)
	}

	proxies, err := db.ListProxies()
	if err != nil {
		t.Fatalf("ListProxies: %v", err)
	}
	if len(proxies) != 2 {
		t.Errorf("proxy count: got %d, want 2", len(proxies))
	}
}

func TestListHealthyProxies(t *testing.T) {
	db := setupTestDB(t)

	p1 := makeProxy("hp-1", "h1.example.com:8080", "US")
	p2 := makeProxy("hp-2", "h2.example.com:8080", "UK")

	if err := db.CreateProxy(p1); err != nil {
		t.Fatalf("CreateProxy: %v", err)
	}
	if err := db.CreateProxy(p2); err != nil {
		t.Fatalf("CreateProxy: %v", err)
	}

	// Mark one as healthy
	if err := db.UpdateProxyHealth("hp-1", models.ProxyStatusHealthy, 50); err != nil {
		t.Fatalf("UpdateProxyHealth: %v", err)
	}

	healthy, err := db.ListHealthyProxies()
	if err != nil {
		t.Fatalf("ListHealthyProxies: %v", err)
	}
	if len(healthy) != 1 {
		t.Errorf("healthy count: got %d, want 1", len(healthy))
	}
	if healthy[0].ID != "hp-1" {
		t.Errorf("healthy proxy ID: got %q, want %q", healthy[0].ID, "hp-1")
	}
}

func TestUpdateProxyHealth(t *testing.T) {
	db := setupTestDB(t)
	p := makeProxy("health-1", "health.example.com:8080", "US")
	if err := db.CreateProxy(p); err != nil {
		t.Fatalf("CreateProxy: %v", err)
	}

	if err := db.UpdateProxyHealth("health-1", models.ProxyStatusHealthy, 100); err != nil {
		t.Fatalf("UpdateProxyHealth: %v", err)
	}

	proxies, err := db.ListProxies()
	if err != nil {
		t.Fatalf("ListProxies: %v", err)
	}

	var found bool
	for _, px := range proxies {
		if px.ID == "health-1" {
			found = true
			if px.Status != models.ProxyStatusHealthy {
				t.Errorf("Status: got %q, want %q", px.Status, models.ProxyStatusHealthy)
			}
			if px.Latency != 100 {
				t.Errorf("Latency: got %d, want 100", px.Latency)
			}
			if px.LastChecked == nil {
				t.Error("LastChecked should be set")
			}
		}
	}
	if !found {
		t.Error("proxy not found in list")
	}
}

func TestIncrementProxyUsage(t *testing.T) {
	db := setupTestDB(t)
	p := makeProxy("usage-1", "usage.example.com:8080", "US")
	if err := db.CreateProxy(p); err != nil {
		t.Fatalf("CreateProxy: %v", err)
	}

	// Record 3 successes and 1 failure
	for i := 0; i < 3; i++ {
		if err := db.IncrementProxyUsage("usage-1", true); err != nil {
			t.Fatalf("IncrementProxyUsage(success) %d: %v", i, err)
		}
	}
	if err := db.IncrementProxyUsage("usage-1", false); err != nil {
		t.Fatalf("IncrementProxyUsage(failure): %v", err)
	}

	proxies, err := db.ListProxies()
	if err != nil {
		t.Fatalf("ListProxies: %v", err)
	}
	for _, px := range proxies {
		if px.ID == "usage-1" {
			if px.TotalUsed != 4 {
				t.Errorf("TotalUsed: got %d, want 4", px.TotalUsed)
			}
			// Success rate should be 0.75 (3/4)
			if px.SuccessRate < 0.74 || px.SuccessRate > 0.76 {
				t.Errorf("SuccessRate: got %f, want ~0.75", px.SuccessRate)
			}
		}
	}
}

func TestDeleteProxy(t *testing.T) {
	db := setupTestDB(t)
	p := makeProxy("del-p-1", "del.example.com:8080", "US")
	if err := db.CreateProxy(p); err != nil {
		t.Fatalf("CreateProxy: %v", err)
	}

	if err := db.DeleteProxy("del-p-1"); err != nil {
		t.Fatalf("DeleteProxy: %v", err)
	}

	proxies, err := db.ListProxies()
	if err != nil {
		t.Fatalf("ListProxies: %v", err)
	}
	if len(proxies) != 0 {
		t.Errorf("proxy count after delete: got %d, want 0", len(proxies))
	}
}

func TestNewDatabaseInvalidPath(t *testing.T) {
	_, err := New("/nonexistent/path/to/db.db")
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

func TestCreateTaskWithNilStepsAndTags(t *testing.T) {
	db := setupTestDB(t)
	task := models.Task{
		ID:        "nil-fields",
		Name:      "Nil Fields",
		URL:       "https://example.com",
		Status:    models.TaskStatusPending,
		CreatedAt: time.Now(),
		// Steps and Tags are nil
	}

	if err := db.CreateTask(task); err != nil {
		t.Fatalf("CreateTask with nil slices: %v", err)
	}

	got, err := db.GetTask("nil-fields")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	// nil steps serializes as "null", which Unmarshal treats as nil slice
	if got.Steps != nil && len(got.Steps) != 0 {
		t.Errorf("Steps: got %v, want nil or empty", got.Steps)
	}
}

func TestMigrationIdempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create database twice - migration should be idempotent
	db1, err := New(dbPath)
	if err != nil {
		t.Fatalf("first New: %v", err)
	}
	db1.Close()

	db2, err := New(dbPath)
	if err != nil {
		t.Fatalf("second New: %v", err)
	}
	db2.Close()
}

func TestNewDatabasePermissions(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}
	dir := t.TempDir()
	// Make directory read-only
	if err := os.Chmod(dir, 0o444); err != nil {
		t.Skip("cannot change permissions")
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	dbPath := filepath.Join(dir, "test.db")
	_, err := New(dbPath)
	if err == nil {
		t.Error("expected error for read-only directory")
	}
}
