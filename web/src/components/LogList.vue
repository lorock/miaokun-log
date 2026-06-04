<template>
  <div class="log-list-container">
    <div v-if="logs.length === 0" class="empty-state">
      <div class="empty-icon">🔍</div>
      <p>没有搜索结果</p>
      <p class="empty-hint">请输入关键词开始搜索</p>
    </div>

    <div v-else class="log-content">
      <template v-for="(group, gi) in logGroups" :key="gi">
        <div class="file-header">
          <span class="file-icon">📄</span>
          <span class="file-path" :title="group.file">{{ shortenPath(group.file) }}</span>
          <span class="file-count">{{ group.count }} 条</span>
        </div>

        <div v-for="(match, mi) in group.logs" :key="mi">
          <div v-if="match.before_context && match.before_context.length > 0">
            <div
              v-for="(ctx, ci) in match.before_context"
              :key="'before-' + gi + '-' + mi + '-' + ci"
              class="log-row context-row"
            >
              <span class="line-num"></span>
              <span class="file-path-inline">{{ getShortFile(match.file) }}</span>
              <span class="level"></span>
              <span class="content">{{ ctx }}</span>
            </div>
          </div>

          <div class="log-row match-row">
            <span class="line-num">{{ match.globalIndex + 1 }}</span>
            <span class="file-path-inline">{{ getShortFile(match.file) }}</span>
            <span :class="['level', `level-${getLevel(match.raw)}`, 'level-badge']">
              {{ getLevel(match.raw) }}
            </span>
            <span class="content">{{ match.raw }}</span>
          </div>

          <div v-if="match.after_context && match.after_context.length > 0">
            <div
              v-for="(ctx, ci) in match.after_context"
              :key="'after-' + gi + '-' + mi + '-' + ci"
              class="log-row context-row"
            >
              <span class="line-num"></span>
              <span class="file-path-inline">{{ getShortFile(match.file) }}</span>
              <span class="level"></span>
              <span class="content">{{ ctx }}</span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { LogMatch } from '../types';

const props = defineProps<{
  logs: LogMatch[];
}>();

interface FileGroup {
  file: string;
  count: number;
  logs: (LogMatch & { globalIndex: number })[];
}

const logGroups = computed(() => {
  const groups: FileGroup[] = [];
  const fileMap = new Map<string, FileGroup>();
  let globalIndex = 0;

  for (const log of props.logs) {
    if (!fileMap.has(log.file)) {
      fileMap.set(log.file, {
        file: log.file,
        count: 0,
        logs: [],
      });
    }
    const group = fileMap.get(log.file)!;
    group.logs.push({ ...log, globalIndex });
    group.count++;
    globalIndex++;
  }

  fileMap.forEach(g => groups.push(g));
  return groups;
});

const shortenPath = (path: string): string => {
  if (path.length <= 60) return path;
  const parts = path.split('/');
  if (parts.length <= 3) return path;
  return `${parts[0]}/.../${parts.slice(-2).join('/')}`;
};

const getShortFile = (path: string): string => {
  const name = path.split('/').pop() || path;
  return name;
};

const getLevel = (line: string): string => {
  const upper = line.toUpperCase();
  const levels = ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'];
  for (const l of levels) {
    if (upper.includes(l)) {
      return l;
    }
  }
  return 'OTHER';
};
</script>

<style scoped>
.log-list-container {
  height: 100%;
  overflow-y: auto;
  background: #1e1e1e;
  position: relative;
}

.log-content {
  padding-bottom: 20px;
}

.file-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #252525;
  border-bottom: 1px solid #333;
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #8b949e;
  position: sticky;
  top: 0;
  z-index: 10;
}

.file-icon {
  font-size: 13px;
  flex-shrink: 0;
}

.file-path {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #c9d1d9;
}

.file-count {
  flex-shrink: 0;
  background: #30363d;
  color: #8b949e;
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 11px;
}

.log-row {
  display: flex;
  align-items: flex-start;
  padding: 6px 12px;
  border-bottom: 1px solid #2a2a2a;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.6;
  transition: background-color 0.15s;
  min-height: 28px;
}

.log-row:hover {
  background-color: #2a2a2a;
}

.match-row {
  background-color: rgba(64, 158, 255, 0.05);
}

.context-row {
  background-color: #1a1a1a;
  font-style: italic;
  font-size: 12px;
}

.line-num {
  width: 50px;
  text-align: right;
  color: #6e7681;
  padding-right: 12px;
  user-select: none;
  flex-shrink: 0;
  font-size: 12px;
  padding-top: 2px;
}

.context-row .line-num {
  color: #3a3a3a;
}

.file-path-inline {
  width: 120px;
  text-align: left;
  color: #7a8599;
  font-size: 11px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding-right: 12px;
  flex-shrink: 0;
  padding-top: 2px;
}

.level-badge {
  min-width: 52px;
  text-align: center;
  font-weight: bold;
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 3px;
  margin-right: 12px;
  flex-shrink: 0;
  margin-top: 2px;
}

.level-ERROR {
  background-color: rgba(245, 108, 108, 0.15);
  color: #f56c6c;
}

.level-WARN {
  background-color: rgba(230, 162, 60, 0.15);
  color: #e6a23c;
}

.level-INFO {
  background-color: rgba(64, 158, 255, 0.15);
  color: #409eff;
}

.level-DEBUG {
  background-color: rgba(103, 194, 58, 0.15);
  color: #67c23a;
}

.level-TRACE {
  background-color: rgba(144, 147, 153, 0.15);
  color: #909399;
}

.level-OTHER {
  background-color: rgba(144, 147, 153, 0.15);
  color: #909399;
}

.content {
  flex: 1;
  word-break: break-all;
  color: #e6e6e6;
  white-space: pre-wrap;
  padding-top: 2px;
}

.context-row .content {
  color: #8b949e;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #6e7681;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.empty-state p {
  margin: 4px 0;
}

.empty-hint {
  font-size: 14px;
  color: #4a4a4a;
}

::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: #1e1e1e;
}

::-webkit-scrollbar-thumb {
  background: #3a3a3a;
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: #4a4a4a;
}
</style>
