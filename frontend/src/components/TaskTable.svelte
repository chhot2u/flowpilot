<script lang="ts">
  import { filteredTasks, selectedTaskId } from '../lib/store';
  import { StartTask, CancelTask, DeleteTask } from '../../wailsjs/go/main/App';
  import type { Task } from '../lib/types';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  function selectTask(task: Task) {
    selectedTaskId.set(task.id);
  }

  async function startTask(id: string) {
    try {
      await StartTask(id);
    } catch (err) {
      console.error('Failed to start task:', err);
    }
    dispatch('refresh');
  }

  async function cancelTask(id: string) {
    try {
      await CancelTask(id);
    } catch (err) {
      console.error('Failed to cancel task:', err);
    }
    dispatch('refresh');
  }

  async function deleteTask(id: string) {
    if (!confirm('Delete this task?')) return;
    try {
      await DeleteTask(id);
    } catch (err) {
      console.error('Failed to delete task:', err);
    }
    dispatch('refresh');
  }

  function formatDate(dateStr: string): string {
    if (!dateStr) return '-';
    const d = new Date(dateStr);
    return d.toLocaleTimeString();
  }

  function shortId(id: string): string {
    return id.substring(0, 8);
  }
</script>

<div class="table-container">
  <table>
    <thead>
      <tr>
        <th>ID</th>
        <th>Name</th>
        <th>URL</th>
        <th>Status</th>
        <th>Priority</th>
        <th>Retries</th>
        <th>Created</th>
        <th>Actions</th>
      </tr>
    </thead>
    <tbody>
      {#each $filteredTasks as task (task.id)}
        <tr
          class:selected={$selectedTaskId === task.id}
          on:click={() => selectTask(task)}
        >
          <td class="font-mono">{shortId(task.id)}</td>
          <td class="truncate" style="max-width: 200px">{task.name}</td>
          <td class="truncate text-muted" style="max-width: 250px">{task.url}</td>
          <td>
            <span class="badge badge-{task.status}">{task.status}</span>
          </td>
          <td>{task.priority}</td>
          <td>{task.retryCount}/{task.maxRetries}</td>
          <td class="text-sm text-muted">{formatDate(task.createdAt)}</td>
          <td>
            <div class="actions">
              {#if task.status === 'pending' || task.status === 'failed'}
                <button class="btn-primary btn-sm" on:click|stopPropagation={() => startTask(task.id)}>
                  Start
                </button>
              {/if}
              {#if task.status === 'running' || task.status === 'queued'}
                <button class="btn-danger btn-sm" on:click|stopPropagation={() => cancelTask(task.id)}>
                  Cancel
                </button>
              {/if}
              {#if task.status !== 'running' && task.status !== 'queued'}
                <button class="btn-secondary btn-sm" on:click|stopPropagation={() => deleteTask(task.id)}>
                  Del
                </button>
              {/if}
            </div>
          </td>
        </tr>
      {:else}
        <tr>
          <td colspan="8" style="text-align: center; padding: 40px; color: var(--text-muted);">
            No tasks found. Click "+ New Task" to get started.
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>

<style>
  .table-container {
    flex: 1;
    overflow-y: auto;
  }
  tr.selected {
    background: rgba(59, 130, 246, 0.1) !important;
    border-left: 3px solid var(--accent);
  }
  .actions {
    display: flex;
    gap: 4px;
  }
  tbody tr {
    cursor: pointer;
  }
</style>
