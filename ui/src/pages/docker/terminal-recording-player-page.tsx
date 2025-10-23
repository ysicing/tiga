import React, { useEffect, useRef, useState } from 'react';
import { useParams, Link, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import {
  Video,
  ArrowLeft,
  Download,
  Clock,
  HardDrive,
  User,
  Server,
  Box,
  Calendar,
  Terminal,
  AlertCircle,
  RefreshCw,
} from 'lucide-react';
import * as AsciinemaPlayer from 'asciinema-player';
import 'asciinema-player/dist/bundle/asciinema-player.css';

interface RecordingMetadata {
  id: string;
  session_id: string;
  instance_id: string;
  container_id: string;
  username: string;
  started_at: string;
  ended_at?: string;
  duration: number;
  file_size: number;
  format: string;
  rows: number;
  cols: number;
  shell: string;
  client_ip: string;
  description?: string;
}

interface RecordingResponse {
  success: boolean;
  data: RecordingMetadata;
}

// API functions
const fetchRecording = async (id: string): Promise<RecordingResponse> => {
  const response = await fetch(`/api/v1/docker/recordings/${id}`, {
    headers: {
      'Authorization': `Bearer ${localStorage.getItem('token')}`,
    },
  });

  if (!response.ok) {
    throw new Error('Failed to fetch recording');
  }

  return response.json();
};

// Utility functions
const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  if (minutes < 60) return `${minutes}m ${remainingSeconds}s`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return `${hours}h ${remainingMinutes}m ${remainingSeconds}s`;
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

export default function TerminalRecordingPlayerPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const playerContainerRef = useRef<HTMLDivElement>(null);
  const [playerLoaded, setPlayerLoaded] = useState(false);
  const [playerError, setPlayerError] = useState<string | null>(null);

  // Fetch recording metadata
  const { data: recordingData, isLoading, error, refetch } = useQuery({
    queryKey: ['terminal-recording', id],
    queryFn: () => fetchRecording(id!),
    enabled: !!id,
  });

  const recording = recordingData?.data;

  // Initialize asciinema player
  useEffect(() => {
    if (!recording || !playerContainerRef.current || playerLoaded) {
      return;
    }

    // Clear container before initializing player
    playerContainerRef.current.innerHTML = '';

    try {
      const playbackUrl = `/api/v1/docker/recordings/${id}/playback`;

      AsciinemaPlayer.create(playbackUrl, playerContainerRef.current, {
        speed: 1.0,
        theme: 'monokai',
        loop: false,
        autoPlay: false,
        controls: true,
        fit: 'width',
        terminalFontSize: '14px',
        terminalLineHeight: 1.33333333,
      });

      setPlayerLoaded(true);
      setPlayerError(null);
    } catch (err) {
      console.error('Failed to initialize asciinema player:', err);
      setPlayerError('Failed to load terminal recording player');
    }
  }, [recording, id, playerLoaded]);

  // Handle download
  const handleDownload = () => {
    if (!id) return;

    const downloadUrl = `/api/v1/docker/recordings/${id}/playback`;
    const link = document.createElement('a');
    link.href = downloadUrl;
    link.download = `recording-${id}.cast`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  if (isLoading) {
    return (
      <div className="p-6 space-y-6">
        <div className="flex items-center justify-between">
          <Link
            to="/docker/recordings"
            className="flex items-center text-gray-600 hover:text-gray-900"
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            返回录制列表
          </Link>
        </div>
        <div className="flex items-center justify-center py-20">
          <RefreshCw className="w-8 h-8 text-gray-400 animate-spin" />
          <p className="ml-4 text-gray-500">加载录制信息...</p>
        </div>
      </div>
    );
  }

  if (error || !recording) {
    return (
      <div className="p-6 space-y-6">
        <div className="flex items-center justify-between">
          <Link
            to="/docker/recordings"
            className="flex items-center text-gray-600 hover:text-gray-900"
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            返回录制列表
          </Link>
        </div>
        <div className="flex flex-col items-center justify-center py-20">
          <AlertCircle className="w-16 h-16 text-red-500 mb-4" />
          <p className="text-red-600 text-lg mb-4">
            {error ? (error as Error).message : '录制不存在'}
          </p>
          <button
            onClick={() => refetch()}
            className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            重试
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-4">
          <Link
            to="/docker/recordings"
            className="flex items-center text-gray-600 hover:text-gray-900"
          >
            <ArrowLeft className="w-4 h-4 mr-2" />
            返回录制列表
          </Link>
          <div className="flex items-center space-x-3">
            <Video className="w-8 h-8 text-blue-600" />
            <div>
              <h1 className="text-2xl font-bold text-gray-900">终端录制回放</h1>
              <p className="text-sm text-gray-500">
                录制于 {format(new Date(recording.started_at), 'yyyy-MM-dd HH:mm:ss')}
              </p>
            </div>
          </div>
        </div>

        <button
          onClick={handleDownload}
          className="flex items-center px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
        >
          <Download className="w-4 h-4 mr-2" />
          下载录制文件
        </button>
      </div>

      {/* Recording Metadata Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500">用户</p>
              <p className="text-lg font-semibold text-gray-900">{recording.username}</p>
            </div>
            <User className="w-8 h-8 text-blue-500" />
          </div>
        </div>

        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500">容器 ID</p>
              <p className="text-lg font-semibold text-gray-900 font-mono">
                {recording.container_id.substring(0, 12)}
              </p>
            </div>
            <Box className="w-8 h-8 text-green-500" />
          </div>
        </div>

        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500">时长</p>
              <p className="text-lg font-semibold text-gray-900">
                {formatDuration(recording.duration)}
              </p>
            </div>
            <Clock className="w-8 h-8 text-purple-500" />
          </div>
        </div>

        <div className="bg-white p-4 rounded-lg border border-gray-200">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500">文件大小</p>
              <p className="text-lg font-semibold text-gray-900">
                {formatFileSize(recording.file_size)}
              </p>
            </div>
            <HardDrive className="w-8 h-8 text-orange-500" />
          </div>
        </div>
      </div>

      {/* Additional Details */}
      <div className="bg-white p-6 rounded-lg border border-gray-200">
        <h3 className="text-lg font-semibold mb-4">录制详情</h3>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="flex items-start">
            <Calendar className="w-5 h-5 text-gray-400 mr-3 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-gray-700">开始时间</p>
              <p className="text-sm text-gray-600">
                {format(new Date(recording.started_at), 'yyyy-MM-dd HH:mm:ss')}
              </p>
            </div>
          </div>

          {recording.ended_at && (
            <div className="flex items-start">
              <Calendar className="w-5 h-5 text-gray-400 mr-3 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-gray-700">结束时间</p>
                <p className="text-sm text-gray-600">
                  {format(new Date(recording.ended_at), 'yyyy-MM-dd HH:mm:ss')}
                </p>
              </div>
            </div>
          )}

          <div className="flex items-start">
            <Terminal className="w-5 h-5 text-gray-400 mr-3 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-gray-700">Shell</p>
              <p className="text-sm text-gray-600 font-mono">{recording.shell}</p>
            </div>
          </div>

          <div className="flex items-start">
            <Server className="w-5 h-5 text-gray-400 mr-3 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-gray-700">终端尺寸</p>
              <p className="text-sm text-gray-600">
                {recording.cols} × {recording.rows}
              </p>
            </div>
          </div>

          <div className="flex items-start">
            <Server className="w-5 h-5 text-gray-400 mr-3 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-gray-700">客户端 IP</p>
              <p className="text-sm text-gray-600">{recording.client_ip}</p>
            </div>
          </div>

          <div className="flex items-start">
            <Video className="w-5 h-5 text-gray-400 mr-3 mt-0.5" />
            <div>
              <p className="text-sm font-medium text-gray-700">格式</p>
              <p className="text-sm text-gray-600">{recording.format}</p>
            </div>
          </div>
        </div>

        {recording.description && (
          <div className="mt-4 pt-4 border-t border-gray-200">
            <p className="text-sm font-medium text-gray-700 mb-2">描述</p>
            <p className="text-sm text-gray-600">{recording.description}</p>
          </div>
        )}
      </div>

      {/* Terminal Player */}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div className="bg-gray-800 px-4 py-3 flex items-center space-x-2">
          <div className="flex space-x-2">
            <div className="w-3 h-3 rounded-full bg-red-500"></div>
            <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
          </div>
          <span className="text-gray-300 text-sm ml-4">
            {recording.username}@{recording.container_id.substring(0, 12)}
          </span>
        </div>

        {playerError ? (
          <div className="p-8 text-center">
            <AlertCircle className="w-12 h-12 text-red-500 mx-auto mb-4" />
            <p className="text-red-600">{playerError}</p>
            <button
              onClick={() => {
                setPlayerLoaded(false);
                setPlayerError(null);
              }}
              className="mt-4 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              重试
            </button>
          </div>
        ) : (
          <div
            ref={playerContainerRef}
            className="bg-black"
            style={{ minHeight: '400px' }}
          ></div>
        )}
      </div>

      {/* Player Instructions */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h4 className="text-sm font-semibold text-blue-900 mb-2">播放提示</h4>
        <ul className="text-sm text-blue-800 space-y-1">
          <li>• 点击播放按钮开始回放终端会话</li>
          <li>• 可以使用进度条快进或后退</li>
          <li>• 支持暂停和倍速播放</li>
          <li>• 点击下载按钮可保存录制文件（.cast 格式）</li>
        </ul>
      </div>
    </div>
  );
}
