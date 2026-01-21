import { useState, useEffect, useCallback } from 'react';
import { api } from '../services/api';
import type { Task, CreateTaskRequest, UpdateTaskRequest } from '../types';

export const useTasks = (date?: string) => {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchTasks = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const fetchedTasks = await api.getTasks(date);
      setTasks(fetchedTasks);
    } catch (err) {
      console.error('Error fetching tasks:', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch tasks');
    } finally {
      setIsLoading(false);
    }
  }, [date]);

  useEffect(() => {
    fetchTasks();
  }, [fetchTasks]);

  const createTask = async (taskData: CreateTaskRequest): Promise<Task | null> => {
    try {
      const newTask = await api.createTask(taskData);
      setTasks((prev) => [...prev, newTask]);
      return newTask;
    } catch (err) {
      console.error('Error creating task:', err);
      setError(err instanceof Error ? err.message : 'Failed to create task');
      return null;
    }
  };

  const updateTask = async (id: string, updates: UpdateTaskRequest): Promise<Task | null> => {
    try {
      const updatedTask = await api.updateTask(id, updates);
      setTasks((prev) => prev.map((task) => (task.id === id ? updatedTask : task)));
      return updatedTask;
    } catch (err) {
      console.error('Error updating task:', err);
      setError(err instanceof Error ? err.message : 'Failed to update task');
      return null;
    }
  };

  const deleteTask = async (id: string): Promise<boolean> => {
    try {
      await api.deleteTask(id);
      setTasks((prev) => prev.filter((task) => task.id !== id));
      return true;
    } catch (err) {
      console.error('Error deleting task:', err);
      setError(err instanceof Error ? err.message : 'Failed to delete task');
      return false;
    }
  };

  const toggleTaskCompletion = async (id: string): Promise<Task | null> => {
    const task = tasks.find((t) => t.id === id);
    if (!task) return null;

    return updateTask(id, { completed: !task.completed });
  };

  // WebSocket update handlers
  const handleTaskCreated = useCallback((task: Task) => {
    // Only add if not already in list (avoid duplicates from our own actions)
    setTasks((prev) => {
      if (prev.some(t => t.id === task.id)) {
        return prev;
      }
      return [...prev, task];
    });
  }, []);

  const handleTaskUpdated = useCallback((task: Task) => {
    setTasks((prev) => prev.map((t) => (t.id === task.id ? task : t)));
  }, []);

  const handleTaskDeleted = useCallback((task: Task) => {
    setTasks((prev) => prev.filter((t) => t.id !== task.id));
  }, []);

  return {
    tasks,
    isLoading,
    error,
    refetch: fetchTasks,
    createTask,
    updateTask,
    deleteTask,
    toggleTaskCompletion,
    // WebSocket handlers
    handleTaskCreated,
    handleTaskUpdated,
    handleTaskDeleted,
  };
};
