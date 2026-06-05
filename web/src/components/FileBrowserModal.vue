<template>
  <el-dialog
    v-model="dialogVisible"
    title="📁 服务器文件浏览"
    width="90%"
    max-width="1200px"
    :close-on-click-modal="false"
    :close-on-press-escape="true"
    class="file-browser-modal"
  >
    <!-- 搜索和工具栏 -->
    <div class="modal-toolbar">
      <el-input
        v-model="searchQuery"
        placeholder="搜索文件名..."
        size="default"
        class="search-input"
        clearable
        @input="handleSearch"
      >
        <template #prefix>
          <span>🔍</span>
        </template>
      </el-input>

      <div class="toolbar-actions">
        <el-select
          v-model="sortField"
          size="default"
          :teleported="false"
          class="sort-select"
          @change="handleSortChange"
        >
          <el-option label="按名称排序" value="name" />
          <el-option label="按大小排序" value="size" />
          <el-option label="按修改时间排序" value="mod_time" />
        </el-select>

        <el-select
          v-model="sortOrder"
          size="default"
          :teleported="false"
          class="order-select"
          @change="handleSortChange"
        >
          <el-option label="升序" value="asc" />
          <el-option label="降序" value="desc" />
        </el-select>
      </div>
    </div>

    <!-- 当前路径导航 -->
    <div class="path-nav">
      <el-breadcrumb separator="/">
        <el-breadcrumb-item
          v-for="(item, index) in pathSegments"
          :key="index"
          :class="{ active: index === pathSegments.length - 1 }"
          @click="navigateToPath(pathSegments.slice(0, index + 1).join('/'))"
        >
          <span>{{ item || '根目录' }}</span>
        </el-breadcrumb-item>
      </el-breadcrumb>
    </div>

    <!-- 错误提示 -->
    <el-alert
      v-if="error"
      :title="error"
      type="warning"
      show-icon
      :closable="false"
      class="error-alert"
    />

    <!-- 文件列表 -->
    <div v-loading="loading" class="file-list-wrapper">
      <div v-if="files.length === 0 && !loading && !error" class="empty-state">
        <div class="empty-icon">📂</div>
        <p>该目录为空</p>
      </div>

      <el-table
        v-else-if="files.length > 0"
        :data="filteredFiles"
        stripe
        :header-cell-style="tableHeaderStyle"
        class="files-table"
        row-class-name="file-row"
        @row-click="handleRowClick"
      >
        <el-table-column label="名称" min-width="200">
          <template #default="scope">
            <div class="file-cell">
              <span class="file-icon" @click.stop="handleRowClick(scope.row)">
                {{ scope.row.is_dir ? '📁' : '📄' }}
              </span>
              <span class="file-name" :title="scope.row.name">
                {{ scope.row.name }}
              </span>
              <el-tag
                v-if="scope.row.file_type && !scope.row.is_dir"
                size="small"
                effect="plain"
                type="info"
                class="type-tag"
              >
                {{ scope.row.file_type }}
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="大小" width="120" align="right">
          <template #default="scope">
            <span class="size-cell">{{ scope.row.is_dir ? '-' : scope.row.size_readable }}</span>
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

        <el-table-column label="操作" width="140" align="center">
          <template #default="scope">
            <el-button
              v-if="scope.row.is_dir && scope.row.is_readable"
              link
              type="primary"
              size="small"
              @click.stop="navigateTo(scope.row.full_path)"
            >
              进入
            </el-button>
            <el-button
              v-if="!scope.row.is_dir"
              link
              type="primary"
              size="small"
              @click.stop="handleSelect(scope.row)"
            >
              选择
            </el-button>
            <el-button
              v-if="scope.row.is_dir && scope.row.is_readable"
              link
              type="primary"
              size="small"
              @click.stop="handleSelect(scope.row)"
            >
              选择路径
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 分页 -->
    <div v-if="files.length > 0" class="modal-footer">
      <div class="pagination-info">
        共 {{ pagination.total }} 个文件，当前第 {{ pagination.page }} / {{ pagination.total_pages }} 页
      </div>
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
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onUnmounted } from 'vue';
import type { FileInfo } from '../types';
import { useFileList } from '../composables/useFileList';

const props = defineProps<{
  visible: boolean;
}>();

const emit = defineEmits<{
  'update:visible': [visible: boolean];
  select: [file: FileInfo];
}>();

// 双向绑定 visible prop
const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val),
});

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
const searchQuery = ref('');
const sortField = ref<'name' | 'size' | 'mod_time'>('name');
const sortOrder = ref<'asc' | 'desc'>('asc');

const tableHeaderStyle = {
  background: 'rgba(79, 70, 229, 0.04)',
  color: '#334155',
  fontWeight: '600',
};

const pathSegments = computed(() => {
  if (!currentPath.value) return [];
  const parts = currentPath.value.split('/').filter(p => p);
  return ['', ...parts];
});

const filteredFiles = computed(() => {
  let result = [...files.value];

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    result = result.filter(file =>
      file.name.toLowerCase().includes(query) ||
      file.path.toLowerCase().includes(query)
    );
  }

  result.sort((a, b) => {
    let comparison = 0;
    switch (sortField.value) {
      case 'name':
        comparison = a.name.localeCompare(b.name);
        break;
      case 'size':
        comparison = a.size - b.size;
        break;
      case 'mod_time':
        comparison = new Date(a.mod_time).getTime() - new Date(b.mod_time).getTime();
        break;
    }
    return sortOrder.value === 'asc' ? comparison : -comparison;
  });

  return result;
});

const refreshFiles = () => {
  currentPage.value = 1;
  const pathToFetch = currentPath.value && currentPath.value.trim() !== ''
    ? currentPath.value
    : undefined;
  fetchFiles({
    path: pathToFetch,
    page: 1,
    page_size: currentPageSize.value,
  });
};

const handleSearch = () => {
  currentPage.value = 1;
};

const handleSortChange = () => {
  // Sorting is handled by computed property
};

const handlePageChange = (page: number) => {
  currentPage.value = page;
  goToPage(page, {
    path: currentPath.value || undefined,
    page_size: currentPageSize.value,
  });
};

const navigateTo = (path: string) => {
  if (!path || path.trim() === '') {
    currentPath.value = '/';
  } else {
    currentPath.value = path;
  }
  refreshFiles();
};

const navigateToPath = (path: string) => {
  // 面包屑点击：第一个元素是 ''，代表根目录
  if (!path || path.trim() === '') {
    currentPath.value = '/';
  } else {
    currentPath.value = path;
  }
  refreshFiles();
};

const handleRowClick = (file: FileInfo) => {
  if (file.is_dir && file.is_readable) {
    navigateTo(file.full_path);
  }
};

const handleSelect = (file: FileInfo) => {
  emit('select', file);
  emit('update:visible', false);
};

const handleKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.visible) {
    emit('update:visible', false);
  }
};

watch(() => props.visible, (val) => {
  if (val) {
    refreshFiles();
  }
});

watch(() => pagination.page, (p) => {
  currentPage.value = p;
});

onMounted(() => {
  window.addEventListener('keydown', handleKeydown);
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown);
});
</script>

<style scoped>
.file-browser-modal :deep(.el-dialog__header) {
  background: linear-gradient(135deg, rgba(79, 70, 229, 0.08) 0%, rgba(99, 102, 241, 0.05) 100%);
  border-bottom: 1px solid rgba(99, 102, 241, 0.15);
}

.file-browser-modal :deep(.el-dialog__title) {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}

.modal-toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  align-items: center;
}

.search-input {
  flex: 1;
  min-width: 200px;
}

.toolbar-actions {
  display: flex;
  gap: 8px;
}

.sort-select,
.order-select {
  width: 120px;
}

.path-nav {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 6px;
}

.path-nav :deep(.el-breadcrumb__item) {
  cursor: pointer;
}

.path-nav :deep(.el-breadcrumb__item:not(.active):hover) {
  color: #6366f1;
}

.path-nav :deep(.el-breadcrumb__item.active) {
  color: #4f46e5;
  font-weight: 500;
}

.error-alert {
  margin-bottom: 12px;
}

.file-list-wrapper {
  max-height: 500px;
  overflow-y: auto;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: #94a3b8;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 12px;
}

.files-table {
  border-radius: 6px;
  overflow: hidden;
}

.files-table :deep(.file-row) {
  cursor: pointer;
  transition: background 0.15s ease;
}

.files-table :deep(.file-row:hover) {
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

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  margin-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.pagination-info {
  font-size: 12px;
  color: #94a3b8;
}

@media (max-width: 768px) {
  .modal-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .sort-select,
  .order-select {
    width: 50%;
  }

  .toolbar-actions {
    width: 100%;
  }

  .modal-footer {
    flex-direction: column;
    gap: 8px;
    align-items: flex-start;
  }
}
</style>
