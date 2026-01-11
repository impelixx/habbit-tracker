// User types
export interface User {
  id: string;
  telegramId: number;
  username: string;
  firstName: string;
  lastName: string;
  languageCode: string;
  timezone: string;
  preferences: UserPreferences;
  createdAt: string;
  updatedAt: string;
}

export interface UserPreferences {
  reminderEnabled: boolean;
  reminderTime: string;
  theme: string;
}

// Task types
export interface Task {
  id: string;
  userId: string;
  title: string;
  description: string;
  priority: 'low' | 'medium' | 'high';
  dueDate: string;
  completed: boolean;
  completedAt?: string;
  source: 'bot' | 'webapp' | 'voice';
  metadata?: TaskMetadata;
  createdAt: string;
  updatedAt: string;
}

export interface TaskMetadata {
  transcription?: string;
  aiModel?: string;
}

export interface CreateTaskRequest {
  title: string;
  description?: string;
  priority?: 'low' | 'medium' | 'high';
  dueDate?: string;
}

export interface UpdateTaskRequest {
  title?: string;
  description?: string;
  priority?: 'low' | 'medium' | 'high';
  dueDate?: string;
  completed?: boolean;
}

// Statistics types
export interface TaskStats {
  totalTasks: number;
  completedTasks: number;
  completionRate: number;
  streak: number;
  tasksByPriority: {
    high: number;
    medium: number;
    low: number;
  };
  tasksBySource: {
    bot: number;
    webapp: number;
    voice: number;
  };
}

// Auth types
export interface AuthResponse {
  token: string;
  user: User;
}

// API Response types
export interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// WebSocket message types
export interface WebSocketMessage {
  type: 'task_created' | 'task_updated' | 'task_deleted' | 'ping' | 'pong';
  payload?: any;
}
