import { reactive, watch, computed } from 'vue';
import type {
  AuthState,
  User,
  TokenPair,
  LoginRequest,
  LoginResponse,
  RefreshResponse,
  Permission,
  Role,
} from '../types/auth';
import {
  STORAGE_KEYS,
  AUTH_EVENTS,
  ROLE_PERMISSIONS,
} from '../types/auth';

const TOKEN_REFRESH_THRESHOLD = 60 * 1000;
const REFRESH_INTERVAL = 5 * 60 * 1000;

const authState = reactive<AuthState>({
  isAuthenticated: false,
  user: null,
  token: null,
  isLoading: false,
  error: null,
});

let refreshInterval: ReturnType<typeof setInterval> | null = null;
let originalFetch: typeof window.fetch | null = null;
let authInitialized = false;

const saveToStorage = (tokenInfo: TokenPair, userInfo: User) => {
  try {
    localStorage.setItem(STORAGE_KEYS.AUTH_TOKEN, tokenInfo.access_token);
    localStorage.setItem(STORAGE_KEYS.AUTH_REFRESH_TOKEN, tokenInfo.refresh_token);
    localStorage.setItem(STORAGE_KEYS.AUTH_USER, JSON.stringify(userInfo));
    localStorage.setItem(STORAGE_KEYS.AUTH_EXPIRES_AT, tokenInfo.expires_at.toString());
  } catch (e) {
    console.warn('Failed to save auth data to storage:', e);
  }
};

const loadFromStorage = (): { token: TokenPair | null; user: User | null } => {
  try {
    const accessToken = localStorage.getItem(STORAGE_KEYS.AUTH_TOKEN);
    const refreshToken = localStorage.getItem(STORAGE_KEYS.AUTH_REFRESH_TOKEN);
    const userStr = localStorage.getItem(STORAGE_KEYS.AUTH_USER);
    const expiresAtStr = localStorage.getItem(STORAGE_KEYS.AUTH_EXPIRES_AT);

    if (accessToken && refreshToken && userStr && expiresAtStr) {
      const expiresAt = parseInt(expiresAtStr, 10);
      if (Date.now() < expiresAt - TOKEN_REFRESH_THRESHOLD) {
        return {
          token: {
            access_token: accessToken,
            refresh_token: refreshToken,
            expires_at: expiresAt,
            token_type: 'Bearer',
          },
          user: JSON.parse(userStr),
        };
      }
    }
  } catch (e) {
    console.warn('Failed to load auth data from storage:', e);
  }
  return { token: null, user: null };
};

const clearStorage = () => {
  try {
    localStorage.removeItem(STORAGE_KEYS.AUTH_TOKEN);
    localStorage.removeItem(STORAGE_KEYS.AUTH_REFRESH_TOKEN);
    localStorage.removeItem(STORAGE_KEYS.AUTH_USER);
    localStorage.removeItem(STORAGE_KEYS.AUTH_EXPIRES_AT);
  } catch (e) {
    console.warn('Failed to clear auth data from storage:', e);
  }
};

const triggerEvent = (eventName: string, detail?: unknown) => {
  window.dispatchEvent(new CustomEvent(eventName, { detail }));
};

const updateState = (newState: Partial<AuthState>) => {
  Object.assign(authState, newState);
};

const startRefreshTimer = () => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
  }
  refreshInterval = setInterval(() => {
    if (!authState.token) return;
    const now = Date.now();
    const expiresAt = authState.token.expires_at;
    if (expiresAt - now < TOKEN_REFRESH_THRESHOLD) {
      refreshToken();
    }
  }, REFRESH_INTERVAL);
};

const initAuth = () => {
  const { token: storedToken, user: storedUser } = loadFromStorage();

  if (storedToken && storedUser) {
    updateState({
      isAuthenticated: true,
      user: storedUser,
      token: storedToken,
      isLoading: false,
      error: null,
    });
    startRefreshTimer();
    if (!authState.token) return;
    const now = Date.now();
    if (authState.token.expires_at - now < TOKEN_REFRESH_THRESHOLD) {
      refreshToken();
    }
  } else {
    updateState({
      isAuthenticated: false,
      user: null,
      token: null,
      isLoading: false,
      error: null,
    });
  }
};

const createAuthInterceptor = () => {
  if (originalFetch) return;

  const originalFetchRef = window.fetch;
  originalFetch = originalFetchRef;

  window.fetch = async (...args: Parameters<typeof fetch>) => {
    const [url, options = {}] = args;

    if (typeof url === 'string' && url.startsWith('/api/v1/')) {
      const newOptions = { ...options };
      newOptions.headers = { ...((options.headers as Record<string, string>) || {}) };

      if (authState.token?.access_token) {
        (newOptions.headers as Record<string, string>).Authorization = `Bearer ${authState.token.access_token}`;
      }

      try {
        const response = await originalFetchRef(url, newOptions);

        if (response.status === 401) {
          let errorMessage = '登录已过期，请重新登录';
          let errorCode = '';

          try {
            const data = await response.clone().json();
            if (data.error) {
              errorMessage = data.error.message || errorMessage;
              errorCode = data.error.code || '';
            }
          } catch (parseError) {
            // Response may not be JSON, use default message
          }

          // Check if this is a login/refresh endpoint failure - those should not force logout
          const isAuthEndpoint =
            url.endsWith('/api/v1/auth/login') ||
            url.endsWith('/api/v1/auth/refresh') ||
            url.endsWith('/api/v1/auth/logout');

          if (!isAuthEndpoint) {
            await logout();
            triggerEvent(AUTH_EVENTS.UNAUTHORIZED, {
              message: errorMessage,
              code: errorCode,
            });
          }
        }

        return response;
      } catch (e) {
        return originalFetchRef(...args);
      }
    }

    return originalFetchRef(...args);
  };
};

const login = async (credentials: LoginRequest): Promise<boolean> => {
  updateState({ isLoading: true, error: null });

  try {
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(credentials),
    });

    const data: LoginResponse = await res.json();

    if (data.success && data.data) {
      const { user: userInfo, token: tokenInfo } = data.data;
      tokenInfo.expires_at = tokenInfo.expires_at * 1000;
      saveToStorage(tokenInfo, userInfo);
      updateState({
        isAuthenticated: true,
        user: userInfo,
        token: tokenInfo,
        isLoading: false,
        error: null,
      });
      startRefreshTimer();
      triggerEvent(AUTH_EVENTS.LOGIN, { user: userInfo });
      return true;
    } else {
      updateState({
        isLoading: false,
        error: data.error?.message || '登录失败',
      });
      return false;
    }
  } catch (e) {
    updateState({
      isLoading: false,
      error: '网络请求失败，请检查服务器连接',
    });
    return false;
  }
};

const logout = async () => {
  if (refreshInterval) {
    clearInterval(refreshInterval);
    refreshInterval = null;
  }

  try {
    await fetch('/api/v1/auth/logout', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${authState.token?.access_token}`,
      },
    });
  } catch (e) {
    console.warn('Logout request failed:', e);
  }

  clearStorage();
  updateState({
    isAuthenticated: false,
    user: null,
    token: null,
    isLoading: false,
    error: null,
  });
  triggerEvent(AUTH_EVENTS.LOGOUT);
};

const refreshToken = async (): Promise<boolean> => {
  if (!authState.token?.refresh_token) return false;

  try {
    const res = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ refresh_token: authState.token.refresh_token }),
    });

    const data: RefreshResponse = await res.json();

    if (data.success && data.data) {
      const newToken: TokenPair = {
        ...authState.token,
        access_token: data.data.access_token,
        expires_at: data.data.expires_at * 1000,
      };

      if (authState.user) {
        localStorage.setItem(STORAGE_KEYS.AUTH_TOKEN, newToken.access_token);
        localStorage.setItem(STORAGE_KEYS.AUTH_EXPIRES_AT, newToken.expires_at.toString());
      }

      updateState({ token: newToken });
      triggerEvent(AUTH_EVENTS.TOKEN_REFRESH);
      return true;
    } else {
      await logout();
      return false;
    }
  } catch (e) {
    console.warn('Token refresh failed:', e);
    await logout();
    return false;
  }
};

const hasPermission = (permission: Permission): boolean => {
  if (!authState.user) return false;

  for (const role of authState.user.roles) {
    const perms = ROLE_PERMISSIONS[role as Role] || [];
    if (perms.includes(permission)) {
      return true;
    }
  }
  return false;
};

const hasRole = (role: Role): boolean => {
  return authState.user?.roles.includes(role) ?? false;
};

export function useAuth() {
  if (!authInitialized) {
    authInitialized = true;
    initAuth();
    createAuthInterceptor();
  }

  watch(
    () => authState.isAuthenticated,
    (newVal) => {
      if (!newVal && refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
      }
    }
  );

  return {
    isAuthenticated: computed(() => authState.isAuthenticated),
    user: computed(() => authState.user),
    token: computed(() => authState.token),
    isLoading: computed(() => authState.isLoading),
    error: computed(() => authState.error),
    isAdmin: computed(() => hasRole('admin')),
    login,
    logout,
    refreshToken,
    hasPermission,
    hasRole,
    initAuth,
  };
}

export function useAuthState() {
  if (!authInitialized) {
    authInitialized = true;
    initAuth();
    createAuthInterceptor();
  }

  return {
    isAuthenticated: computed(() => authState.isAuthenticated),
    user: computed(() => authState.user),
    token: computed(() => authState.token),
    isLoading: computed(() => authState.isLoading),
    error: computed(() => authState.error),
  };
}
