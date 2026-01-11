import { TaskItem } from './TaskItem';
import type { Task } from '../types';

interface TaskListProps {
  tasks: Task[];
  onToggleTask: (id: string) => void;
  onTaskClick: (task: Task) => void;
}

export const TaskList = ({ tasks, onToggleTask, onTaskClick }: TaskListProps) => {
  if (tasks.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">📋</div>
        <div className="empty-state-text">No tasks yet</div>
        <div className="empty-state-subtext">
          Tap the + button to create your first task
        </div>
      </div>
    );
  }

  // Separate completed and incomplete tasks
  const incompleteTasks = tasks.filter((task) => !task.completed);
  const completedTasks = tasks.filter((task) => task.completed);

  return (
    <div>
      {incompleteTasks.length > 0 && (
        <div className="task-list">
          {incompleteTasks.map((task) => (
            <TaskItem
              key={task.id}
              task={task}
              onToggle={onToggleTask}
              onClick={onTaskClick}
            />
          ))}
        </div>
      )}

      {completedTasks.length > 0 && (
        <>
          <div style={{ padding: '16px', color: 'var(--tg-theme-hint-color, #999)', fontSize: '14px' }}>
            Completed ({completedTasks.length})
          </div>
          <div className="task-list">
            {completedTasks.map((task) => (
              <TaskItem
                key={task.id}
                task={task}
                onToggle={onToggleTask}
                onClick={onTaskClick}
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
};
