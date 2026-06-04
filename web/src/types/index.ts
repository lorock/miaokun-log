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
