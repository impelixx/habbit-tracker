import axios, { AxiosInstance, AxiosError } from 'axios';
import type {
  ApiResponse,
  AuthResponse,
  Task,
  CreateTaskRequest,
  UpdateTaskRequest,
  TaskStats,
} from '../types';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

class ApiService {
  private client: AxiosInstance;
  private token: string | null = null;

  constructor() {
    this.client = axios.create({
      baseURL: `${API_URL}/api`,
      timeout: 10000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add request interceptor to include auth token
    this.client.interceptors.request.use((config) => {
      if (this.token) {
        config.headers.Authorization = `Bearer ${this.token}`;
      }
      return config;
    });

    // Add response interceptor to handle errors
    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError<ApiResponse<any>>) => {
        if (error.response?.status === 401) {
          // Token expired or invalid
          this.clearToken();
        }
        throw error;
      }
    );

    // Load token from localStorage
    this.loadToken();
  }

  // Token management
  setToken(token: string) {
    this.token = token;
    localStorage.setItem('auth_token', token);
  }

  clearToken() {
    this.token = null;
    localStorage.removeItem('auth_token');
  }

  loadToken() {
    const token = localStorage.getItem('auth_token');
    if (token) {
      this.token = token;
    }
  }

  getToken(): string | null {
    return this.token;
  }

  // Auth endpoints
  async verifyAuth(initData: string): Promise<AuthResponse> {
    const response = await this.client.post<ApiResponse<AuthResponse>>('/auth/verify', {
      initData,
    });

    if (response.data.success && response.data.data) {
      this.setToken(response.data.data.token);
      return response.data.data;
    }

    throw new Error(response.data.error || 'Authentication failed');
  }

  // Task endpoints
  async getTasks(date?: string): Promise<Task[]> {
    const params = date ? { date } : {};
    const response = await this.client.get<ApiResponse<{ tasks: Task[] }>>('/tasks', { params });

    if (response.data.success && response.data.data) {
      return response.data.data.tasks;
    }

    return [];
  }

  async createTask(task: CreateTaskRequest): Promise<Task> {
    const response = await this.client.post<ApiResponse<{ task: Task }>>('/tasks', task);

    if (response.data.success && response.data.data) {
      return response.data.data.task;
    }

    throw new Error(response.data.error || 'Failed to create task');
  }

  async updateTask(id: string, updates: UpdateTaskRequest): Promise<Task> {
    const response = await this.client.patch<ApiResponse<{ task: Task }>>(`/tasks/${id}`, updates);

    if (response.data.success && response.data.data) {
      return response.data.data.task;
    }

    throw new Error(response.data.error || 'Failed to update task');
  }

  async deleteTask(id: string): Promise<void> {
    const response = await this.client.delete<ApiResponse<void>>(`/tasks/${id}`);

    if (!response.data.success) {
      throw new Error(response.data.error || 'Failed to delete task');
    }
  }

  // Statistics endpoint
  async getStats(period: 'week' | 'month' | 'year' = 'week'): Promise<TaskStats> {
    const response = await this.client.get<ApiResponse<TaskStats>>('/stats', {
      params: { period },
    });

    if (response.data.success && response.data.data) {
      return response.data.data;
    }

    throw new Error(response.data.error || 'Failed to get statistics');
  }
}

export const api = new ApiService();
