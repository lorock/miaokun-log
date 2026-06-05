export interface LogMatch {
  file: string;
  line_num: number;
  raw: string;
  before_context?: string[];
  after_context?: string[];
  context?: string[];
}

export interface SearchRequest {
  pattern: string;
  paths?: string[];
  level?: string;
  before?: number;
  after?: number;
  max_count?: number;
  case_insensitive?: boolean;
  since_days?: number;
  from?: string;
  to?: string;
}

export interface LogStats {
  total: number;
  by_level: Record<string, number>;
  total_files?: number;
}

export interface StreamEvent {
  type: 'match' | 'done' | 'error';
  data: LogMatch | { total_matches: number; duration_ms: number } | { message: string };
}

export interface FileInfo {
  name: string;
  path: string;
  full_path: string;
  size: number;
  size_readable: string;
  mod_time: string;
  mod_time_str: string;
  file_type: string;
  is_dir: boolean;
  is_readable: boolean;
}

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

export interface FileListResponse {
  success: boolean;
  data: FileInfo[];
  pagination: Pagination;
  error?: {
    code: string;
    message: string;
    details?: string;
  };
}

export interface FileListRequest {
  path?: string;
  page?: number;
  page_size?: number;
  since?: number;
  api_key?: string;
}
