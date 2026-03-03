import { writable, derived } from 'svelte/store';
import type { Task, Proxy, TaskEvent, TaskStatus } from './types';

export const tasks = writable<Task[]>([]);
export const proxies = writable<Proxy[]>([]);
export const selectedTaskId = writable<string | null>(null);
export const activeTab = writable<'tasks' | 'proxies' | 'logs'>('tasks');
export const statusFilter = writable<TaskStatus | 'all'>('all');

export const selectedTask = derived(
  [tasks, selectedTaskId],
  ([$tasks, $selectedTaskId]) => $tasks.find(t => t.id === $selectedTaskId) ?? null
);

export const filteredTasks = derived(
  [tasks, statusFilter],
  ([$tasks, $statusFilter]) => {
    if ($statusFilter === 'all') return $tasks;
    return $tasks.filter(t => t.status === $statusFilter);
  }
);

export const taskStats = derived(tasks, ($tasks) => {
  const stats: Record<string, number> = {
    total: $tasks.length,
    pending: 0,
    queued: 0,
    running: 0,
    completed: 0,
    failed: 0,
    cancelled: 0,
    retrying: 0,
  };
  for (const t of $tasks) {
    stats[t.status] = (stats[t.status] || 0) + 1;
  }
  return stats;
});

// Update a single task in the store
export function updateTaskInStore(event: TaskEvent) {
  tasks.update(list => 
    list.map(t => 
      t.id === event.taskId 
        ? { ...t, status: event.status, error: event.error || t.error }
        : t
    )
  );
}
