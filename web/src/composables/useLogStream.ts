import { ref } from 'vue';
import type { LogMatch, LogStats, SearchRequest } from '../types';

export class RingBuffer<T> {
  private buffer: (T | null)[];
  private start = 0;
  private count = 0;

  constructor(public readonly capacity: number = 50000) {
    this.buffer = new Array(capacity).fill(null);
  }

  append(item: T): void {
    const idx = (this.start + this.count) % this.capacity;
    this.buffer[idx] = item;
    if (this.count < this.capacity) {
      this.count++;
    } else {
      this.start = (this.start + 1) % this.capacity;
    }
  }

  get(index: number): T | null {
    if (index < 0 || index >= this.count) return null;
    return this.buffer[(this.start + index) % this.capacity];
  }

  size(): number {
    return this.count;
  }

  toArray(): T[] {
    const result: T[] = [];
    for (let i = 0; i < this.count; i++) {
      const item = this.get(i);
      if (item !== null) {
        result.push(item);
      }
    }
    return result;
  }

  clear(): void {
    this.start = 0;
    this.count = 0;
    this.buffer.fill(null);
  }
}

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
  const buffer = new RingBuffer<LogMatch>(50000);
  let abortController: AbortController | null = null;

  const start = async (request: SearchRequest) => {
    stop();
    buffer.clear();
    logs.value = [];
    stats.value = { total: 0, by_level: {}, total_files: 0 };
    error.value = null;
    isStreaming.value = true;

    abortController = new AbortController();

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
        error.value = `请求失败: ${response.status}`;
        isStreaming.value = false;
        return;
      }

      if (!response.body) {
        error.value = '响应体为空';
        isStreaming.value = false;
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
                buffer.append(match);
                logs.value = buffer.toArray();
                stats.value.total++;
                const level = extractLevel(match.raw);
                if (!stats.value.by_level[level]) {
                  stats.value.by_level[level] = 0;
                }
                stats.value.by_level[level]++;
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

      isStreaming.value = false;
    } catch (err) {
      if (err instanceof Error && err.name !== 'AbortError') {
        error.value = err.message;
      }
      isStreaming.value = false;
    }
  };

  const stop = () => {
    if (abortController) {
      abortController.abort();
      abortController = null;
    }
    isStreaming.value = false;
  };

  const clear = () => {
    buffer.clear();
    logs.value = [];
    stats.value = { total: 0, by_level: {}, total_files: 0 };
    error.value = null;
  };

  return {
    logs,
    stats,
    isStreaming,
    error,
    start,
    stop,
    clear,
  };
}
