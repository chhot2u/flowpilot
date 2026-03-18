package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"flowpilot/internal/models"
)

const flowExportVersion = "1.0"

func (a *App) ExportTask(taskID string) (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	task, err := a.db.GetTask(a.ctx, taskID)
	if err != nil {
		return "", fmt.Errorf("get task: %w", err)
	}
	if task == nil {
		return "", fmt.Errorf("task not found: %s", taskID)
	}

	export := models.TaskExport{
		Version:    flowExportVersion,
		ExportedAt: time.Now(),
		Name:       task.Name,
		Task:       *task,
	}

	exportPath := filepath.Join(a.dataDir, fmt.Sprintf("task_%s_%d.json", sanitizeFilename(task.Name), time.Now().Unix()))
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal task to JSON: %w", err)
	}

	if err := os.WriteFile(exportPath, data, 0o600); err != nil {
		return "", fmt.Errorf("write export file: %w", err)
	}
	return exportPath, nil
}

func (a *App) ImportTask(exportPath string) (*models.Task, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(exportPath)
	if err != nil {
		return nil, fmt.Errorf("read export file: %w", err)
	}

	var export models.TaskExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("unmarshal task export: %w", err)
	}

	if export.Version == "" {
		return nil, fmt.Errorf("invalid export file: missing version")
	}

	task := export.Task
	task.ID = ""
	task.Status = models.TaskStatusPending
	task.CreatedAt = time.Now()
	task.StartedAt = nil
	task.CompletedAt = nil
	task.Result = nil
	task.Error = ""

	if err := a.db.CreateTask(a.ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return &task, nil
}

func (a *App) ExportFlow(flowID string) (string, error) {
	if err := a.ready(); err != nil {
		return "", err
	}
	flow, err := a.db.GetRecordedFlow(a.ctx, flowID)
	if err != nil {
		return "", fmt.Errorf("get flow: %w", err)
	}
	if flow == nil {
		return "", fmt.Errorf("flow not found: %s", flowID)
	}

	export := models.FlowExport{
		Version:    flowExportVersion,
		ExportedAt: time.Now(),
		FlowName:   flow.Name,
		Tasks:      nil,
	}

	exportPath := filepath.Join(a.dataDir, fmt.Sprintf("flow_%s_%d.json", sanitizeFilename(flow.Name), time.Now().Unix()))
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal flow to JSON: %w", err)
	}

	if err := os.WriteFile(exportPath, data, 0o600); err != nil {
		return "", fmt.Errorf("write export file: %w", err)
	}
	return exportPath, nil
}

func (a *App) ImportFlow(exportPath string) ([]models.Task, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(exportPath)
	if err != nil {
		return nil, fmt.Errorf("read export file: %w", err)
	}

	var export models.FlowExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("unmarshal flow export: %w", err)
	}

	if export.Version == "" {
		return nil, fmt.Errorf("invalid export file: missing version")
	}

	now := time.Now()
	for i := range export.Tasks {
		export.Tasks[i].ID = ""
		export.Tasks[i].Status = models.TaskStatusPending
		export.Tasks[i].CreatedAt = now
		export.Tasks[i].StartedAt = nil
		export.Tasks[i].CompletedAt = nil
		export.Tasks[i].Result = nil
		export.Tasks[i].Error = ""
	}

	created := make([]models.Task, 0, len(export.Tasks))
	for i := range export.Tasks {
		task := export.Tasks[i]
		if err := a.db.CreateTask(a.ctx, task); err != nil {
			return nil, fmt.Errorf("create task %d: %w", i, err)
		}
		created = append(created, task)
	}
	return created, nil
}

func sanitizeFilename(name string) string {
	if name == "" {
		return "unnamed"
	}
	safe := make([]rune, 0, len(name))
	for _, r := range name {
		if r == '/' || r == '\\' || r == '.' || r == ':' || r == '\x00' {
			safe = append(safe, '_')
		} else {
			safe = append(safe, r)
		}
	}
	return string(safe)
}
