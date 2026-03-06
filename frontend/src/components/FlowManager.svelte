<script lang="ts">
  import { ListRecordedFlows, DeleteRecordedFlow } from '../../wailsjs/go/main/App';
  import { recordedFlows } from '../lib/store';
  import { createEventDispatcher, onMount } from 'svelte';

  const dispatch = createEventDispatcher();

  let loading = false;
  let errorMessage = '';

  async function refresh() {
    loading = true;
    try {
      errorMessage = '';
      const flows = await ListRecordedFlows();
      recordedFlows.set(flows || []);
    } catch (err: any) {
      errorMessage = err?.message || String(err);
    } finally {
      loading = false;
    }
  }

  async function removeFlow(id: string) {
    try {
      await DeleteRecordedFlow(id);
      await refresh();
    } catch (err: any) {
      errorMessage = err?.message || String(err);
    }
  }

  onMount(() => {
    refresh();
  });
</script>

<div class="panel">
  <div class="panel-header">
    <h2>Recorded Flows</h2>
    <button class="btn-secondary btn-sm" on:click={refresh} disabled={loading}>Refresh</button>
  </div>

  {#if errorMessage}
    <div class="error-banner">{errorMessage}</div>
  {/if}

  <div class="panel-body">
    {#if $recordedFlows.length === 0}
      <div class="empty">No recorded flows yet.</div>
    {:else}
      {#each $recordedFlows as flow}
        <div class="flow-row">
          <div>
            <strong>{flow.name}</strong>
            <div class="muted">{flow.originUrl}</div>
          </div>
          <div class="actions">
            <button class="btn-primary btn-sm" on:click={() => dispatch('use', flow)}>Use</button>
            <button class="btn-danger btn-sm" on:click={() => removeFlow(flow.id)}>Delete</button>
          </div>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .panel {
    background: var(--bg-secondary);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 16px;
  }
  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .flow-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 0;
    border-bottom: 1px solid var(--border);
  }
  .muted {
    color: var(--text-muted);
    font-size: 11px;
  }
  .actions {
    display: flex;
    gap: 8px;
  }
</style>
