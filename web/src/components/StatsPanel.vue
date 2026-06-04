<template>
  <div class="stats-panel">
    <el-card shadow="never" class="stats-card">
      <div class="stats-grid">
        <div class="stat-item main-stat">
          <div class="stat-value">{{ stats.total }}</div>
          <div class="stat-label">总匹配</div>
        </div>

        <div
          v-for="(count, level) in levelStats"
          :key="level"
          class="stat-item"
        >
          <div :class="['stat-value', `stat-${level.toLowerCase()}`]">
            {{ count }}
          </div>
          <div class="stat-label">{{ level }}</div>
        </div>

        <div v-if="stats.total_files" class="stat-item">
          <div class="stat-value stat-other">{{ stats.total_files }}</div>
          <div class="stat-label">扫描文件</div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { LogStats } from '../types';

const props = defineProps<{
  stats: LogStats;
}>();

const levelStats = computed(() => {
  const filtered: Record<string, number> = {};
  const levels = ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'];
  for (const level of levels) {
    if (props.stats.by_level && props.stats.by_level[level]) {
      filtered[level] = props.stats.by_level[level];
    }
  }
  return filtered;
});
</script>

<style scoped>
.stats-panel {
  padding: 0;
}

.stats-card {
  margin: 0;
  border: none;
  box-shadow: none;
  padding: 0;
}

.stats-grid {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  align-items: center;
}

.stat-item {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 6px;
}

.main-stat {
  padding-right: 16px;
  border-right: 1px solid #ebeef5;
  margin-right: 0;
}

.stat-value {
  font-size: 18px;
  font-weight: 600;
  line-height: 1;
  color: #303133;
}

.stat-error {
  color: #f56c6c;
}

.stat-warn {
  color: #e6a23c;
}

.stat-info {
  color: #409eff;
}

.stat-debug {
  color: #67c23a;
}

.stat-trace {
  color: #909399;
}

.stat-other {
  color: #606266;
}

.stat-label {
  font-size: 12px;
  color: #909399;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
</style>
