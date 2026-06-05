<template>
  <div class="file-list-container">
    <el-card shadow="never" class="file-card">
      <!-- 标题栏 -->
      <div class="files-header">
        <div class="files-title-row">
          <span class="files-title">📁 服务器文件列表</span>
          <el-tag size="small" effect="plain" type="info">
            共 {{ pagination.total }} 个文件
          </el-tag>
        </div>

        <div class="files-toolbar">
          <el-input
            v-model="currentPath"
            placeholder="路径：例如 /var/log"
            size="default"
            class="path-input"
            clearable
            @keyup.enter="refreshFiles"
          >
            <template #prefix>
              <span class="path-prefix-icon">📍</span>
            </template>
          </el-input>

          <el-select
            v-model="currentPageSize"
            size="default"
            :teleported="false"
            class="page-size-select"
            @change="handlePageSizeChange"
          >
            <el-option label="20 条/页" :value="20" />
            <el-option label="50 条/页" :value="50" />
            <el-option label="100 条/页" :value="100" />
            <el-option label="200 条/页" :value="200" />
          </el-select>

          <el-button
            type="primary"
            size="default"
            :loading="loading"
            @click="refreshFiles"
          >
            🔄 刷新
          </el-button>
        </div>
      </div>

      <!-- 错误提示 -->
      <el-alert
        v-if="error"
        :title="error"
        type="warning"
        show-icon
        :closable="false"
        class="files-alert"
      />

      <!-- 文件列表 -->
      <el-table
        v-loading="loading"
        :data="files"
        stripe
        :header-cell-style="tableHeaderStyle"
        class="files-table"
        empty-text="没有找到文件"
      >
        <el-table-column label="名称" min-width="200">
          <template #default="scope">
            <div class="file-cell">
              <span class="file-icon">{{ scope.row.is_dir ? '📁' : '📄' }}</span>
              <span class="file-name" :title="scope.row.name">{{ scope.row.name }}</span>
              <el-tag
                v-if="scope.row.file_type && !scope.row.is_dir"
                size="small"
                effect="plain"
                type="info"
                class="type-tag"
              >
                {{ scope.row.file_type }}
              </el-tag>
              <el-tag
                v-if="scope.row.is_dir"
                size="small"
                effect="plain"
                type="primary"
                class="type-tag"
              >
                目录
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="路径" min-width="260">
          <template #default="scope">
            <span class="path-cell" :title="scope.row.full_path">
              {{ scope.row.path }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="大小" width="120" align="right">
          <template #default="scope">
            <span class="size-cell">{{ scope.row.size_readable }}</span>
          </template>
        </el-table-column>

        <el-table-column label="修改时间" width="180">
          <template #default="scope">
            <span class="time-cell" :title="scope.row.mod_time">
              {{ scope.row.mod_time_str }}
            </span>
          </template>
        </el-table-column>

        <el-table-column label="状态" width="90" align="center">
          <template #default="scope">
            <el-tag
              size="small"
              :type="scope.row.is_readable ? 'success' : 'info'"
              effect="plain"
            >
              {{ scope.row.is_readable ? '可读' : '受限' }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="120" align="center" fixed="right">
          <template #default="scope">
            <el-button
              link
              type="primary"
              size="small"
              @click="handleSelect(scope.row)"
            >
              选择
            </el-button>
            <el-button
              v-if="scope.row.is_dir && scope.row.is_readable"
              link
              type="primary"
              size="small"
              @click="navigateTo(scope.row.full_path)"
            >
              进入
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div v-if="files.length > 0" class="files-pagination">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="currentPageSize"
          :total="pagination.total"
          :page-count="pagination.total_pages"
          layout="prev, pager, next, jumper"
          :small="true"
          background
          @current-change="handlePageChange"
        />
        <span class="pagination-info">
          第 {{ pagination.page }} / {{ pagination.total_pages || 1 }} 页
        </span>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import type { FileInfo, FileListRequest } from '../types';
import { useFileList } from '../composables/useFileList';

const emit = defineEmits<{
  select: [file: FileInfo];
}>();

const {
  files,
  pagination,
  loading,
  error,
  fetchFiles,
  goToPage,
} = useFileList();

const currentPath = ref('');
const currentPage = ref(1);
const currentPageSize = ref(50);

const tableHeaderStyle = {
  background: 'rgba(79, 70, 229, 0.04)',
  color: '#334155',
  fontWeight: '600',
  borderBottom: '1px solid var(--el-border-color-lighter)',
};

const currentQuery = computed<FileListRequest>(() => ({
  path: currentPath.value || undefined,
  page: currentPage.value,
  page_size: currentPageSize.value,
}));

const refreshFiles = () => {
  currentPage.value = 1;
  fetchFiles(currentQuery.value);
};

const handlePageChange = (page: number) => {
  currentPage.value = page;
  goToPage(page, {
    path: currentPath.value || undefined,
    page_size: currentPageSize.value,
  });
};

const handlePageSizeChange = () => {
  currentPage.value = 1;
  refreshFiles();
};

const handleSelect = (file: FileInfo) => {
  emit('select', file);
};

const navigateTo = (path: string) => {
  currentPath.value = path;
  refreshFiles();
};

onMounted(() => {
  refreshFiles();
});

watch(() => pagination.page, (p) => {
  currentPage.value = p;
});
</script>

<style scoped>
.file-list-container {
  padding: 0 24px 16px;
}

.file-card {
  border-radius: 8px;
}

.files-header {
  margin-bottom: 16px;
}

.files-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.files-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}

.files-toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
}

.path-input {
  flex: 1;
  min-width: 240px;
}

.path-prefix-icon {
  font-size: 14px;
  color: #64748b;
}

.page-size-select {
  width: 120px;
  flex-shrink: 0;
}

.files-alert {
  margin-bottom: 12px;
}

.files-table {
  margin-top: 8px;
  border-radius: 6px;
  overflow: hidden;
}

.files-table :deep(.el-table__row) {
  transition: background 0.15s ease;
}

.files-table :deep(.el-table__row:hover) {
  background: rgba(79, 70, 229, 0.03);
}

.file-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
}

.file-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.file-name {
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #334155;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.type-tag {
  margin-left: 6px;
  flex-shrink: 0;
}

.path-cell {
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}

.size-cell {
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #475569;
}

.time-cell {
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #64748b;
}

.files-pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.pagination-info {
  font-size: 12px;
  color: #94a3b8;
}

@media (max-width: 768px) {
  .files-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .path-input,
  .page-size-select {
    width: 100%;
  }

  .files-pagination {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }
}
</style>
