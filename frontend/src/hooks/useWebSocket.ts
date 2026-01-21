import { useEffect, useCallback, useState } from 'react';
import { wsClient, WSMessage, WSMessageHandler } from '../services/websocket';
import { Task } from '../types';

interface UseWebSocketOptions {
  onTaskCreated?: (task: Task) => void;
  onTaskUpdated?: (task: Task) => void;
  onTaskDeleted?: (task: Task) => void;
}

export const useWebSocket = (options: UseWebSocketOptions = {}) => {
  const [isConnected, setIsConnected] = useState(wsClient.isConnected());

  const handleMessage: WSMessageHandler = useCallback((message: WSMessage) => {
    if (message.type === 'task_update') {
      const task = message.data as Task;

      switch (message.action) {
        case 'created':
          options.onTaskCreated?.(task);
          break;
        case 'updated':
          options.onTaskUpdated?.(task);
          break;
        case 'deleted':
          options.onTaskDeleted?.(task);
          break;
      }
    } else if (message.type === 'tasks_batch') {
      const tasks = message.data as Task[];

      if (message.action === 'created') {
        tasks.forEach(task => options.onTaskCreated?.(task));
      }
    }
  }, [options]);

  useEffect(() => {
    // Add message handler
    wsClient.addMessageHandler(handleMessage);

    // Check connection status periodically
    const interval = setInterval(() => {
      setIsConnected(wsClient.isConnected());
    }, 1000);

    return () => {
      wsClient.removeMessageHandler(handleMessage);
      clearInterval(interval);
    };
  }, [handleMessage]);

  const connect = useCallback((token: string) => {
    wsClient.setToken(token);
  }, []);

  const disconnect = useCallback(() => {
    wsClient.disconnect();
    setIsConnected(false);
  }, []);

  return {
    isConnected,
    connect,
    disconnect,
  };
};
