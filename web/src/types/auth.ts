export type Role = 'admin' | 'user' | 'viewer';

export type Permission = 'search' | 'file_browse' | 'file_read' | 'admin';

export interface User {
  id: string;
  username: string;
  roles: Role[];
  created_at?: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_at: number;
  token_type: string;
}

export interface AuthState {
  isAuthenticated: boolean;
  user: User | null;
  token: TokenPair | null;
  isLoading: boolean;
  error: string | null;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  success: boolean;
  message?: string;
  data?: {
    user: User;
    token: TokenPair;
  };
  error?: AuthError;
}

export interface RefreshRequest {
  refresh_token: string;
}

export interface RefreshResponse {
  success: boolean;
  message?: string;
  data?: {
    access_token: string;
    expires_at: number;
  };
  error?: AuthError;
}

export interface LogoutResponse {
  success: boolean;
  message?: string;
  error?: AuthError;
}

export interface AuthError {
  code: string;
  message: string;
  details?: string;
}

export interface CurrentUserResponse {
  success: boolean;
  data?: User;
  error?: AuthError;
}

export const STORAGE_KEYS = {
  AUTH_TOKEN: 'miaokun_auth_token',
  AUTH_USER: 'miaokun_auth_user',
  AUTH_EXPIRES_AT: 'miaokun_auth_expires_at',
  AUTH_REFRESH_TOKEN: 'miaokun_auth_refresh_token',
};

export const AUTH_EVENTS = {
  LOGIN: 'auth_login',
  LOGOUT: 'auth_logout',
  TOKEN_REFRESH: 'auth_token_refresh',
  AUTH_ERROR: 'auth_error',
  UNAUTHORIZED: 'auth_unauthorized',
};

// API error codes
export const AUTH_ERROR_CODES = {
  AUTHENTICATION_REQUIRED: 'AUTHENTICATION_REQUIRED',
  INVALID_CREDENTIALS: 'INVALID_CREDENTIALS',
  TOKEN_EXPIRED: 'TOKEN_EXPIRED',
  PERMISSION_DENIED: 'PERMISSION_DENIED',
  FORBIDDEN: 'FORBIDDEN',
  NOT_AUTHENTICATED: 'NOT_AUTHENTICATED',
};

// User role display names
export const ROLE_DISPLAY_NAMES: Record<Role, string> = {
  admin: '管理员',
  user: '普通用户',
  viewer: '访客',
};

// Permission descriptions
export const PERMISSION_DESCRIPTIONS: Record<Permission, string> = {
  search: '搜索日志',
  file_browse: '浏览文件',
  file_read: '读取文件',
  admin: '系统管理',
};

// Role permissions mapping
export const ROLE_PERMISSIONS: Record<Role, Permission[]> = {
  admin: ['search', 'file_browse', 'file_read', 'admin'],
  user: ['search', 'file_browse', 'file_read'],
  viewer: ['search'],
};
