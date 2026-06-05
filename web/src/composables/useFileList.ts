import { ref, reactive } from 'vue';
import type { FileInfo, Pagination, FileListRequest, FileListResponse } from '../types';

export function useFileList() {
  const files = ref<FileInfo[]>([]);
  const pagination = reactive<Pagination>({
    page: 1,
    page_size: 50,
    total: 0,
    total_pages: 1,
    has_next: false,
    has_prev: false,
  });
  const loading = ref(false);
  const error = ref<string | null>(null);
  const apiKey = ref<string>('');

  const fetchFiles = async (params: FileListRequest = {}) => {
    loading.value = true;

    try {
      const queryParams = new URLSearchParams();
      if (params.path) queryParams.set('path', params.path);
      if (params.page) queryParams.set('page', params.page.toString());
      if (params.page_size) queryParams.set('page_size', params.page_size.toString());
      if (params.since) queryParams.set('since', params.since.toString());

      const headers: HeadersInit = {
        'Content-Type': 'application/json',
      };
      // Note: Authorization header is added by auth interceptor

      const res = await fetch('/api/v1/files/list?' + queryParams.toString(), {
        method: 'GET',
        headers,
      });

      let data: FileListResponse | null = null;
      try {
        data = await res.json();
      } catch (parseErr) {
        // Response body may not be JSON - handle gracefully
      }

      if (data && data.success && res.ok) {
        // 正常返回数据，更新文件列表，清除错误
        files.value = data.data || [];
        if (data.pagination) {
          pagination.page = data.pagination.page ?? pagination.page;
          pagination.page_size = data.pagination.page_size ?? pagination.page_size;
          pagination.total = data.pagination.total ?? pagination.total;
          pagination.total_pages = data.pagination.total_pages ?? pagination.total_pages;
          pagination.has_next = data.pagination.has_next ?? false;
          pagination.has_prev = data.pagination.has_prev ?? false;
        }
        error.value = null;
      } else if (res.status === 401) {
        // 401 由 auth interceptor 处理，这里保持原有数据，不清空
        error.value = data?.error?.message || '登录已过期，请重新登录';
        // 注意：不调用 files.value = [] 以保持模态框打开
      } else {
        // 普通错误（如路径无效）：设置错误消息，不清空文件列表
        // 这样空目录浏览失败时用户能看到错误，而不是模态框消失
        error.value = data?.error?.message || '加载文件列表失败';
      }
    } catch (e) {
      error.value = '网络请求失败，请检查服务器连接';
      console.warn('Failed to fetch files:', e);
    } finally {
      loading.value = false;
    }
  };

  const goToPage = (page: number, params: FileListRequest = {}) => {
    return fetchFiles({
      ...params,
      page,
      page_size: pagination.page_size,
    });
  };

  const nextPage = (params: FileListRequest = {}) => {
    if (pagination.has_next) {
      return goToPage(pagination.page + 1, params);
    }
  };

  const prevPage = (params: FileListRequest = {}) => {
    if (pagination.has_prev) {
      return goToPage(pagination.page - 1, params);
    }
  };

  return {
    files,
    pagination,
    loading,
    error,
    apiKey,
    fetchFiles,
    goToPage,
    nextPage,
    prevPage,
  };
}
