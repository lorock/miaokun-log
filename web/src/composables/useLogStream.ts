import { ref, computed } from 'vue';
import type { LogMatch, LogStats, SearchRequest } from '../types';

const MAX_LOGS = 50000;
const UPDATE_INTERVAL = 100;

function extractLevel(line: string): string {
  const upper = line.toUpperCase();
  const levels = ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'];
  for (const l of levels) {
    if (upper.includes(l)) {
      return l;
    }
  }
  return 'OTHER';
}

export function useLogStream() {
  const logs = ref<LogMatch[]>([]);
  const isStreaming = ref(false);
  const stats = ref<LogStats>({
    total: 0,
    by_level: {},
    total_files: 0,
  });
  const error = ref<string | null>(null);
  const reachedLimit = ref(false);
  const searchDurationMs = ref(0);
  
  // 进度状态
  const progress = ref({
    currentFile: 0,
    totalFiles: 0,
    currentFileName: '',
  });
  
  let abortController: AbortController | null = null;
  let pendingLogs: LogMatch[] = [];
  let updateTimer: ReturnType<typeof setTimeout> | null = null;
  let totalReceived = 0;

  const flushPendingLogs = () => {
    if (pendingLogs.length === 0) return;
    
    const newLogs = pendingLogs;
    pendingLogs = [];
    
    if (logs.value.length + newLogs.length > MAX_LOGS) {
      const overflow = logs.value.length + newLogs.length - MAX_LOGS;
      logs.value = logs.value.slice(overflow);
      reachedLimit.value = true;
    }
    
    logs.value = [...logs.value, ...newLogs];
  };

  const start = async (request: SearchRequest) => {
    stop();
    
    logs.value = [];
    pendingLogs = [];
    totalReceived = 0;
    stats.value = { total: 0, by_level: {}, total_files: 0 };
    error.value = null;
    reachedLimit.value = false;
    isStreaming.value = true;
    const startTime = performance.now();

    abortController = new AbortController();
    
    updateTimer = setInterval(flushPendingLogs, UPDATE_INTERVAL);

    try {
      const response = await fetch('/api/v1/search/stream', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(request),
        signal: abortController.signal,
      });

      if (!response.ok) {
        if (response.status === 401) {
          error.value = '登录已过期，请重新登录后再搜索';
        } else if (response.status === 403) {
          error.value = '没有权限访问该资源';
        } else {
          error.value = `请求失败 (${response.status})，请重试`;
        }
        isStreaming.value = false;
        if (updateTimer) {
          clearInterval(updateTimer);
          updateTimer = null;
        }
        return;
      }

      if (!response.body) {
        error.value = '响应体为空';
        isStreaming.value = false;
        if (updateTimer) {
          clearInterval(updateTimer);
          updateTimer = null;
        }
        return;
      }

      const reader = response.body.getReader();
      const decoder = new TextDecoder('utf-8');
      let bufferStr = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        bufferStr += decoder.decode(value, { stream: true });

        while (true) {
          const lineEnd = bufferStr.indexOf('\n\n');
          if (lineEnd === -1) break;

          const line = bufferStr.substring(0, lineEnd);
          bufferStr = bufferStr.substring(lineEnd + 2);

          if (line.startsWith('data: ')) {
            const dataStr = line.substring(6);
            try {
              const data = JSON.parse(dataStr);

              if (data.type === 'match') {
                const match = data.data as LogMatch;
                pendingLogs.push(match);
                totalReceived++;
                stats.value.total++;
                const level = extractLevel(match.raw);
                if (!stats.value.by_level[level]) {
                  stats.value.by_level[level] = 0;
                }
                stats.value.by_level[level]++;
              } else if (data.type === 'progress') {
                // 进度更新
                progress.value.currentFile = data.data.current_file || 0;
                progress.value.totalFiles = data.data.total_files || 0;
                progress.value.currentFileName = data.data.file_name || '';
              } else if (data.type === 'done') {
                console.log('搜索完成:', data.data);
              } else if (data.type === 'error') {
                error.value = data.data.message;
              }
            } catch (err) {
              console.error('解析日志失败:', err);
            }
          }
        }
      }

      flushPendingLogs();
      if (updateTimer) {
        clearInterval(updateTimer);
        updateTimer = null;
      }
      isStreaming.value = false;
      searchDurationMs.value = Math.round(performance.now() - startTime);
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        error.value = err.message;
      }
      flushPendingLogs();
      if (updateTimer) {
        clearInterval(updateTimer);
        updateTimer = null;
      }
      isStreaming.value = false;
      searchDurationMs.value = Math.round(performance.now() - startTime);
    }
  };

  const stop = () => {
    if (abortController) {
      abortController.abort();
      abortController = null;
    }
    if (updateTimer) {
      clearInterval(updateTimer);
      updateTimer = null;
    }
    flushPendingLogs();
    isStreaming.value = false;
  };

  const clear = () => {
    logs.value = [];
    pendingLogs = [];
    totalReceived = 0;
    stats.value = { total: 0, by_level: {}, total_files: 0 };
    error.value = null;
    reachedLimit.value = false;
  };

  const displayTotal = computed(() => totalReceived);
  const displayCount = computed(() => logs.value.length);
  const isOverLimit = computed(() => reachedLimit.value);

  return {
    logs,
    stats,
    isStreaming,
    error,
    reachedLimit,
    displayTotal,
    displayCount,
    isOverLimit,
    start,
    stop,
    clear,
    MAX_LOGS,
    searchDurationMs,
    progress,
  };
}