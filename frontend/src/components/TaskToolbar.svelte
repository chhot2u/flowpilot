<script lang="ts">
  import { statusFilter } from '../lib/store';
  import { StartAllPending, ExportResultsJSON, ExportResultsCSV } from '../../wailsjs/go/main/App';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let exporting = false;

  async function startAll() {
    try {
      await StartAllPending();
    } catch (err) {
      console.error('Failed to start tasks:', err);
    }
  }

  async function exportJSON() {
    exporting = true;
    try {
      const path = await ExportResultsJSON();
      alert(`Exported to: ${path}`);
    } catch (err) {
      console.error('Export failed:', err);
    }
    exporting = false;
  }

  async function exportCSV() {
    exporting = true;
    try {
      const path = await ExportResultsCSV();
      alert(`Exported to: ${path}`);
    } catch (err) {
      console.error('Export failed:', err);
    }
    exporting = false;
  }
</script>

<div class="toolbar">
  <div class="toolbar-left">
    <button class="btn-primary" on:click={() => dispatch('create')}>
      + New Task
    </button>
    <button class="btn-success" on:click={startAll}>
      Start All Pending
    </button>
  </div>

  <div class="toolbar-center">
    <select bind:value={$statusFilter}>
      <option value="all">All Status</option>
      <option value="pending">Pending</option>
      <option value="queued">Queued</option>
      <option value="running">Running</option>
      <option value="completed">Completed</option>
      <option value="failed">Failed</option>
      <option value="cancelled">Cancelled</option>
    </select>
  </div>

  <div class="toolbar-right">
    <button class="btn-secondary btn-sm" on:click={exportJSON} disabled={exporting}>
      Export JSON
    </button>
    <button class="btn-secondary btn-sm" on:click={exportCSV} disabled={exporting}>
      Export CSV
    </button>
  </div>
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 20px;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }
  .toolbar-left, .toolbar-right {
    display: flex;
    gap: 8px;
  }
  select {
    min-width: 150px;
  }
</style>
