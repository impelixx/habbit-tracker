import { useState } from 'react';
import { useTelegram } from './hooks/useTelegram';
import { useAuth } from './hooks/useAuth';
import { useTasks } from './hooks/useTasks';
import { TaskList } from './components/TaskList';
import { TaskForm } from './components/TaskForm';
import type { Task, CreateTaskRequest } from './types';

function App() {
  const { hapticFeedback } = useTelegram();
  const { isAuthenticated, isLoading: authLoading, error: authError } = useAuth();
  const {
    tasks,
    isLoading: tasksLoading,
    error: tasksError,
    createTask,
    toggleTaskCompletion,
  } = useTasks();

  const [showForm, setShowForm] = useState(false);
  const [selectedTask, setSelectedTask] = useState<Task | null>(null);
  const [activeTab, setActiveTab] = useState<'all' | 'active' | 'completed'>('all');

  const handleCreateTask = async (taskData: CreateTaskRequest) => {
    const success = await createTask({ ...taskData, source: 'webapp' } as any);
    if (success) {
      setShowForm(false);
      hapticFeedback.notification('success');
    } else {
      hapticFeedback.notification('error');
    }
  };

  const handleToggleTask = async (id: string) => {
    await toggleTaskCompletion(id);
    hapticFeedback.impact('light');
  };

  const handleTaskClick = (task: Task) => {
    setSelectedTask(task);
    hapticFeedback.selection();
    // For now, just show task details (can extend to edit later)
  };

  const handleAddClick = () => {
    setShowForm(true);
    hapticFeedback.impact('medium');
  };

  // Filter tasks based on active tab
  const filteredTasks = tasks.filter((task) => {
    if (activeTab === 'active') return !task.completed;
    if (activeTab === 'completed') return task.completed;
    return true;
  });

  if (authLoading) {
    return <div className="loading">Loading...</div>;
  }

  if (authError) {
    return (
      <div className="container">
        <div className="error">
          <strong>Authentication Error</strong>
          <p>{authError}</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <div className="container">
        <div className="error">
          <strong>Not Authenticated</strong>
          <p>Please open this app from Telegram</p>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="header">
        <div className="header-title">📋 Habit Tracker</div>
        <div className="header-subtitle">
          {tasks.filter((t) => !t.completed).length} active tasks
        </div>
      </div>

      <div className="tabs">
        <button
          className={`tab ${activeTab === 'all' ? 'active' : ''}`}
          onClick={() => setActiveTab('all')}
        >
          All ({tasks.length})
        </button>
        <button
          className={`tab ${activeTab === 'active' ? 'active' : ''}`}
          onClick={() => setActiveTab('active')}
        >
          Active ({tasks.filter((t) => !t.completed).length})
        </button>
        <button
          className={`tab ${activeTab === 'completed' ? 'active' : ''}`}
          onClick={() => setActiveTab('completed')}
        >
          Completed ({tasks.filter((t) => t.completed).length})
        </button>
      </div>

      <div className="container">
        {tasksLoading ? (
          <div className="loading">Loading tasks...</div>
        ) : tasksError ? (
          <div className="error">
            <strong>Error</strong>
            <p>{tasksError}</p>
          </div>
        ) : (
          <TaskList
            tasks={filteredTasks}
            onToggleTask={handleToggleTask}
            onTaskClick={handleTaskClick}
          />
        )}
      </div>

      {!showForm && (
        <button className="fab" onClick={handleAddClick}>
          +
        </button>
      )}

      {showForm && (
        <div className="modal-overlay" onClick={() => setShowForm(false)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <div className="modal-title">New Task</div>
              <button className="modal-close" onClick={() => setShowForm(false)}>
                ×
              </button>
            </div>
            <div className="modal-body">
              <TaskForm
                onSubmit={handleCreateTask}
                onCancel={() => setShowForm(false)}
              />
            </div>
          </div>
        </div>
      )}

      {selectedTask && (
        <div className="modal-overlay" onClick={() => setSelectedTask(null)}>
          <div className="modal-content" onClick={(e) => e.stopPropagation()}>
            <div className="modal-header">
              <div className="modal-title">Task Details</div>
              <button className="modal-close" onClick={() => setSelectedTask(null)}>
                ×
              </button>
            </div>
            <div className="modal-body">
              <h3>{selectedTask.title}</h3>
              {selectedTask.description && <p>{selectedTask.description}</p>}
              <div className="task-meta">
                <span className={`task-priority ${selectedTask.priority}`}>
                  {selectedTask.priority}
                </span>
                <span>Source: {selectedTask.source}</span>
                <span>Status: {selectedTask.completed ? 'Completed' : 'Active'}</span>
              </div>
              <button
                className="button"
                style={{ marginTop: '16px' }}
                onClick={() => {
                  handleToggleTask(selectedTask.id);
                  setSelectedTask(null);
                }}
              >
                {selectedTask.completed ? 'Mark as Active' : 'Mark as Completed'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
