<script lang="ts">
  import { selectedTask } from '../lib/store';

  function formatDuration(ns: number): string {
    if (!ns) return '-';
    const ms = ns / 1000000;
    if (ms < 1000) return `${ms.toFixed(0)}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  }
</script>

<div class="detail-panel">
  {#if $selectedTask}
    <div class="detail-header">
      <h3>{$selectedTask.name}</h3>
      <span class="badge badge-{$selectedTask.status}">{$selectedTask.status}</span>
    </div>

    <div class="detail-section">
      <h4>Info</h4>
      <div class="detail-grid">
        <div class="detail-item">
          <span class="label">ID</span>
          <span class="value font-mono">{$selectedTask.id}</span>
        </div>
        <div class="detail-item">
          <span class="label">URL</span>
          <span class="value">{$selectedTask.url}</span>
        </div>
        <div class="detail-item">
          <span class="label">Priority</span>
          <span class="value">{$selectedTask.priority}</span>
        </div>
        <div class="detail-item">
          <span class="label">Retries</span>
          <span class="value">{$selectedTask.retryCount} / {$selectedTask.maxRetries}</span>
        </div>
        {#if $selectedTask.error}
          <div class="detail-item error">
            <span class="label">Error</span>
            <span class="value">{$selectedTask.error}</span>
          </div>
        {/if}
      </div>
    </div>

    {#if $selectedTask.proxy?.server}
      <div class="detail-section">
        <h4>Proxy</h4>
        <div class="detail-grid">
          <div class="detail-item">
            <span class="label">Server</span>
            <span class="value font-mono">{$selectedTask.proxy.server}</span>
          </div>
          {#if $selectedTask.proxy.geo}
            <div class="detail-item">
              <span class="label">Geo</span>
              <span class="value">{$selectedTask.proxy.geo}</span>
            </div>
          {/if}
        </div>
      </div>
    {/if}

    {#if $selectedTask.steps?.length}
      <div class="detail-section">
        <h4>Steps ({$selectedTask.steps.length})</h4>
        <div class="steps-list">
          {#each $selectedTask.steps as step, i}
            <div class="step">
              <span class="step-num">{i + 1}</span>
              <span class="step-action">{step.action}</span>
              {#if step.selector}
                <span class="step-selector font-mono">{step.selector}</span>
              {/if}
              {#if step.value}
                <span class="step-value">= {step.value}</span>
              {/if}
            </div>
          {/each}
        </div>
      </div>
    {/if}

    {#if $selectedTask.result}
      <div class="detail-section">
        <h4>Result</h4>
        <div class="detail-grid">
          <div class="detail-item">
            <span class="label">Duration</span>
            <span class="value">{formatDuration($selectedTask.result.duration)}</span>
          </div>
          <div class="detail-item">
            <span class="label">Success</span>
            <span class="value">{$selectedTask.result.success ? 'Yes' : 'No'}</span>
          </div>
        </div>

        {#if $selectedTask.result.extractedData && Object.keys($selectedTask.result.extractedData).length}
          <h4 class="mt-2">Extracted Data</h4>
          <div class="extracted-data font-mono">
            {#each Object.entries($selectedTask.result.extractedData) as [key, val]}
              <div><strong>{key}:</strong> {val}</div>
            {/each}
          </div>
        {/if}

        {#if $selectedTask.result.logs?.length}
          <h4 class="mt-2">Logs</h4>
          <div class="log-viewer font-mono">
            {#each $selectedTask.result.logs as log}
              <div class="log-entry log-{log.level}">
                <span class="log-level">[{log.level}]</span>
                {log.message}
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {:else}
    <div class="empty-detail">
      <p>Select a task to view details</p>
    </div>
  {/if}
</div>

<style>
  .detail-panel {
    width: 380px;
    background: var(--bg-secondary);
    border-left: 1px solid var(--border);
    overflow-y: auto;
    padding: 16px;
    flex-shrink: 0;
  }
  .detail-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }
  .detail-header h3 {
    font-size: 16px;
    font-weight: 600;
    margin: 0;
  }
  .detail-section {
    margin-bottom: 16px;
  }
  .detail-section h4 {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 8px;
  }
  .detail-grid {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .detail-item {
    display: flex;
    gap: 8px;
  }
  .detail-item .label {
    color: var(--text-muted);
    min-width: 70px;
    font-size: 12px;
  }
  .detail-item .value {
    font-size: 12px;
    word-break: break-all;
  }
  .detail-item.error .value {
    color: var(--danger);
  }
  .steps-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .step {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    background: var(--bg-tertiary);
    border-radius: 4px;
    font-size: 12px;
  }
  .step-num {
    color: var(--text-muted);
    min-width: 16px;
  }
  .step-action {
    color: var(--accent);
    font-weight: 600;
  }
  .step-selector {
    color: var(--warning);
  }
  .step-value {
    color: var(--text-secondary);
  }
  .extracted-data {
    padding: 8px;
    background: var(--bg-tertiary);
    border-radius: 4px;
    font-size: 11px;
  }
  .log-viewer {
    max-height: 200px;
    overflow-y: auto;
    padding: 8px;
    background: var(--bg-primary);
    border-radius: 4px;
  }
  .log-entry {
    font-size: 11px;
    padding: 2px 0;
  }
  .log-level {
    font-weight: 600;
  }
  .log-info .log-level { color: var(--accent); }
  .log-warn .log-level { color: var(--warning); }
  .log-error .log-level { color: var(--danger); }
  .empty-detail {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-muted);
  }
</style>
