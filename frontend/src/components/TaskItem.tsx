import type { Task } from '../types';

interface TaskItemProps {
  task: Task;
  onToggle: (id: string) => void;
  onClick: (task: Task) => void;
}

export const TaskItem = ({ task, onToggle, onClick }: TaskItemProps) => {
  const handleCheckboxClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onToggle(task.id);
  };

  return (
    <div
      className={`task-item ${task.completed ? 'completed' : ''}`}
      onClick={() => onClick(task)}
    >
      <div
        className={`task-checkbox ${task.completed ? 'checked' : ''}`}
        onClick={handleCheckboxClick}
      >
        {task.completed && '✓'}
      </div>

      <div className="task-content">
        <div className={`task-title ${task.completed ? 'completed' : ''}`}>
          {task.title}
        </div>

        {task.description && (
          <div className="task-description">{task.description}</div>
        )}

        <div className="task-meta">
          <span className={`task-priority ${task.priority}`}>
            {task.priority}
          </span>
          <span className="task-source">
            {task.source === 'bot' && '🤖'}
            {task.source === 'webapp' && '📱'}
            {task.source === 'voice' && '🎤'}
            {task.source}
          </span>
        </div>
      </div>
    </div>
  );
};
