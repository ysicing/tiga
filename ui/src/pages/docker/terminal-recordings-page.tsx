import React, { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { format } from 'date-fns';
import {
  Video,
  Trash2,
  Play,
  Calendar,
  Clock,
  HardDrive,
  User,
  Server,
  Box,
  Filter,
  RefreshCw,
} from 'lucide-react';

interface TerminalRecording {
  id: string;
  session_id: string;
  instance_id: string;
  container_id: string;
  username: string;
  started_at: string;
  ended_at?: string;
  duration: number;
  file_size: number;
  client_ip: string;
  description?: string;
}

interface RecordingsResponse {
  success: boolean;
  data: {
    recordings: TerminalRecording[];
    pagination: {
      total: number;
      page: number;
      page_size: number;
    };
  };
}

interface StatisticsResponse {
  success: boolean;
  data: {
    total_recordings: number;
    total_duration: number;
    total_size: number;
    avg_duration: number;
    avg_size: number;
    recordings_today: number;
  };
}

// API functions
const fetchRecordings = async (page: number, pageSize: number, filters: any): Promise<RecordingsResponse> => {
  const params = new URLSearchParams({
    page: page.toString(),
    page_size: pageSize.toString(),
  });

  if (filters.userId) params.append('user_id', filters.userId);
  if (filters.instanceId) params.append('instance_id', filters.instanceId);
  if (filters.containerId) params.append('container_id', filters.containerId);
  if (filters.startDate) params.append('start_date', filters.startDate);
  if (filters.endDate) params.append('end_date', filters.endDate);

  const response = await fetch(`/api/v1/docker/recordings?${params}`, {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch recordings');
  }

  return response.json();
};

const fetchStatistics = async (): Promise<StatisticsResponse> => {
  const response = await fetch('/api/v1/docker/recordings/statistics', {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch statistics');
  }

  return response.json();
};

const deleteRecording = async (id: string): Promise<void> => {
  const response = await fetch(`/api/v1/docker/recordings/${id}`, {
    method: 'DELETE',
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to delete recording');
  }
};

// Utility functions
const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds}s`;
};

const formatFileSize = (bytes: number): string => {
  const KB = 1024;
  const MB = KB * 1024;
  const GB = MB * 1024;

  if (bytes >= GB) return `${(bytes / GB).toFixed(2)} GB`;
  if (bytes >= MB) return `${(bytes / MB).toFixed(2)} MB`;
  if (bytes >= KB) return `${(bytes / KB).toFixed(2)} KB`;
  return `${bytes} B`;
};

export default function TerminalRecordingsPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [filters, setFilters] = useState({});
  const [showFilters, setShowFilters] = useState(false);

  // Fetch recordings
  const { data: recordingsData, isLoading, error, refetch } = useQuery({
    queryKey: ['terminal-recordings', page, pageSize, filters],
    queryFn: () => fetchRecordings(page, pageSize, filters),
  });

  // Fetch statistics
  const { data: statsData } = useQuery({
    queryKey: ['terminal-recordings-statistics'],
    queryFn: fetchStatistics,
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: deleteRecording,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['terminal-recordings'] });
      queryClient.invalidateQueries({ queryKey: ['terminal-recordings-statistics'] });
    },
  });

  const handleDelete = async (id: string, username: string) => {
    if (window.confirm(`确定要删除 ${username} 的终端录制吗？此操作无法撤销。`)) {
      try {
        await deleteMutation.mutateAsync(id);
      } catch (error) {
        alert('删除失败: ' + (error as Error).message);
      }
    }
  };

  const recordings = recordingsData?.data.recordings || [];
  const pagination = recordingsData?.data.pagination;
  const stats = statsData?.data;

  const totalPages = pagination ? Math.ceil(pagination.total / pagination.page_size) : 0;

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <Video className="w-8 h-8 text-blue-600" />
          <div>
            <h1 className="text-2xl font-bold text-gray-900">终端录制</h1>
            <p className="text-sm text-gray-500">查看和管理所有终端会话录制</p>
          </div>
        </div>

        <div className="flex space-x-3">
          <button
            onClick={() => setShowFilters(!showFilters)}
            className="flex items-center px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
          >
            <Filter className="w-4 h-4 mr-2" />
            {showFilters ? '隐藏' : '显示'}筛选
          </button>
          <button
            onClick={() => refetch()}
            className="flex items-center px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
          >
            <RefreshCw className="w-4 h-4 mr-2" />
            刷新
          </button>
        </div>
      </div>

      {/* Statistics Cards */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-6 gap-4">
          <div className="bg-white p-4 rounded-lg border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">总录制数</p>
                <p className="text-2xl font-bold text-gray-900">{stats.total_recordings}</p>
              </div>
              <Video className="w-8 h-8 text-blue-500" />
            </div>
          </div>

          <div className="bg-white p-4 rounded-lg border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">今日录制</p>
                <p className="text-2xl font-bold text-green-600">{stats.recordings_today}</p>
              </div>
              <Calendar className="w-8 h-8 text-green-500" />
            </div>
          </div>

          <div className="bg-white p-4 rounded-lg border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">总时长</p>
                <p className="text-2xl font-bold text-purple-600">{formatDuration(stats.total_duration)}</p>
              </div>
              <Clock className="w-8 h-8 text-purple-500" />
            </div>
          </div>

          <div className="bg-white p-4 rounded-lg border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">平均时长</p>
                <p className="text-2xl font-bold text-orange-600">{formatDuration(Math.floor(stats.avg_duration))}</p>
              </div>
              <Clock className="w-8 h-8 text-orange-500" />
            </div>
          </div>

          <div className="bg-white p-4 rounded-lg border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">总存储</p>
                <p className="text-2xl font-bold text-red-600">{formatFileSize(stats.total_size)}</p>
              </div>
              <HardDrive className="w-8 h-8 text-red-500" />
            </div>
          </div>

          <div className="bg-white p-4 rounded-lg border border-gray-200">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-gray-500">平均大小</p>
                <p className="text-2xl font-bold text-indigo-600">{formatFileSize(Math.floor(stats.avg_size))}</p>
              </div>
              <HardDrive className="w-8 h-8 text-indigo-500" />
            </div>
          </div>
        </div>
      )}

      {/* Filters */}
      {showFilters && (
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <h3 className="text-lg font-semibold mb-4">筛选条件</h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <input
              type="text"
              placeholder="容器 ID"
              className="px-3 py-2 border border-gray-300 rounded-lg"
              onChange={(e) => setFilters({ ...filters, containerId: e.target.value })}
            />
            <input
              type="date"
              placeholder="开始日期"
              className="px-3 py-2 border border-gray-300 rounded-lg"
              onChange={(e) => setFilters({ ...filters, startDate: e.target.value })}
            />
            <input
              type="date"
              placeholder="结束日期"
              className="px-3 py-2 border border-gray-300 rounded-lg"
              onChange={(e) => setFilters({ ...filters, endDate: e.target.value })}
            />
            <button
              onClick={() => {
                setFilters({});
                setPage(1);
              }}
              className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200"
            >
              清除筛选
            </button>
          </div>
        </div>
      )}

      {/* Recordings Table */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        {isLoading ? (
          <div className="p-12 text-center">
            <RefreshCw className="w-8 h-8 mx-auto text-gray-400 animate-spin" />
            <p className="mt-4 text-gray-500">加载中...</p>
          </div>
        ) : error ? (
          <div className="p-12 text-center">
            <p className="text-red-600">加载失败: {(error as Error).message}</p>
            <button
              onClick={() => refetch()}
              className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              重试
            </button>
          </div>
        ) : recordings.length === 0 ? (
          <div className="p-12 text-center">
            <Video className="w-12 h-12 mx-auto text-gray-400" />
            <p className="mt-4 text-gray-500">暂无录制记录</p>
          </div>
        ) : (
          <>
            <table className="w-full">
              <thead className="bg-gray-50 border-b border-gray-200">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    用户
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    容器
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    开始时间
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    时长
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    文件大小
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    客户端 IP
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    操作
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {recordings.map((recording) => (
                  <tr key={recording.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <User className="w-4 h-4 text-gray-400 mr-2" />
                        <span className="text-sm font-medium text-gray-900">{recording.username}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <Box className="w-4 h-4 text-gray-400 mr-2" />
                        <span className="text-sm text-gray-500 font-mono">
                          {recording.container_id.substring(0, 12)}
                        </span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-gray-900">
                        {format(new Date(recording.started_at), 'yyyy-MM-dd HH:mm:ss')}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <Clock className="w-4 h-4 text-gray-400 mr-2" />
                        <span className="text-sm text-gray-900">{formatDuration(recording.duration)}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="flex items-center">
                        <HardDrive className="w-4 h-4 text-gray-400 mr-2" />
                        <span className="text-sm text-gray-900">{formatFileSize(recording.file_size)}</span>
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {recording.client_ip}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <div className="flex items-center justify-end space-x-2">
                        <Link
                          to={`/docker/recordings/${recording.id}/play`}
                          className="inline-flex items-center px-3 py-1.5 bg-blue-600 text-white rounded hover:bg-blue-700"
                        >
                          <Play className="w-4 h-4 mr-1" />
                          播放
                        </Link>
                        <button
                          onClick={() => handleDelete(recording.id, recording.username)}
                          disabled={deleteMutation.isPending}
                          className="inline-flex items-center px-3 py-1.5 bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
                        >
                          <Trash2 className="w-4 h-4 mr-1" />
                          删除
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>

            {/* Pagination */}
            {pagination && pagination.total > pageSize && (
              <div className="px-6 py-4 bg-gray-50 border-t border-gray-200 flex items-center justify-between">
                <div className="text-sm text-gray-700">
                  显示 {(page - 1) * pageSize + 1} 到 {Math.min(page * pageSize, pagination.total)} 条，
                  共 {pagination.total} 条
                </div>
                <div className="flex space-x-2">
                  <button
                    onClick={() => setPage(page - 1)}
                    disabled={page === 1}
                    className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    上一页
                  </button>
                  <div className="flex items-center space-x-1">
                    {[...Array(totalPages)].map((_, i) => {
                      const pageNum = i + 1;
                      // Show first, last, current, and adjacent pages
                      if (
                        pageNum === 1 ||
                        pageNum === totalPages ||
                        (pageNum >= page - 1 && pageNum <= page + 1)
                      ) {
                        return (
                          <button
                            key={pageNum}
                            onClick={() => setPage(pageNum)}
                            className={`px-4 py-2 border rounded-lg ${
                              page === pageNum
                                ? 'bg-blue-600 text-white border-blue-600'
                                : 'border-gray-300 hover:bg-gray-50'
                            }`}
                          >
                            {pageNum}
                          </button>
                        );
                      } else if (pageNum === page - 2 || pageNum === page + 2) {
                        return <span key={pageNum} className="px-2">...</span>;
                      }
                      return null;
                    })}
                  </div>
                  <button
                    onClick={() => setPage(page + 1)}
                    disabled={page === totalPages}
                    className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    下一页
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
