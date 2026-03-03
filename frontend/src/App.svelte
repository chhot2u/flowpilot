<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import Header from './components/Header.svelte';
  import TaskToolbar from './components/TaskToolbar.svelte';
  import TaskTable from './components/TaskTable.svelte';
  import TaskDetail from './components/TaskDetail.svelte';
  import CreateTaskModal from './components/CreateTaskModal.svelte';
  import ProxyPanel from './components/ProxyPanel.svelte';
  import { tasks, activeTab, updateTaskInStore } from './lib/store';
  import { ListTasks } from '../wailsjs/go/main/App';
  import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime';

  let showCreateModal = false;

  async function refreshTasks() {
    try {
      const list = await ListTasks();
      tasks.set(list || []);
    } catch (err) {
      console.error('Failed to load tasks:', err);
    }
  }

  onMount(() => {
    refreshTasks();

    // Listen for real-time task events from Go backend
    EventsOn('task:event', (event: any) => {
      updateTaskInStore(event);
    });
  });

  onDestroy(() => {
    EventsOff('task:event');
  });

  // Auto-refresh every 5 seconds
  const interval = setInterval(refreshTasks, 5000);
  onDestroy(() => clearInterval(interval));
</script>

<div class="app-layout">
  <Header />

  <nav class="tabs">
    <button
      class="tab"
      class:active={$activeTab === 'tasks'}
      on:click={() => $activeTab = 'tasks'}
    >
      Tasks
    </button>
    <button
      class="tab"
      class:active={$activeTab === 'proxies'}
      on:click={() => $activeTab = 'proxies'}
    >
      Proxies
    </button>
  </nav>

  {#if $activeTab === 'tasks'}
    <TaskToolbar on:create={() => showCreateModal = true} />
    <div class="main-content">
      <TaskTable on:refresh={refreshTasks} />
      <TaskDetail />
    </div>
  {:else if $activeTab === 'proxies'}
    <ProxyPanel />
  {/if}
</div>

{#if showCreateModal}
  <CreateTaskModal
    on:close={() => showCreateModal = false}
    on:created={refreshTasks}
  />
{/if}

<style>
  .app-layout {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  .tabs {
    display: flex;
    gap: 0;
    background: var(--bg-secondary);
    border-bottom: 1px solid var(--border);
    padding: 0 20px;
    flex-shrink: 0;
  }
  .tab {
    padding: 10px 20px;
    background: none;
    color: var(--text-muted);
    font-size: 13px;
    font-weight: 500;
    border-radius: 0;
    border-bottom: 2px solid transparent;
    transition: all 0.15s ease;
  }
  .tab:hover {
    color: var(--text-primary);
  }
  .tab.active {
    color: var(--accent);
    border-bottom-color: var(--accent);
  }

  .main-content {
    display: flex;
    flex: 1;
    overflow: hidden;
  }
</style>
