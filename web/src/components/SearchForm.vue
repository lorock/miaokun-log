<template>
  <div class="search-form-container">
    <el-card shadow="never" class="search-card">
      <!-- 主搜索区 -->
      <div class="form-main">
        <el-input
          v-model="form.pattern"
          placeholder="🔍 输入关键词搜索日志 (支持正则表达式)"
          class="search-input"
          size="large"
          @keyup.enter="handleSearch"
          clearable
        />
        
        <!-- 统计信息 -->
        <div class="stats-bar">
          <div class="stat-item">
            <span class="stat-value">{{ stats.total }}</span>
            <span class="stat-label">总匹配</span>
          </div>
          <div v-if="stats.by_level?.ERROR" class="stat-item">
            <span class="stat-value stat-error">{{ stats.by_level.ERROR }}</span>
            <span class="stat-label">ERROR</span>
          </div>
          <div v-if="stats.by_level?.WARN" class="stat-item">
            <span class="stat-value stat-warn">{{ stats.by_level.WARN }}</span>
            <span class="stat-label">WARN</span>
          </div>
          <div v-if="stats.by_level?.INFO" class="stat-item">
            <span class="stat-value stat-info">{{ stats.by_level.INFO }}</span>
            <span class="stat-label">INFO</span>
          </div>
        </div>
        
        <div class="search-buttons">
          <el-button
            type="primary"
            size="large"
            :loading="isStreaming"
            @click="handleSearch"
            :disabled="!form.pattern.trim()"
          >
            <span v-if="!isStreaming">🚀 搜索</span>
            <span v-else>搜索中...</span>
          </el-button>
          <el-button
            v-if="isStreaming"
            size="large"
            type="danger"
            @click="$emit('stop')"
          >
            ⏹ 停止
          </el-button>
        </div>
      </div>

      <!-- 高级筛选区 -->
      <el-collapse v-model="activeFilters" class="filter-collapse">
        <el-collapse-item title="🔧 高级筛选" name="filters">
          <el-form label-position="top" class="filter-form">
            <!-- 第一行：日志级别 + 忽略大小写 + 扫描文件时间范围 -->
            <el-row :gutter="24" class="filter-row">
              <el-col :xs="24" :sm="12" :md="8">
                <el-form-item label="日志级别">
                  <el-select
                    v-model="form.level"
                    placeholder="全部级别"
                    clearable
                    size="default"
                    style="width: 100%"
                    @change="handleSearch"
                  >
                    <el-option label="ERROR" value="ERROR" />
                    <el-option label="WARN" value="WARN" />
                    <el-option label="INFO" value="INFO" />
                    <el-option label="DEBUG" value="DEBUG" />
                    <el-option label="TRACE" value="TRACE" />
                  </el-select>
                </el-form-item>
              </el-col>

              <el-col :xs="24" :sm="12" :md="8">
                <el-form-item label="忽略大小写">
                  <el-switch
                    v-model="form.caseInsensitive"
                    active-text="开启"
                    inactive-text="关闭"
                  />
                </el-form-item>
              </el-col>

              <el-col :xs="24" :sm="12" :md="8">
                <el-form-item label="扫描文件时间范围">
                  <el-select
                    v-model="form.sinceDays"
                    placeholder="选择"
                    size="default"
                    style="width: 100%"
                  >
                    <el-option-group label="分钟级">
                      <el-option label="最近 15 分钟" :value="0.01" />
                      <el-option label="最近 30 分钟" :value="0.02" />
                      <el-option label="最近 1 小时" :value="0.04" />
                    </el-option-group>
                    <el-option-group label="小时级">
                      <el-option label="最近 3 小时" :value="0.125" />
                      <el-option label="最近 6 小时" :value="0.25" />
                      <el-option label="最近 12 小时" :value="0.5" />
                    </el-option-group>
                    <el-option-group label="天级">
                      <el-option label="最近 1 天" :value="1" />
                      <el-option label="最近 3 天" :value="3" />
                      <el-option label="最近 7 天" :value="7" />
                      <el-option label="最近 30 天" :value="30" />
                    </el-option-group>
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <!-- 第二行：日志时间范围 + 上下文行数 -->
            <el-row :gutter="24" class="filter-row">
              <el-col :xs="24" :sm="24" :md="16">
                <el-form-item label="日志时间范围">
                  <el-date-picker
                    v-model="form.timeRange"
                    type="datetimerange"
                    range-separator="至"
                    start-placeholder="开始时间"
                    end-placeholder="结束时间"
                    format="YYYY-MM-DD HH:mm"
                    value-format="YYYY-MM-DD HH:mm"
                    :shortcuts="timeShortcuts"
                    size="default"
                    style="width: 100%"
                  />
                </el-form-item>
              </el-col>

              <el-col :xs="24" :sm="24" :md="8">
                <el-form-item label="上下文行数（前 / 后）">
                  <div class="context-group">
                    <el-input-number
                      v-model="form.before"
                      :min="0"
                      :max="50"
                      size="default"
                      style="flex: 1"
                      placeholder="前"
                    />
                    <span class="context-sep">行</span>
                    <el-input-number
                      v-model="form.after"
                      :min="0"
                      :max="50"
                      size="default"
                      style="flex: 1"
                      placeholder="后"
                    />
                    <span class="context-sep">行</span>
                    <el-switch
                      v-model="form.showContext"
                      active-text="显示"
                      inactive-text="隐藏"
                      :disabled="form.before === 0 && form.after === 0"
                      class="context-switch"
                    />
                  </div>
                </el-form-item>
              </el-col>
            </el-row>

            <!-- 第三行：日志路径 - 占满整行 -->
            <el-row class="filter-row">
              <el-col :span="24">
                <el-form-item>
                  <template #label>
                    <div class="path-label">
                      <span>日志路径</span>
                      <el-tag
                        size="small"
                        type="info"
                        effect="plain"
                        class="tag-hint"
                      >
                        {{ selectedPaths.length > 0 ? '已选 ' + selectedPaths.length + ' 个路径' : '点击选择或输入自定义路径' }}
                      </el-tag>
                      <el-button
                        size="small"
                        type="primary"
                        plain
                        class="browse-btn"
                        @click="showFileBrowser = true"
                      >
                        📁 浏览文件
                      </el-button>
                    </div>
                  </template>
                  <el-select
                    v-model="selectedPaths"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    reserve-keyword
                    placeholder="从服务器选择路径，或输入自定义路径后回车"
                    size="default"
                    style="width: 100%"
                    :loading="pathsLoading"
                  >
                    <el-option-group label="服务器可用路径">
                      <el-option
                        v-for="p in availablePaths"
                        :key="p.path"
                        :label="p.path"
                        :value="p.path"
                      >
                        <span class="path-option">
                          <span class="path-option-name">{{ p.path }}</span>
                          <span class="path-option-meta">
                            <el-tag v-if="p.is_default" size="small" type="primary" effect="plain">默认</el-tag>
                            <el-tag size="small" :type="p.file_count > 0 ? 'success' : 'info'" effect="plain">
                              {{ p.file_count }} 个文件
                            </el-tag>
                            <span class="path-size">{{ formatSize(p.total_size) }}</span>
                          </span>
                        </span>
                      </el-option>
                    </el-option-group>
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
        </el-collapse-item>
      </el-collapse>
    </el-card>

    <!-- 文件浏览模态框 -->
    <FileBrowserModal
      v-model:visible="showFileBrowser"
      @select="handleFileSelect"
    />
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';

import type { LogStats, SearchRequest, FileInfo } from '../types';
import FileBrowserModal from './FileBrowserModal.vue';

defineProps<{
  isStreaming: boolean;
  stats: LogStats;
}>();

const emit = defineEmits<{
  search: [request: SearchRequest];
  stop: [];
}>();

interface PathCandidate {
  path: string;
  exists: boolean;
  file_count: number;
  total_size: number;
  is_default: boolean;
}

const showFileBrowser = ref(false);

const handleFileSelect = (file: FileInfo) => {
  const path = file.is_dir ? file.full_path : file.path;
  if (!selectedPaths.value.includes(path)) {
    selectedPaths.value.push(path);
  }
};

const timeShortcuts = [
  {
    text: '最近 15 分钟',
    value: () => {
      const end = new Date();
      const start = new Date(end.getTime() - 15 * 60 * 1000);
      return [start, end];
    },
  },
  {
    text: '最近 30 分钟',
    value: () => {
      const end = new Date();
      const start = new Date(end.getTime() - 30 * 60 * 1000);
      return [start, end];
    },
  },
  {
    text: '最近 1 小时',
    value: () => {
      const end = new Date();
      const start = new Date(end.getTime() - 60 * 60 * 1000);
      return [start, end];
    },
  },
  {
    text: '最近 3 小时',
    value: () => {
      const end = new Date();
      const start = new Date(end.getTime() - 3 * 60 * 60 * 1000);
      return [start, end];
    },
  },
  {
    text: '最近 6 小时',
    value: () => {
      const end = new Date();
      const start = new Date(end.getTime() - 6 * 60 * 60 * 1000);
      return [start, end];
    },
  },
  {
    text: '最近 12 小时',
    value: () => {
      const end = new Date();
      const start = new Date(end.getTime() - 12 * 60 * 60 * 1000);
      return [start, end];
    },
  },
  {
    text: '今天',
    value: () => {
      const start = new Date();
      start.setHours(0, 0, 0, 0);
      const end = new Date();
      return [start, end];
    },
  },
  {
    text: '昨天',
    value: () => {
      const start = new Date();
      start.setDate(start.getDate() - 1);
      start.setHours(0, 0, 0, 0);
      const end = new Date();
      end.setDate(end.getDate() - 1);
      end.setHours(23, 59, 0, 0);
      return [start, end];
    },
  },
  {
    text: '最近 7 天',
    value: () => {
      const end = new Date();
      const start = new Date();
      start.setDate(start.getDate() - 7);
      return [start, end];
    },
  },
];

const form = reactive({
  pattern: '',
  level: '',
  caseInsensitive: true,
  sinceDays: 3,
  before: 3,
  after: 5,
  showContext: true,
  timeRange: [] as string[],
});

const selectedPaths = ref<string[]>([]);
const availablePaths = ref<PathCandidate[]>([]);
const pathsLoading = ref(false);
const activeFilters = ref<string[]>([]);

const formatSize = (bytes: number) => {
  if (bytes === 0) return '';
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB';
  if (bytes < 1024 * 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
};

const loadPaths = async () => {
  pathsLoading.value = true;
  try {
    const res = await fetch('/api/v1/paths?since=' + form.sinceDays);
    if (res.ok) {
      const data = await res.json() as PathCandidate[];
      availablePaths.value = data.filter(p => p.exists);
      const defaults = data.filter(p => p.is_default && p.exists).map(p => p.path);
      if (selectedPaths.value.length === 0 && defaults.length > 0) {
        selectedPaths.value = defaults;
      }
      if (selectedPaths.value.length === 0 && availablePaths.value.length > 0) {
        const withFiles = availablePaths.value.filter(p => p.file_count > 0);
        if (withFiles.length > 0) {
          selectedPaths.value = [withFiles[0].path];
        }
      }
    } else if (res.status === 401) {
      console.warn('加载路径失败: 登录已过期');
    }
  } catch (e) {
    console.warn('加载路径失败', e);
  } finally {
    pathsLoading.value = false;
  }
};

onMounted(() => {
  loadPaths();
});

const handleSearch = () => {
  if (!form.pattern.trim()) return;

  const request: SearchRequest = {
    pattern: form.pattern.trim(),
    case_insensitive: form.caseInsensitive,
    since_days: form.sinceDays,
    max_count: 50000,
  };

  if (form.level) request.level = form.level;
  
  // 根据是否显示上下文来决定是否传递上下文行数
  if (form.showContext) {
    if (form.before > 0) request.before = form.before;
    if (form.after > 0) request.after = form.after;
  }
  
  if (selectedPaths.value.length > 0) {
    request.paths = [...selectedPaths.value];
  }
  if (form.timeRange.length === 2) {
    request.from = form.timeRange[0];
    request.to = form.timeRange[1];
  }

  emit('search', request);
};
</script>

<style scoped>
.search-form-container {
  padding: 16px 24px;
  flex-shrink: 0;
}

.search-card {
  border-radius: 8px;
}

.form-main {
  display: flex;
  gap: 16px;
  align-items: center;
}

.search-input {
  flex: 1;
  min-width: 200px;
}

/* 统计信息栏 */
.stats-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 8px 16px;
  background: linear-gradient(135deg, rgba(79, 70, 229, 0.08) 0%, rgba(99, 102, 241, 0.05) 100%);
  border: 1px solid rgba(99, 102, 241, 0.15);
  border-radius: 8px;
}

.stat-item {
  display: flex;
  align-items: baseline;
  gap: 4px;
}

.stat-value {
  font-size: 16px;
  font-weight: 600;
  color: #4f46e5;
  min-width: 32px;
}

.stat-error {
  color: #dc2626;
}

.stat-warn {
  color: #d97706;
}

.stat-info {
  color: #2563eb;
}

.stat-label {
  font-size: 12px;
  color: #64748b;
}

.search-buttons {
  display: flex;
  gap: 8px;
}

.filter-collapse {
  margin-top: 12px;
  border: none;
}

.filter-collapse :deep(.el-collapse-item__header) {
  font-weight: 500;
  background: transparent;
  padding-left: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.filter-collapse :deep(.el-collapse-item__wrap) {
  border-bottom: none;
}

.filter-form {
  padding: 12px 0 8px;
}

.filter-row {
  margin-bottom: 0;
}

.filter-row :deep(.el-row) {
  margin-bottom: 0;
}

.filter-row :deep(.el-form-item) {
  margin-bottom: 16px;
}

/* 路径标签样式 */
.path-label {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.browse-btn {
  margin-left: auto;
  flex-shrink: 0;
}

.tag-hint {
  font-weight: normal;
}

/* 上下文行数组合控件 */
.context-group {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.context-sep {
  color: #909399;
  font-size: 13px;
  flex-shrink: 0;
}

/* 路径选项样式 */
.path-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  gap: 12px;
}

.path-option-name {
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.path-option-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.path-size {
  color: #909399;
  font-size: 12px;
}

/* 响应式 - 小屏幕 */
@media (max-width: 768px) {
  .form-main {
    flex-direction: column;
  }

  .form-main .el-button {
    width: 100%;
  }

  .context-group {
    flex-wrap: wrap;
  }
}
</style>
