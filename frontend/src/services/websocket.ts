import { Task } from '../types';

const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/ws';

export interface WSMessage {
  type: string;
  action: 'created' | 'updated' | 'deleted';
  data: Task | Task[];
}

export type WSMessageHandler = (message: WSMessage) => void;

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private token: string | null = null;
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000; // Start with 1 second
  private reconnectTimeout: number | null = null;
  private messageHandlers: Set<WSMessageHandler> = new Set();
  private isIntentionallyClosed = false;

  constructor() {
    // Auto-connect will happen when token is set
  }

  setToken(token: string) {
    this.token = token;
    this.connect();
  }

  connect() {
    if (!this.token) {
      console.warn('WebSocket: Cannot connect without token');
      return;
    }

    if (this.ws?.readyState === WebSocket.OPEN) {
      console.log('WebSocket: Already connected');
      return;
    }

    this.isIntentionallyClosed = false;

    try {
      const wsUrl = `${WS_URL}?token=${this.token}`;
      this.ws = new WebSocket(wsUrl);

      this.ws.onopen = () => {
        console.log('WebSocket: Connected');
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data);
          console.log('WebSocket: Received message', message);

          // Notify all handlers
          this.messageHandlers.forEach(handler => {
            try {
              handler(message);
            } catch (error) {
              console.error('WebSocket: Error in message handler', error);
            }
          });
        } catch (error) {
          console.error('WebSocket: Failed to parse message', error);
        }
      };

      this.ws.onerror = (error) => {
        console.error('WebSocket: Error', error);
      };

      this.ws.onclose = (event) => {
        console.log('WebSocket: Connection closed', event.code, event.reason);
        this.ws = null;

        // Attempt to reconnect unless intentionally closed
        if (!this.isIntentionallyClosed) {
          this.scheduleReconnect();
        }
      };
    } catch (error) {
      console.error('WebSocket: Failed to connect', error);
      this.scheduleReconnect();
    }
  }

  private scheduleReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('WebSocket: Max reconnect attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1); // Exponential backoff

    console.log(`WebSocket: Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);

    this.reconnectTimeout = setTimeout(() => {
      this.connect();
    }, delay);
  }

  disconnect() {
    this.isIntentionallyClosed = true;

    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    console.log('WebSocket: Disconnected');
  }

  addMessageHandler(handler: WSMessageHandler) {
    this.messageHandlers.add(handler);
  }

  removeMessageHandler(handler: WSMessageHandler) {
    this.messageHandlers.delete(handler);
  }

  isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN;
  }

  send(data: unknown) {
    if (!this.isConnected()) {
      console.warn('WebSocket: Cannot send message, not connected');
      return;
    }

    try {
      this.ws?.send(JSON.stringify(data));
    } catch (error) {
      console.error('WebSocket: Failed to send message', error);
    }
  }
}

// Singleton instance
export const wsClient = new WebSocketClient();
