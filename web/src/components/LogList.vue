<template>
  <div class="log-list-container" ref="containerRef" @scroll="handleScroll">
    <div v-if="logs.length === 0" class="empty-state">
      <div class="empty-icon">🔍</div>
      <p>没有搜索结果</p>
      <p class="empty-hint">请输入关键词开始搜索</p>
    </div>

    <template v-else>
      <!-- 顶部空白占位 -->
      <div :style="{ height: paddingTop + 'px' }"></div>
      
      <!-- 可见内容区域 -->
      <div>
        <div
          v-for="item in visibleItems"
          :key="item.key"
        >
          <div v-if="item.type === 'header'" class="file-header">
            <span class="file-icon">📄</span>
            <span class="file-path" :title="item.file">{{ shortenPath(item.file) }}</span>
            <span class="file-count">{{ item.count }} 条</span>
          </div>

          <template v-else>
            <div
              v-for="(ctx, ci) in item.beforeContext"
              :key="'before-' + item.key + '-' + ci"
              class="log-row context-row"
            >
              <span class="line-num"></span>
              <span class="file-path-inline">{{ getShortFile(item.file) }}</span>
              <span class="level"></span>
              <span class="content" v-html="highlightKeyword(ctx)"></span>
            </div>

            <div class="log-row match-row">
              <span class="line-num">{{ item.globalIndex! + 1 }}</span>
              <span class="file-path-inline">{{ getShortFile(item.file) }}</span>
              <span :class="['level', `level-${item.level}`, 'level-badge']">
                {{ item.level }}
              </span>
              <span class="content" v-html="highlightKeyword(item.raw!)"></span>
            </div>

            <div
              v-for="(ctx, ci) in item.afterContext"
              :key="'after-' + item.key + '-' + ci"
              class="log-row context-row"
            >
              <span class="line-num"></span>
              <span class="file-path-inline">{{ getShortFile(item.file) }}</span>
              <span class="level"></span>
              <span class="content" v-html="highlightKeyword(ctx)"></span>
            </div>
          </template>
        </div>
      </div>

      <!-- 底部空白占位 -->
      <div :style="{ height: paddingBottom + 'px' }"></div>
    </template>

    <!-- 滚动到底部按钮 -->
    <transition name="fade">
      <div
        v-if="showScrollToBottom"
        class="scroll-to-bottom"
        @click="scrollToLatest"
        title="滚动到最新日志"
      >
        <span class="scroll-icon">↓</span>
        <span class="scroll-text">最新</span>
        <span v-if="newLogsCount > 0" class="new-badge">{{ newLogsCount > 99 ? '99+' : newLogsCount }}</span>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue';
import type { LogMatch } from '../types';

const props = defineProps<{
  logs: LogMatch[];
  isStreaming?: boolean;
  searchPattern?: string;
}>();

const MATCH_ROW_HEIGHT = 36;
const CONTEXT_ROW_HEIGHT = 28;
const HEADER_HEIGHT = 40;
const BUFFER = 50;

const LEVELS = ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'];
const extractLevel = (line: string): string => {
  const upper = line.toUpperCase();
  for (const l of LEVELS) {
    if (upper.includes(l)) return l;
  }
  return 'OTHER';
};

interface VirtualItem {
  key: string;
  type: 'header' | 'match';
  file: string;
  count?: number;
  raw?: string;
  level?: string;
  globalIndex?: number;
  beforeContext?: string[];
  afterContext?: string[];
}

const containerRef = ref<HTMLElement | null>(null);
const scrollTop = ref(0);
const containerHeight = ref(600);
const isAtBottom = ref(true);
const newLogsCount = ref(0);
const userScrolled = ref(false);

// 预计算所有虚拟项（响应式）
const computedData = computed(() => {
  const items: VirtualItem[] = [];
  const offsets: number[] = [];
  const heights: number[] = [];
  const fileMap = new Map<string, { count: number; startOffset: number }>();
  let globalIndex = 0;
  let currentOffset = 0;
  let currentFile = '';

  for (const log of props.logs) {
    if (log.file !== currentFile) {
      items.push({
        key: `header-${log.file}-${items.length}`,
        type: 'header',
        file: log.file,
        count: 0,
      });
      offsets.push(currentOffset);
      heights.push(HEADER_HEIGHT);
      currentOffset += HEADER_HEIGHT;
      fileMap.set(log.file, { count: 0, startOffset: items.length - 1 });
      currentFile = log.file;
    }

    const fileInfo = fileMap.get(log.file)!;
    fileInfo.count++;

    const level = extractLevel(log.raw);
    const beforeContext = log.before_context || [];
    const afterContext = log.after_context || [];
    const totalContextRows = beforeContext.length + afterContext.length;
    const itemHeight = MATCH_ROW_HEIGHT + CONTEXT_ROW_HEIGHT * totalContextRows;

    items.push({
      key: `log-${globalIndex}-${log.file.slice(-20)}`,
      type: 'match',
      file: log.file,
      raw: log.raw,
      level,
      globalIndex,
      beforeContext: beforeContext.length > 0 ? beforeContext : undefined,
      afterContext: afterContext.length > 0 ? afterContext : undefined,
    });

    offsets.push(currentOffset);
    heights.push(itemHeight);
    currentOffset += itemHeight;
    globalIndex++;
  }

  fileMap.forEach((info) => {
    const headerItem = items[info.startOffset];
    if (headerItem.type === 'header') {
      headerItem.count = info.count;
    }
  });

  return { allItems: items, itemOffsets: offsets, itemHeights: heights, totalHeight: currentOffset };
});

const allItems = computed(() => computedData.value.allItems);
const itemOffsets = computed(() => computedData.value.itemOffsets);
const totalHeight = computed(() => computedData.value.totalHeight);

// 二分查找起始索引
const findStartIndex = (scroll: number): number => {
  const offsets = itemOffsets.value;
  if (offsets.length === 0) return 0;
  let lo = 0, hi = offsets.length - 1;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (offsets[mid] < scroll) lo = mid + 1;
    else hi = mid;
  }
  return Math.max(0, lo - 5);
};

const startIndex = computed(() => Math.max(0, findStartIndex(scrollTop.value) - BUFFER));
const endIndex = computed(() => {
  const items = allItems.value;
  if (items.length === 0) return 0;
  const visibleEnd = scrollTop.value + containerHeight.value;
  const offsets = itemOffsets.value;
  let lo = startIndex.value, hi = items.length - 1;
  while (lo < hi) {
    const mid = (lo + hi) >> 1;
    if (offsets[mid] < visibleEnd) lo = mid + 1;
    else hi = mid;
  }
  return Math.min(items.length - 1, lo + BUFFER);
});

const visibleItems = computed(() => allItems.value.slice(startIndex.value, endIndex.value + 1));
const paddingTop = computed(() => (startIndex.value > 0 ? itemOffsets.value[startIndex.value] : 0));
const paddingBottom = computed(() => {
  const items = allItems.value;
  if (items.length === 0 || endIndex.value >= items.length - 1) return 0;
  const usedHeight = itemOffsets.value[endIndex.value] + computedData.value.itemHeights[endIndex.value];
  return Math.max(0, totalHeight.value - usedHeight);
});

const handleScroll = () => {
  if (containerRef.value) {
    scrollTop.value = containerRef.value.scrollTop;
    const scrollHeight = containerRef.value.scrollHeight;
    const clientHeight = containerRef.value.clientHeight;
    const scrollBottom = scrollHeight - scrollTop.value - clientHeight;
    isAtBottom.value = scrollBottom < 100;
    if (!isAtBottom.value) {
      userScrolled.value = true;
    }
  }
};

const showScrollToBottom = computed(() => {
  return props.logs.length > 0 && (!isAtBottom.value || newLogsCount.value > 0);
});

const scrollToLatest = () => {
  if (containerRef.value) {
    containerRef.value.scrollTop = containerRef.value.scrollHeight;
    newLogsCount.value = 0;
    userScrolled.value = false;
  }
};

// 日志变化时自动滚动到底部
watch(() => props.logs.length, (newCount, oldCount) => {
  if (newCount > oldCount) {
    const diff = newCount - oldCount;
    if (userScrolled.value) {
      newLogsCount.value += diff;
    } else {
      nextTick(() => {
        scrollToLatest();
      });
    }
  }
});

// 搜索开始时重置
watch(() => props.isStreaming, (streaming) => {
  if (streaming) {
    userScrolled.value = false;
    newLogsCount.value = 0;
    scrollTop.value = 0;
  }
});

const updateContainerHeight = () => {
  if (containerRef.value) {
    containerHeight.value = containerRef.value.clientHeight;
  }
};

const shortenPath = (path: string): string => {
  if (path.length <= 60) return path;
  const parts = path.split('/');
  if (parts.length <= 3) return path;
  return `${parts[0]}/.../${parts.slice(-2).join('/')}`;
};

const getShortFile = (path: string): string => {
  return path.split('/').pop() || path;
};

// 关键词高亮渲染函数
const highlightKeyword = (text: string): string => {
  if (!props.searchPattern || !text) return text;
  
  try {
    // 尝试作为正则表达式匹配
    const regex = new RegExp(props.searchPattern, 'gi');
    return text.replace(regex, (match) => {
      // 对匹配文本进行 HTML 转义，避免 XSS
      const escaped = match
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
      return `<span class="highlight">${escaped}</span>`;
    });
  } catch {
    // 正则表达式无效时，作为普通文本匹配
    const pattern = props.searchPattern.toLowerCase();
    const lowerText = text.toLowerCase();
    let result = '';
    let lastIndex = 0;
    
    let index = lowerText.indexOf(pattern);
    while (index !== -1) {
      // 添加前面的普通文本（需要转义）
      result += text.slice(lastIndex, index)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
      
      // 添加高亮的匹配文本
      const matched = text.slice(index, index + pattern.length);
      result += `<span class="highlight">${matched
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;')}</span>`;
      
      lastIndex = index + pattern.length;
      index = lowerText.indexOf(pattern, lastIndex);
    }
    
    // 添加剩余的普通文本
    result += text.slice(lastIndex)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
    
    return result;
  }
};

let resizeObserver: ResizeObserver | null = null;

onMounted(() => {
  updateContainerHeight();
  if (containerRef.value) {
    resizeObserver = new ResizeObserver(updateContainerHeight);
    resizeObserver.observe(containerRef.value);
  }
});

onUnmounted(() => {
  resizeObserver?.disconnect();
});

defineExpose({ scrollToLatest });
</script>

<style scoped>
.log-list-container {
  height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  background: #1e1e1e;
  position: relative;
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
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.6;
  transition: background-color 0.15s;
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

.content :deep(.highlight) {
  background: linear-gradient(135deg, #ffd700 0%, #ffec8b 100%);
  color: #1a1a1a;
  padding: 1px 4px;
  border-radius: 3px;
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(255, 215, 0, 0.3);
}

.context-row .content {
  color: #8b949e;
}

.context-row .content :deep(.highlight) {
  background: rgba(255, 215, 0, 0.3);
  color: #c9d1d9;
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

.scroll-to-bottom {
  position: absolute;
  bottom: 20px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: linear-gradient(135deg, #4f46e5 0%, #6366f1 100%);
  color: white;
  border-radius: 20px;
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.4);
  font-size: 13px;
  font-weight: 500;
  z-index: 100;
  transition: all 0.2s ease;
}

.scroll-to-bottom:hover {
  transform: translateX(-50%) scale(1.05);
  box-shadow: 0 6px 16px rgba(99, 102, 241, 0.5);
}

.scroll-icon {
  font-size: 16px;
}

.scroll-text {
  font-weight: 600;
}

.new-badge {
  background: #f56c6c;
  color: white;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: 10px;
  min-width: 20px;
  text-align: center;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(10px);
}
</style>