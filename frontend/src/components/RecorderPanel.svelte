<script lang="ts">
  import { CreateRecordedFlow } from '../../wailsjs/go/main/App';
  import type { RecordedStep } from '../lib/types';
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let recording = false;
  let flowName = '';
  let flowDescription = '';
  let originUrl = '';
  let steps: RecordedStep[] = [];
  let errorMessage = '';
  let saving = false;

  function toggleRecording() {
    recording = !recording;
    if (!recording) return;
    steps = [];
  }

  function addStep(step: RecordedStep) {
    steps = [...steps, step];
  }

  async function saveFlow() {
    if (!flowName || steps.length === 0) return;
    saving = true;
    try {
      errorMessage = '';
      await CreateRecordedFlow(flowName, flowDescription, originUrl, steps);
      dispatch('saved');
      flowName = '';
      flowDescription = '';
      steps = [];
      recording = false;
    } catch (err: any) {
      errorMessage = err?.message || String(err);
    } finally {
      saving = false;
    }
  }
</script>

<div class="panel">
  <div class="panel-header">
    <h2>Live Recorder</h2>
    <button class="btn-primary" on:click={toggleRecording}>
      {recording ? 'Stop Recording' : 'Start Recording'}
    </button>
  </div>

  <div class="panel-body">
    <div class="form-group">
      <label for="flow-name">Flow Name</label>
      <input id="flow-name" bind:value={flowName} placeholder="Checkout flow" />
    </div>
    <div class="form-group">
      <label for="flow-desc">Description</label>
      <input id="flow-desc" bind:value={flowDescription} placeholder="Optional" />
    </div>
    <div class="form-group">
      <label for="origin-url">Origin URL</label>
      <input id="origin-url" bind:value={originUrl} placeholder="https://example.com" />
    </div>

    <div class="steps">
      <h4>Recorded Steps ({steps.length})</h4>
      {#if steps.length === 0}
        <div class="empty">No steps recorded yet.</div>
      {:else}
        <ul>
          {#each steps as step}
            <li>
              <strong>{step.action}</strong>
              {#if step.selector} <span class="muted">{step.selector}</span> {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  </div>

  {#if errorMessage}
    <div class="error-banner">{errorMessage}</div>
  {/if}

  <div class="panel-footer">
    <button class="btn-secondary" on:click={() => dispatch('close')}>Close</button>
    <button class="btn-primary" disabled={!flowName || steps.length === 0 || saving} on:click={saveFlow}>
      {saving ? 'Saving...' : 'Save Recorded Flow'}
    </button>
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
  .panel-body {
    margin-top: 12px;
  }
  .steps {
    margin-top: 16px;
  }
  .steps ul {
    list-style: none;
    padding: 0;
  }
  .steps li {
    padding: 6px 0;
    border-bottom: 1px solid var(--border);
  }
  .empty {
    font-size: 12px;
    color: var(--text-muted);
  }
  .muted {
    color: var(--text-muted);
    font-size: 11px;
  }
  .panel-footer {
    margin-top: 16px;
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
</style>
