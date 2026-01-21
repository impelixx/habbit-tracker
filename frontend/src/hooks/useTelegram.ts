import { useEffect, useState } from 'react';

interface TelegramWebApp {
  initData: string;
  initDataUnsafe: {
    query_id?: string;
    user?: {
      id: number;
      first_name: string;
      last_name?: string;
      username?: string;
      language_code?: string;
    };
    auth_date?: number;
    hash?: string;
  };
  version: string;
  platform: string;
  colorScheme: 'light' | 'dark';
  themeParams: {
    bg_color?: string;
    text_color?: string;
    hint_color?: string;
    link_color?: string;
    button_color?: string;
    button_text_color?: string;
  };
  isExpanded: boolean;
  viewportHeight: number;
  viewportStableHeight: number;
  headerColor: string;
  backgroundColor: string;
  BackButton: {
    isVisible: boolean;
    onClick: (callback: () => void) => void;
    offClick: (callback: () => void) => void;
    show: () => void;
    hide: () => void;
  };
  MainButton: {
    text: string;
    color: string;
    textColor: string;
    isVisible: boolean;
    isActive: boolean;
    isProgressVisible: boolean;
    setText: (text: string) => void;
    onClick: (callback: () => void) => void;
    offClick: (callback: () => void) => void;
    show: () => void;
    hide: () => void;
    enable: () => void;
    disable: () => void;
    showProgress: (leaveActive?: boolean) => void;
    hideProgress: () => void;
  };
  HapticFeedback: {
    impactOccurred: (style: 'light' | 'medium' | 'heavy' | 'rigid' | 'soft') => void;
    notificationOccurred: (type: 'error' | 'success' | 'warning') => void;
    selectionChanged: () => void;
  };
  ready: () => void;
  expand: () => void;
  close: () => void;
  enableClosingConfirmation: () => void;
  disableClosingConfirmation: () => void;
}

declare global {
  interface Window {
    Telegram?: {
      WebApp: TelegramWebApp;
    };
  }
}

export const useTelegram = () => {
  const [webApp, setWebApp] = useState<TelegramWebApp | null>(null);
  const [isReady, setIsReady] = useState(false);

  useEffect(() => {
    const tg = window.Telegram?.WebApp;

    if (tg) {
      tg.ready();
      tg.expand();
      setWebApp(tg);
      setIsReady(true);

      // Set theme colors
      if (tg.themeParams.bg_color) {
        document.body.style.backgroundColor = tg.themeParams.bg_color;
      }
    } else {
      // For development without Telegram
      console.warn('Telegram WebApp not available. Running in standalone mode.');
      setIsReady(true);
    }
  }, []);

  return {
    webApp,
    isReady,
    user: webApp?.initDataUnsafe?.user,
    initData: webApp?.initData || '',
    colorScheme: webApp?.colorScheme || 'light',
    themeParams: webApp?.themeParams || {},
    platform: webApp?.platform || 'unknown',

    // Helper methods
    showBackButton: () => webApp?.BackButton.show(),
    hideBackButton: () => webApp?.BackButton.hide(),
    onBackButton: (callback: () => void) => webApp?.BackButton.onClick(callback),

    showMainButton: (text: string, callback: () => void) => {
      if (webApp) {
        webApp.MainButton.setText(text);
        webApp.MainButton.onClick(callback);
        webApp.MainButton.show();
      }
    },
    hideMainButton: () => webApp?.MainButton.hide(),

    hapticFeedback: {
      impact: (style: 'light' | 'medium' | 'heavy' = 'medium') =>
        webApp?.HapticFeedback.impactOccurred(style),
      notification: (type: 'error' | 'success' | 'warning') =>
        webApp?.HapticFeedback.notificationOccurred(type),
      selection: () => webApp?.HapticFeedback.selectionChanged(),
    },

    close: () => webApp?.close(),
  };
};
