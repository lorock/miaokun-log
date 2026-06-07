<template>
  <div class="log-list-root">
    <!-- 顶部摘要工具栏（固定在日志区域顶部，不可滚动） -->
    <div v-if="logs.length > 0" class="summary-toolbar">
      <!-- 左侧：统计信息 -->
      <div class="summary-left">
        <span class="stat-item">{{ logs.length }} 条</span>
        <template v-if="searchDurationMs != null">
          <span class="stat-divider">|</span>
          <span class="stat-item">耗时 {{ searchDurationMs }} ms</span>
        </template>
        <span class="stat-divider">|</span>
        <span class="stat-item">{{ fileCount }} 文件</span>
        <span v-if="isStreaming && progress && progress.totalFiles > 0" class="progress-badge">
          扫描中 {{ progress.currentFile }}/{{ progress.totalFiles }}
        </span>
      </div>

      <!-- 右侧：操作按钮组 -->
      <div class="summary-right">
        <!-- 在结果中搜索 -->
        <div class="find-in-results">
          <input
            v-model="findKeyword"
            type="text"
            class="find-input"
            placeholder="在结果中搜索..."
            @keydown.enter="goToNextFindMatch"
          />
          <button class="find-btn" @click="goToPrevFindMatch" :disabled="findMatches.length === 0" title="上一个">▲</button>
          <button class="find-btn" @click="goToNextFindMatch" :disabled="findMatches.length === 0" title="下一个">▼</button>
          <span class="find-count" v-if="findKeyword">
            {{ findMatches.length > 0 ? `${currentFindIndex + 1} / ${findMatches.length}` : '0 / 0' }}
          </span>
        </div>

        <!-- 导出下拉菜单 -->
        <el-dropdown @command="handleExport" trigger="click">
          <button class="toolbar-btn" title="导出">⬇ 导出</button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="txt">📄 导出 TXT</el-dropdown-item>
              <el-dropdown-item command="json">🗂 导出 JSON</el-dropdown-item>
              <el-dropdown-item command="csv">📊 导出 CSV</el-dropdown-item>
              <el-dropdown-item command="md">📑 导出 Markdown</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <!-- 复制全部 -->
        <button class="toolbar-btn" @click="copyAllLogs" title="复制全部日志">📋 复制全部</button>

        <!-- 时间跳转区域 -->
        <div class="time-jump-area">
          <el-date-picker
            v-model="jumpTimestamp"
            type="datetime"
            placeholder="选择时间..."
            format="YYYY-MM-DD HH:mm:ss"
            value-format="YYYY-MM-DD HH:mm:ss"
            size="small"
            class="jump-datetime-picker"
            @change="jumpToTimestamp"
          />
          <button class="jump-quick-btn" @click="jumpToFirst" title="跳到首条">⏮</button>
          <button class="jump-quick-btn" @click="jumpBackward10min" title="向前10分">◀-10分</button>
          <button class="jump-quick-btn" @click="jumpForward10min" title="向后10分">+10分▶</button>
          <button class="jump-quick-btn" @click="jumpToLast" title="跳到末条">⏭</button>
        </div>

        <!-- 设置按钮 -->
        <button
          :class="['toolbar-btn', 'settings-btn', { active: showSettings }]"
          @click="showSettings = !showSettings"
          title="设置"
        >⚙ 设置</button>
      </div>

      <!-- 设置面板 -->
      <div v-if="showSettings" class="settings-panel">
        <div class="settings-row">
          <label class="settings-label">
            <input type="checkbox" v-model="showContext" />
            <span>显示上下文行</span>
          </label>
          <label class="settings-label">
            <input type="checkbox" v-model="formatJson" />
            <span>JSON 美化</span>
          </label>
        </div>
      </div>
    </div>

    <!-- 三种空状态 -->
    <!-- 正在搜索 -->
    <div v-if="isStreaming && logs.length === 0" class="empty-state">
      <div class="empty-icon search-spinner">🔍</div>
      <p>正在搜索日志...</p>
      <p class="empty-hint">扫描中，请稍候</p>
    </div>
    <!-- 搜索过但无结果 -->
    <div v-else-if="!isStreaming && logs.length === 0 && searchPattern" class="empty-state">
      <div class="empty-icon">📭</div>
      <p>未找到匹配的日志</p>
      <p class="empty-hint">试试其他关键词或调整筛选条件</p>
    </div>
    <!-- 还没有搜索过 -->
    <div v-else-if="logs.length === 0" class="empty-state">
      <div class="empty-icon">💡</div>
      <p>请开始搜索</p>
      <p class="empty-hint">在上方输入关键词并点击"搜索"</p>
    </div>

    <!-- 可滚动的日志区域 -->
    <div
      v-else
      class="log-list-container"
      ref="containerRef"
      @scroll="handleScroll"
    >
      <div class="virtual-spacer" :style="{ height: totalHeight + 'px' }">
        <div
          class="virtual-inner"
          :style="{ transform: 'translateY(' + paddingTop + 'px)' }"
        >
          <template v-for="item in visibleItems" :key="item.key">
            <!-- 文件分组标题（可折叠） -->
            <div
              v-if="item.type === 'header'"
              class="file-header"
              :style="{ height: HEADER_HEIGHT + 'px' }"
              @click="toggleFileCollapse(item.file)"
            >
              <span class="file-header-arrow">{{ fileCollapsed.has(item.file) ? '▶' : '▼' }}</span>
              <span class="file-icon">📄</span>
              <span class="file-path" :title="item.file">{{ shortenPath(item.file) }}</span>
              <span class="file-count">{{ item.count }} 条</span>
            </div>

            <!-- 匹配日志行 / 上下文行 -->
            <template v-else-if="!fileCollapsed.has(item.file)">
              <template v-for="(ctx, ci) in (showContext ? (item.beforeContext || []) : [])" :key="'before-' + item.key + '-' + ci">
                <div class="log-row context-row" :style="{ height: CONTEXT_ROW_HEIGHT + 'px' }">
                  <span class="line-num"></span>
                  <span class="file-path-inline">{{ getShortFile(item.file) }}</span>
                  <span class="level"></span>
                  <span class="content" v-html="highlightKeyword(ctx)"></span>
                </div>
              </template>

              <div
                :class="['log-row', 'match-row', { highlighted: currentFindIndex >= 0 && findMatches[currentFindIndex]?.globalIndex === item.globalIndex }]"
                :style="{ height: (item.virtualHeight || MATCH_ROW_HEIGHT) + 'px', minHeight: MIN_MATCH_ROW_HEIGHT + 'px' }"
              >
                <span class="line-num">{{ item.globalIndex !== undefined ? item.globalIndex + 1 : '' }}</span>
                <span class="file-path-inline">{{ getShortFile(item.file) }}</span>
                <span :class="['level', 'level-' + item.level, 'level-badge']">{{ item.level }}</span>
                <span
                  v-if="extractTimestamp(item.raw)"
                  class="timestamp clickable"
                  @click.stop="jumpToLogTime(item.raw!)"
                  title="点击以此时间为基准跳转"
                >{{ extractTimestamp(item.raw) }}</span>
                <div class="content-wrapper">
                  <template v-if="expandedLogs.has(item.globalIndex!) || !isLongLog(item.raw)">
                    <span class="content" v-html="renderContent(item.raw)"></span>
                  </template>
                  <template v-else>
                    <span class="content" v-html="renderContent(truncateLog(item.raw))"></span>
                    <span class="expand-btn" @click.stop="toggleExpanded(item.globalIndex!)">[展开]</span>
                  </template>
                  <span
                    v-if="expandedLogs.has(item.globalIndex!) && isLongLog(item.raw)"
                    class="expand-btn"
                    @click.stop="toggleExpanded(item.globalIndex!)"
                  >[收起]</span>
                  <span class="copy-single-btn" @click.stop="copySingleLog(item.raw)" title="复制此条">📋</span>
                </div>
              </div>

              <template v-for="(ctx, ci) in (showContext ? (item.afterContext || []) : [])" :key="'after-' + item.key + '-' + ci">
                <div class="log-row context-row" :style="{ height: CONTEXT_ROW_HEIGHT + 'px' }">
                  <span class="line-num"></span>
                  <span class="file-path-inline">{{ getShortFile(item.file) }}</span>
                  <span class="level"></span>
                  <span class="content" v-html="highlightKeyword(ctx)"></span>
                </div>
              </template>
            </template>
          </template>
        </div>
      </div>
    </div>

    <!-- 滚动到最新按钮 -->
    <transition name="fade">
      <div
        v-if="logs.length > 0 && showScrollToBottom"
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
import { ElMessage } from 'element-plus';
import type { LogMatch } from '../types';

const props = withDefaults(defineProps<{
  logs: LogMatch[];
  isStreaming?: boolean;
  searchPattern?: string;
  searchDurationMs?: number;
  progress?: { currentFile: number; totalFiles: number; currentFileName: string };
}>(), {
  isStreaming: false,
  searchPattern: '',
});

const MATCH_ROW_HEIGHT = 36;
const CONTEXT_ROW_HEIGHT = 28;
const HEADER_HEIGHT = 40;
const BUFFER = 50;
const LONG_LOG_THRESHOLD = 500;

// 动态行高计算相关常量
const CHARS_PER_LINE = 150;        // 估算每行可容纳字符数
const EXTRA_LINE_HEIGHT = 18;       // 每行额外高度（对应 font-size ~13px + line-height）
const MIN_MATCH_ROW_HEIGHT = MATCH_ROW_HEIGHT;  // 最小行高

const LEVELS: string[] = ['ERROR', 'WARN', 'INFO', 'DEBUG', 'TRACE'];
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
  virtualHeight?: number;  // 动态计算的行高
}

// 状态
const containerRef = ref<HTMLElement | null>(null);
const scrollTop = ref(0);
const containerHeight = ref(600);
const isAtBottom = ref(true);
const newLogsCount = ref(0);
const userScrolled = ref(false);
const jumpTimestamp = ref<string>('');
const showSettings = ref(false);
const showContext = ref(true);
const formatJson = ref(false);
const fileCollapsed = ref<Set<string>>(new Set());
const expandedLogs = ref<Set<number>>(new Set());
const findKeyword = ref('');
const currentFindIndex = ref(-1);

// ===== 动态行高计算 =====
const calcMatchRowHeight = (raw: string | undefined, globalIdx: number | undefined): number => {
  if (!raw) return MIN_MATCH_ROW_HEIGHT;
  const len = raw.length;
  // 超过阈值的长日志：根据展开状态计算
  if (len > LONG_LOG_THRESHOLD) {
    const isExpanded = globalIdx !== undefined && expandedLogs.value.has(globalIdx);
    if (!isExpanded) {
      // 折叠状态：展示固定行高（截断到阈值 + [展开] 按钮）
      return MIN_MATCH_ROW_HEIGHT;
    }
    // 展开状态：按实际内容估算行数
    const estLines = Math.max(1, Math.ceil(len / CHARS_PER_LINE));
    return Math.max(MIN_MATCH_ROW_HEIGHT, MIN_MATCH_ROW_HEIGHT + (estLines - 1) * EXTRA_LINE_HEIGHT);
  }
  // 短日志：按实际字符数估算（避免非常窄容器下也能正确换行）
  const estLines = Math.max(1, Math.ceil(len / CHARS_PER_LINE));
  if (estLines <= 1) return MIN_MATCH_ROW_HEIGHT;
  return Math.max(MIN_MATCH_ROW_HEIGHT, MIN_MATCH_ROW_HEIGHT + (estLines - 1) * EXTRA_LINE_HEIGHT);
};

// 文件数
const fileCount = computed(() => {
  const files = new Set<string>();
  for (const l of props.logs) files.add(l.file);
  return files.size;
});

// 时间戳提取/解析
const TS_REGEX = /(\d{4}[-/]\d{2}[-/]\d{2}[ T]\d{2}:\d{2}:\d{2}(?:[.,]\d{1,6})?(?:Z|[+-]\d{2}:?\d{2})?)/;
const extractTimestamp = (text: string | undefined): string => {
  if (!text) return '';
  const m = text.match(TS_REGEX);
  return m ? m[1] : '';
};

const parseTimestampFromLog = (ts: string): Date | null => {
  const patterns = [/^(\d{4})[-/](\d{2})[-/](\d{2})[\sT](\d{2}):(\d{2}):(\d{2})/];
  for (const pattern of patterns) {
    const match = ts.match(pattern);
    if (match) {
      const [, year, month, day, hour, minute, second] = match;
      return new Date(
        parseInt(year), parseInt(month) - 1, parseInt(day),
        parseInt(hour), parseInt(minute), parseInt(second),
      );
    }
  }
  return null;
};

// 虚拟滚动数据
const computedData = computed(() => {
  const items: VirtualItem[] = [];
  const offsets: number[] = [];
  let currentOffset = 0;
  const fileMap = new Map<string, { count: number; startOffset: number }>();
  let globalIndex = 0;
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
      currentOffset += HEADER_HEIGHT;
      fileMap.set(log.file, { count: 0, startOffset: items.length - 1 });
      currentFile = log.file;
    }

    const fileInfo = fileMap.get(log.file)!;
    fileInfo.count++;

    const level = extractLevel(log.raw);
    const beforeContext = log.before_context || [];
    const afterContext = log.after_context || [];
    const totalContextRows = showContext.value ? beforeContext.length + afterContext.length : 0;
    // 动态计算 match 行的高度（根据内容长度）
    const matchHeight = calcMatchRowHeight(log.raw, globalIndex);
    const itemHeight = matchHeight + CONTEXT_ROW_HEIGHT * totalContextRows;

    items.push({
      key: `log-${globalIndex}-${log.file.slice(-20)}`,
      type: 'match',
      file: log.file,
      raw: log.raw,
      level,
      globalIndex,
      beforeContext: beforeContext.length > 0 ? beforeContext : undefined,
      afterContext: afterContext.length > 0 ? afterContext : undefined,
      virtualHeight: matchHeight,
    });

    offsets.push(currentOffset);
    currentOffset += itemHeight;
    globalIndex++;
  }

  fileMap.forEach((info) => {
    const headerItem = items[info.startOffset];
    if (headerItem.type === 'header') {
      headerItem.count = info.count;
    }
  });

  return { allItems: items, itemOffsets: offsets, totalHeight: currentOffset };
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

// 文件折叠切换
const toggleFileCollapse = (file: string) => {
  const next = new Set(fileCollapsed.value);
  if (next.has(file)) next.delete(file);
  else next.add(file);
  fileCollapsed.value = next;
};

// 长日志判断/截断/展开切换
const isLongLog = (raw: string | undefined): boolean => {
  return !!raw && raw.length > LONG_LOG_THRESHOLD;
};

const truncateLog = (raw: string | undefined): string => {
  if (!raw) return '';
  return raw.slice(0, LONG_LOG_THRESHOLD);
};

const toggleExpanded = (globalIndex: number) => {
  const next = new Set(expandedLogs.value);
  if (next.has(globalIndex)) next.delete(globalIndex);
  else next.add(globalIndex);
  expandedLogs.value = next;
};

// 内容渲染（JSON美化 + 关键词高亮 + find高亮）
const escapeHtml = (text: string): string => {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
};

const tryFormatJson = (text: string): string => {
  if (!formatJson.value) return text;
  const trimmed = text.trim();
  if (trimmed.length < 2) return text;
  const first = trimmed[0];
  if (first !== '{' && first !== '[') return text;
  try {
    const obj = JSON.parse(trimmed);
    return JSON.stringify(obj, null, 2);
  } catch {
    return text;
  }
};

const highlightKeyword = (text: string | undefined): string => {
  if (!text) return '';
  let toRender = tryFormatJson(text);

  if (!props.searchPattern) {
    return applyFindHighlight(escapeHtml(toRender));
  }

  try {
    const regex = new RegExp(props.searchPattern, 'gi');
    const html = toRender.replace(regex, (match) => {
      return `<span class="highlight">${escapeHtml(match)}</span>`;
    });
    // 如果没发生替换，则说明没有匹配，但我们仍需对原始内容做转义
    const finalHtml = html === toRender ? escapeHtml(toRender) : html;
    return applyFindHighlight(finalHtml);
  } catch {
    const pattern = props.searchPattern.toLowerCase();
    const lowerText = toRender.toLowerCase();
    let result = '';
    let lastIndex = 0;
    let index = lowerText.indexOf(pattern);
    while (index !== -1) {
      result += escapeHtml(toRender.slice(lastIndex, index));
      const matched = toRender.slice(index, index + pattern.length);
      result += `<span class="highlight">${escapeHtml(matched)}</span>`;
      lastIndex = index + pattern.length;
      index = lowerText.indexOf(pattern, lastIndex);
    }
    result += escapeHtml(toRender.slice(lastIndex));
    return applyFindHighlight(result);
  }
};

// "在结果中搜索" 匹配集合
const findMatches = computed(() => {
  const kw = findKeyword.value.trim().toLowerCase();
  if (!kw) return [] as { globalIndex: number; file: string }[];
  const results: { globalIndex: number; file: string }[] = [];
  for (let i = 0; i < props.logs.length; i++) {
    if (props.logs[i].raw.toLowerCase().includes(kw)) {
      results.push({ globalIndex: i, file: props.logs[i].file });
    }
  }
  return results;
});

watch(findKeyword, (val) => {
  if (!val.trim()) {
    currentFindIndex.value = -1;
    return;
  }
  currentFindIndex.value = findMatches.value.length > 0 ? 0 : -1;
  if (currentFindIndex.value >= 0) {
    nextTick(() => scrollToLogIndex(findMatches.value[currentFindIndex.value].globalIndex, false));
  }
});

const applyFindHighlight = (html: string): string => {
  const kw = findKeyword.value.trim();
  if (!kw) return html;
  try {
    const re = new RegExp(kw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'gi');
    return html.replace(re, (m) => `<span class="find-highlight">${m}</span>`);
  } catch {
    return html;
  }
};

const goToNextFindMatch = () => {
  if (findMatches.value.length === 0) return;
  currentFindIndex.value = (currentFindIndex.value + 1) % findMatches.value.length;
  scrollToLogIndex(findMatches.value[currentFindIndex.value].globalIndex, true);
};

const goToPrevFindMatch = () => {
  if (findMatches.value.length === 0) return;
  const n = findMatches.value.length;
  currentFindIndex.value = (currentFindIndex.value <= 0 ? n : currentFindIndex.value) - 1;
  scrollToLogIndex(findMatches.value[currentFindIndex.value].globalIndex, true);
};

const renderContent = (raw: string | undefined): string => {
  return highlightKeyword(raw || '');
};

// 滚动到指定 globalIndex 的日志
const scrollToLogIndex = (globalIndex: number, highlight: boolean) => {
  nextTick(() => {
    if (!containerRef.value) return;
    // 找到该 globalIndex 对应的 allItems 索引
    const items = allItems.value;
    let idx = -1;
    for (let i = 0; i < items.length; i++) {
      if (items[i].type === 'match' && items[i].globalIndex === globalIndex) {
        idx = i;
        break;
      }
    }
    if (idx < 0) return;
    const offset = itemOffsets.value[idx] || 0;
    containerRef.value.scrollTop = Math.max(0, offset - 50);
    scrollTop.value = containerRef.value.scrollTop;
    if (highlight) {
      // 临时高亮（已通过 match-row.highlighted CSS 实现）
    }
  });
};

const scrollToIndex = (index: number, _msg: string) => {
  scrollToLogIndex(index, true);
};

const scrollToNearestLog = (targetDate: Date, msg: string) => {
  let closestIndex = -1;
  let minDiff = Infinity;
  for (let i = 0; i < props.logs.length; i++) {
    const ts = extractTimestamp(props.logs[i].raw);
    if (!ts) continue;
    const logDate = parseTimestampFromLog(ts);
    if (!logDate) continue;
    const diff = Math.abs(logDate.getTime() - targetDate.getTime());
    if (diff < minDiff) {
      minDiff = diff;
      closestIndex = i;
    }
  }
  if (closestIndex !== -1) {
    scrollToIndex(closestIndex, msg);
    ElMessage.success(msg);
  } else {
    ElMessage.info('未找到可解析的时间戳');
  }
};

const jumpToTimestamp = (val?: string) => {
  const targetTs = (val ?? jumpTimestamp.value ?? '').trim();
  if (!targetTs) return;
  const targetDate = new Date(targetTs.replace(' ', 'T'));
  if (isNaN(targetDate.getTime())) {
    ElMessage.warning('时间格式无效');
    return;
  }
  scrollToNearestLog(targetDate, '已跳转到 ' + targetTs);
};

const jumpToLogTime = (raw: string) => {
  const ts = extractTimestamp(raw);
  if (!ts) return;
  const targetDate = parseTimestampFromLog(ts);
  if (!targetDate) return;
  jumpTimestamp.value = ts;
  scrollToNearestLog(targetDate, '已跳转到 ' + ts);
};

const jumpToFirst = () => {
  if (props.logs.length === 0) return;
  scrollToIndex(0, '已跳转到首条日志');
  ElMessage.success('已跳转到首条日志');
};

const jumpToLast = () => {
  if (props.logs.length === 0) return;
  scrollToIndex(props.logs.length - 1, '已跳转到末条日志');
  ElMessage.success('已跳转到末条日志');
};

const jumpBackward10min = () => jumpByMinutes(-10);
const jumpForward10min = () => jumpByMinutes(10);

const jumpByMinutes = (minutes: number) => {
  if (props.logs.length === 0) return;
  let baseTime: Date | null = null;
  if (jumpTimestamp.value) {
    baseTime = new Date(String(jumpTimestamp.value).replace(' ', 'T'));
    if (isNaN(baseTime.getTime())) baseTime = null;
  }
  if (!baseTime) {
    const firstTs = extractTimestamp(props.logs[0].raw);
    if (firstTs) baseTime = parseTimestampFromLog(firstTs);
  }
  if (!baseTime) {
    ElMessage.info('无法解析日志时间');
    return;
  }
  const targetDate = new Date(baseTime.getTime() + minutes * 60 * 1000);
  const dirStr = minutes > 0 ? '向后 ' + minutes + ' 分钟' : '向前 ' + (-minutes) + ' 分钟';
  scrollToNearestLog(targetDate, '已' + dirStr);
};

// 导出功能
const handleExport = (format: string) => {
  if (props.logs.length === 0) {
    ElMessage.info('没有可导出的日志');
    return;
  }
  let content = '';
  let mime = 'text/plain;charset=utf-8';
  let filename = `logs_${new Date().toISOString().replace(/[:.]/g, '-')}`;

  if (format === 'txt') {
    content = props.logs.map((l) => l.raw).join('\n');
    filename += '.txt';
  } else if (format === 'json') {
    const payload = {
      pattern: props.searchPattern || '',
      total: props.logs.length,
      timestamp: new Date().toISOString(),
      duration_ms: props.searchDurationMs ?? null,
      matches: props.logs,
    };
    content = JSON.stringify(payload, null, 2);
    mime = 'application/json;charset=utf-8';
    filename += '.json';
  } else if (format === 'csv') {
    const header = 'file,line_num,raw';
    const rows = props.logs.map((l) => {
      const escRaw = '"' + (l.raw || '').replace(/"/g, '""') + '"';
      return `"${l.file.replace(/"/g, '""')}",${l.line_num},${escRaw}`;
    });
    content = [header, ...rows].join('\n');
    mime = 'text/csv;charset=utf-8';
    filename += '.csv';
  } else if (format === 'md') {
    const headerLines = [
      '# 日志搜索结果',
      '',
      '- **搜索关键词**: ' + (props.searchPattern || '(空)'),
      '- **匹配数量**: ' + props.logs.length,
      '- **搜索耗时**: ' + (props.searchDurationMs ?? '?') + ' ms',
      '- **导出时间**: ' + new Date().toLocaleString('zh-CN'),
      '',
      '---',
      '',
    ];
    const bodyLines = props.logs.map((l, i) => {
      return [
        '## ' + (i + 1) + '. ' + (l.file || '未知文件') + (l.line_num !== undefined ? ' @行 ' + l.line_num : ''),
        '',
        '```',
        l.raw || '',
        '```',
        '',
      ].join('\n');
    });
    content = headerLines.join('\n') + bodyLines.join('\n');
    filename += '.md';
    mime = 'text/markdown;charset=utf-8';
  } else {
    ElMessage.warning('未知导出格式');
    return;
  }

  try {
    const blob = new Blob(['\uFEFF' + content], { type: mime });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    ElMessage.success('导出成功：' + filename);
  } catch (e) {
    ElMessage.error('导出失败');
  }
};

// 复制功能
const copySingleLog = async (raw: string | undefined) => {
  if (!raw) return;
  try {
    await navigator.clipboard.writeText(raw);
    ElMessage.success('已复制');
  } catch (e) {
    const ta = document.createElement('textarea');
    ta.value = raw;
    document.body.appendChild(ta);
    ta.select();
    document.execCommand('copy');
    document.body.removeChild(ta);
    ElMessage.success('已复制');
  }
};

const copyAllLogs = async () => {
  if (props.logs.length === 0) {
    ElMessage.info('没有可复制的日志');
    return;
  }
  const text = props.logs.map((l) => l.raw).join('\n');
  try {
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text);
    } else {
      const ta = document.createElement('textarea');
      ta.value = text;
      ta.style.position = 'fixed';
      ta.style.left = '-9999px';
      document.body.appendChild(ta);
      ta.select();
      document.execCommand('copy');
      document.body.removeChild(ta);
    }
    ElMessage.success('已复制 ' + props.logs.length + ' 条日志到剪贴板');
  } catch {
    ElMessage.error('复制失败');
  }
};

// 滚动处理
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
    scrollTop.value = containerRef.value.scrollTop;
    newLogsCount.value = 0;
    userScrolled.value = false;
  }
};

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

watch(() => props.isStreaming, (streaming) => {
  if (streaming) {
    userScrolled.value = false;
    newLogsCount.value = 0;
    scrollTop.value = 0;
    fileCollapsed.value = new Set();
    expandedLogs.value = new Set();
    findKeyword.value = '';
    currentFindIndex.value = -1;
  }
});

// ===== 键盘快捷键 =====
const handleKeyDown = (e: KeyboardEvent) => {
  // Ctrl/Cmd + F: 聚焦到"在结果中搜索"输入框
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'f') {
    const findInput = document.querySelector<HTMLInputElement>('.find-input');
    if (findInput) {
      e.preventDefault();
      findInput.focus();
      findInput.select();
    }
  }
  // Ctrl/Cmd + G: 下一个搜索结果 (Shift + Ctrl/Cmd + G: 上一个)
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'g') {
    e.preventDefault();
    if (e.shiftKey) {
      goToPrevFindMatch();
    } else {
      goToNextFindMatch();
    }
  }
  // Escape: 清空"在结果中搜索"输入框
  if (e.key === 'Escape' && findKeyword.value) {
    findKeyword.value = '';
  }
};

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

let resizeObserver: ResizeObserver | null = null;

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown);
  updateContainerHeight();
  if (containerRef.value) {
    resizeObserver = new ResizeObserver(updateContainerHeight);
    resizeObserver.observe(containerRef.value);
  }
});

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown);
  resizeObserver?.disconnect();
});

defineExpose({ scrollToLatest });
</script>

<style scoped>
.log-list-root {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  position: relative;
  overflow: hidden;
}

/* 顶部摘要工具栏 */
.summary-toolbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  background: #252525;
  border-bottom: 1px solid #3a3a3a;
  flex-shrink: 0;
  flex-wrap: wrap;
}

.summary-left {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #d4d4d4;
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  flex-shrink: 0;
}

.stat-item {
  color: #b0b0b0;
}

.stat-divider {
  color: #4a4a4a;
}

.summary-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.find-in-results {
  display: flex;
  align-items: center;
  gap: 2px;
  background: #1e1e1e;
  border: 1px solid #3a3a3a;
  border-radius: 4px;
  padding: 2px 4px;
}

.find-input {
  background: transparent;
  border: none;
  outline: none;
  color: #d4d4d4;
  font-size: 12px;
  padding: 4px 6px;
  width: 160px;
  font-family: 'Consolas', 'Monaco', monospace;
}

.find-input::placeholder {
  color: #6e7681;
}

.find-btn {
  background: #2d2d2d;
  border: 1px solid #3a3a3a;
  color: #b0b0b0;
  padding: 4px 8px;
  border-radius: 3px;
  cursor: pointer;
  font-size: 11px;
  transition: all 0.15s;
}

.find-btn:hover:not(:disabled) {
  background: #404040;
  color: #fff;
}

.find-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.find-count {
  color: #7a8599;
  font-size: 11px;
  padding: 0 6px;
  font-family: 'Consolas', 'Monaco', monospace;
}

.toolbar-btn {
  background: #2d2d2d;
  border: 1px solid #3a3a3a;
  color: #b0b0b0;
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.15s;
  white-space: nowrap;
}

.toolbar-btn:hover {
  background: #404040;
  color: #fff;
}

.toolbar-btn.active {
  background: #4f46e5;
  border-color: #6366f1;
  color: #fff;
}

.settings-btn.active:hover {
  background: #6366f1;
}

.progress-badge {
  display: inline-block;
  padding: 2px 10px;
  background: #2d4a3e;
  color: #4ade80;
  border-radius: 12px;
  font-size: 11px;
  margin-left: 8px;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.copy-single-btn {
  display: inline-block;
  padding: 2px 8px;
  background: #2d2d2d;
  border: 1px solid #3a3a3a;
  border-radius: 4px;
  font-size: 11px;
  color: #888;
  cursor: pointer;
  margin-left: 6px;
  opacity: 0;
  transition: all 0.15s;
  flex-shrink: 0;
}

.match-row:hover .copy-single-btn {
  opacity: 1;
}

.copy-single-btn:hover {
  background: #404040;
  color: #fff;
  border-color: #6366f1;
}

.time-jump-area {
  display: flex;
  align-items: center;
  gap: 4px;
}

.jump-datetime-picker {
  width: 200px;
}

.jump-datetime-picker :deep(.el-input__wrapper) {
  background: #1e1e1e;
  box-shadow: 0 0 0 1px #3a3a3a inset;
}

.jump-quick-btn {
  background: #2d2d2d;
  border: 1px solid #3a3a3a;
  color: #b0b0b0;
  padding: 5px 10px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 11px;
  transition: all 0.15s;
  white-space: nowrap;
  font-family: 'Consolas', 'Monaco', monospace;
}

.jump-quick-btn:hover {
  background: #404040;
  color: #fff;
}

/* 设置面板 */
.settings-panel {
  width: 100%;
  margin-top: 8px;
  padding: 8px 12px;
  background: #1e1e1e;
  border: 1px solid #3a3a3a;
  border-radius: 4px;
}

.settings-row {
  display: flex;
  gap: 24px;
  align-items: center;
}

.settings-label {
  display: flex;
  align-items: center;
  gap: 6px;
  color: #d4d4d4;
  font-size: 12px;
  cursor: pointer;
}

.settings-label input[type="checkbox"] {
  cursor: pointer;
}

/* 可滚动的日志容器 */
.log-list-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  background: #1e1e1e;
  min-height: 0;
  position: relative;
}

.virtual-spacer {
  position: relative;
  width: 100%;
}

.virtual-inner {
  position: absolute;
  left: 0;
  right: 0;
  top: 0;
}

/* 可点击的时间戳 */
.timestamp {
  display: inline-block;
  color: #569cd6;
  font-size: 11px;
  font-family: 'Consolas', 'Monaco', monospace;
  padding: 0 8px;
  min-width: 170px;
  flex-shrink: 0;
}

.timestamp.clickable {
  cursor: pointer;
  text-decoration: underline dotted;
  text-decoration-color: #4f46e5;
  transition: all 0.15s;
}

.timestamp.clickable:hover {
  color: #818cf8;
  background: rgba(99, 102, 241, 0.1);
}

/* 文件分组标题 */
.file-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px;
  background: #252525;
  border-bottom: 1px solid #333;
  font-family: 'SF Mono', 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  color: #8b949e;
  cursor: pointer;
  box-sizing: border-box;
  transition: background-color 0.15s;
}

.file-header:hover {
  background: #2d2d2d;
}

.file-header-arrow {
  color: #6e7681;
  font-size: 10px;
  width: 12px;
  flex-shrink: 0;
  user-select: none;
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

/* 日志行 */
.log-row {
  display: flex;
  align-items: flex-start;
  padding: 6px 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  line-height: 1.6;
  transition: background-color 0.15s;
  box-sizing: border-box;
}

.log-row:hover {
  background-color: #2a2a2a;
}

.match-row {
  background-color: rgba(64, 158, 255, 0.05);
}

.match-row.highlighted {
  background-color: rgba(252, 211, 77, 0.2);
  border-left: 3px solid #fcd34d;
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

.content-wrapper {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 4px;
}

.content {
  flex: 1;
  word-break: break-all;
  color: #e6e6e6;
  white-space: pre-wrap;
  min-width: 0;
}

.expand-btn {
  color: #6366f1;
  cursor: pointer;
  font-size: 11px;
  padding: 0 4px;
  flex-shrink: 0;
  user-select: none;
}

.expand-btn:hover {
  color: #818cf8;
  text-decoration: underline;
}

.content :deep(.highlight) {
  background: #fcd34d;
  color: #1e1e1e;
  padding: 1px 4px;
  border-radius: 3px;
  font-weight: 600;
}

.content :deep(.find-highlight) {
  background: #ff6b6b;
  color: #fff;
  padding: 1px 3px;
  border-radius: 3px;
}

.context-row .content {
  color: #8b949e;
}

.context-row .content :deep(.highlight) {
  background: rgba(255, 215, 0, 0.3);
  color: #c9d1d9;
}

/* 空状态 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  color: #6e7681;
  background: #1e1e1e;
}

.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.search-spinner {
  animation: spin 1.5s linear infinite;
  display: inline-block;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.empty-state p {
  margin: 4px 0;
}

.empty-hint {
  font-size: 14px;
  color: #4a4a4a;
}

/* 滚动条 */
.log-list-container::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.log-list-container::-webkit-scrollbar-track {
  background: #1e1e1e;
}

.log-list-container::-webkit-scrollbar-thumb {
  background: #3a3a3a;
  border-radius: 4px;
}

.log-list-container::-webkit-scrollbar-thumb:hover {
  background: #4a4a4a;
}

/* 滚动到最新按钮 */
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
