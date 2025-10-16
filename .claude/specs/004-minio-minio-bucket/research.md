# 研究文档：MinIO 对象存储管理系统

**功能**：004-minio-minio-bucket | **日期**：2025-10-14
**状态**：已完成 | **参考**：[plan.md](./plan.md)

## 研究概述

本文档记录了 MinIO 对象存储管理系统实施前的技术研究，所有决策基于项目现有技术栈、最佳实践和功能需求。

## 1. MinIO Go SDK 使用模式

### 决策：使用 MinIO Go SDK v7

**选择理由**：
- 官方 SDK，稳定可靠，与 MinIO 服务器版本兼容性好
- 提供完整的 S3 API 支持，包括 presigned URL、策略管理、multipart upload
- 文档完善，社区活跃，示例代码丰富
- 支持连接池和自动重试，适合生产环境

**考虑的替代方案**：
- AWS S3 SDK：通用性强，但 MinIO 专属功能支持不完整（如 IAM 策略）
- 直接使用 HTTP 客户端：灵活性高，但需要手动处理签名、错误重试等复杂逻辑

### 客户端连接池管理

**最佳实践**：
```go
// pkg/minio/client.go
type ClientManager struct {
    clients sync.Map // instance_id -> *minio.Client
    mu      sync.RWMutex
}

func (m *ClientManager) GetClient(instance *models.MinIOInstance) (*minio.Client, error) {
    // 1. 从缓存获取客户端
    if client, ok := m.clients.Load(instance.ID); ok {
        return client.(*minio.Client), nil
    }

    // 2. 创建新客户端
    client, err := minio.New(instance.Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(instance.AccessKey, instance.SecretKey, ""),
        Secure: instance.UseSSL,
    })
    if err != nil {
        return nil, err
    }

    // 3. 缓存客户端
    m.clients.Store(instance.ID, client)
    return client, nil
}
```

**连接健康检查**：
```go
func (m *ClientManager) HealthCheck(ctx context.Context, instanceID uint) error {
    client, err := m.GetClient(instanceID)
    if err != nil {
        return err
    }

    // 使用 ListBuckets 作为健康检查
    _, err = client.ListBuckets(ctx)
    return err
}
```

### Presigned URL 生成和过期控制

**配置策略**：
- **下载链接**：7 天过期（默认），可配置 1小时-30天
- **上传链接**：1 小时过期（安全考虑）
- **预览链接**：15 分钟过期（短期缓存，频繁访问）

**实现示例**：
```go
func (s *ShareService) GenerateShareLink(ctx context.Context, bucket, object string, expires time.Duration) (string, error) {
    presignedURL, err := s.minioClient.PresignedGetObject(ctx, bucket, object, expires, nil)
    if err != nil {
        return "", fmt.Errorf("generate presigned URL: %w", err)
    }

    // 记录分享链接到数据库（审计）
    shareLink := &models.MinIOShareLink{
        Bucket:     bucket,
        ObjectKey:  object,
        URL:        presignedURL.String(),
        ExpiresAt:  time.Now().Add(expires),
        CreatedBy:  getCurrentUserID(ctx),
    }

    return presignedURL.String(), s.repo.CreateShareLink(ctx, shareLink)
}
```

### Bucket 策略 JSON 生成

**策略模板**（Bucket 级只读）：
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": ["arn:aws:iam::minio:user/USERNAME"]},
      "Action": ["s3:GetObject", "s3:ListBucket"],
      "Resource": [
        "arn:aws:s3:::BUCKET_NAME/*",
        "arn:aws:s3:::BUCKET_NAME"
      ]
    }
  ]
}
```

**策略模板**（前缀级读写）：
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": ["arn:aws:iam::minio:user/USERNAME"]},
      "Action": [
        "s3:GetObject", "s3:PutObject", "s3:DeleteObject",
        "s3:ListBucket"
      ],
      "Resource": ["arn:aws:s3:::BUCKET_NAME/PREFIX/*"],
      "Condition": {"StringLike": {"s3:prefix": ["PREFIX/*"]}}
    }
  ]
}
```

**策略生成函数**：
```go
// pkg/minio/policy.go
func GenerateBucketPolicy(username, bucket string, permission PermissionType, prefix string) (string, error) {
    actions := getActionsForPermission(permission)
    resources := []string{fmt.Sprintf("arn:aws:s3:::%s/*", bucket)}

    if prefix != "" {
        resources[0] = fmt.Sprintf("arn:aws:s3:::%s/%s/*", bucket, prefix)
    }

    policy := map[string]interface{}{
        "Version": "2012-10-17",
        "Statement": []map[string]interface{}{
            {
                "Effect": "Allow",
                "Principal": map[string]interface{}{
                    "AWS": []string{fmt.Sprintf("arn:aws:iam::minio:user/%s", username)},
                },
                "Action":   actions,
                "Resource": resources,
            },
        },
    }

    return json.Marshal(policy)
}

func getActionsForPermission(perm PermissionType) []string {
    switch perm {
    case PermissionReadOnly:
        return []string{"s3:GetObject", "s3:ListBucket"}
    case PermissionWriteOnly:
        return []string{"s3:PutObject", "s3:DeleteObject"}
    case PermissionReadWrite:
        return []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject", "s3:ListBucket"}
    default:
        return []string{}
    }
}
```

### 错误处理和重试策略

**错误分类**：
1. **可重试错误**：网络超时、503 服务不可用、临时连接失败
2. **不可重试错误**：404 Not Found、403 Forbidden、400 Bad Request
3. **需要重新认证**：401 Unauthorized、SignatureDoesNotMatch

**重试配置**：
```go
type RetryConfig struct {
    MaxRetries int           // 最大重试次数：3
    InitialDelay time.Duration // 初始延迟：100ms
    MaxDelay     time.Duration // 最大延迟：5s
    Multiplier   float64      // 延迟倍数：2.0（指数退避）
}

func withRetry(ctx context.Context, fn func() error) error {
    cfg := getRetryConfig()
    delay := cfg.InitialDelay

    for i := 0; i <= cfg.MaxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }

        if !isRetryable(err) || i == cfg.MaxRetries {
            return err
        }

        select {
        case <-time.After(delay):
            delay = time.Duration(float64(delay) * cfg.Multiplier)
            if delay > cfg.MaxDelay {
                delay = cfg.MaxDelay
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }

    return fmt.Errorf("max retries exceeded")
}
```

## 2. 前端文件上传最佳实践

### 决策：使用 tus-js-client + MinIO multipart upload

**选择理由**：
- tus-js-client 实现了 TUS 协议（断点续传标准），稳定可靠
- MinIO 原生支持 TUS 协议和 S3 multipart upload
- 提供实时进度回调、速度计算、错误恢复
- 浏览器兼容性好，支持大文件（GB 级）

**考虑的替代方案**：
- 直接使用 fetch/axios multipart：需要手动实现断点续传逻辑，复杂度高
- Uppy.js：功能完整的上传组件库，但打包体积较大（200KB+），对于我们的需求过度

### 多文件并发上传

**实现策略**：
```typescript
// ui/src/hooks/use-upload.ts
interface UploadTask {
  id: string;
  file: File;
  progress: number;
  speed: number; // bytes/s
  remaining: number; // seconds
  status: 'pending' | 'uploading' | 'completed' | 'failed' | 'paused';
  error?: string;
}

export function useUpload(bucketName: string, prefix: string) {
  const [tasks, setTasks] = useState<UploadTask[]>([]);
  const MAX_CONCURRENT = 5; // 最大并发数

  const addFiles = (files: File[]) => {
    const newTasks = files.map(file => ({
      id: generateId(),
      file,
      progress: 0,
      speed: 0,
      remaining: 0,
      status: 'pending' as const,
    }));

    setTasks(prev => [...prev, ...newTasks]);
    processQueue();
  };

  const processQueue = () => {
    const uploading = tasks.filter(t => t.status === 'uploading').length;
    const pending = tasks.filter(t => t.status === 'pending');

    const slotsAvailable = MAX_CONCURRENT - uploading;
    const toStart = pending.slice(0, slotsAvailable);

    toStart.forEach(task => uploadFile(task));
  };

  const uploadFile = async (task: UploadTask) => {
    // 更新状态为上传中
    updateTaskStatus(task.id, 'uploading');

    try {
      // 获取 presigned upload URL
      const uploadUrl = await getPresignedUploadUrl(bucketName, prefix, task.file.name);

      // 使用 tus 上传
      const upload = new tus.Upload(task.file, {
        endpoint: uploadUrl,
        retryDelays: [0, 1000, 3000, 5000],
        chunkSize: 5 * 1024 * 1024, // 5MB chunks
        metadata: {
          filename: task.file.name,
          filetype: task.file.type,
        },
        onProgress: (bytesUploaded, bytesTotal) => {
          const progress = (bytesUploaded / bytesTotal) * 100;
          const speed = calculateSpeed(bytesUploaded, task.startTime);
          const remaining = (bytesTotal - bytesUploaded) / speed;

          updateTaskProgress(task.id, progress, speed, remaining);
        },
        onSuccess: () => {
          updateTaskStatus(task.id, 'completed');
          processQueue(); // 继续处理队列
        },
        onError: (error) => {
          updateTaskStatus(task.id, 'failed', error.message);
          processQueue();
        },
      });

      upload.start();
    } catch (error) {
      updateTaskStatus(task.id, 'failed', error.message);
      processQueue();
    }
  };

  return { tasks, addFiles, pauseTask, resumeTask, cancelTask };
}
```

### 断点续传实现

**tus 协议关键特性**：
1. **分片上传**：文件分为 5MB 块上传，失败块可单独重试
2. **状态保存**：上传进度保存到 localStorage，浏览器刷新后可恢复
3. **指纹识别**：基于文件内容生成唯一标识，避免重复上传

**MinIO 配置要求**：
```bash
# MinIO 需要启用 TUS 协议支持（默认已启用）
mc admin config set myminio api requests_max=1024
mc admin config set myminio api requests_deadline=10m
```

### 实时进度显示

**速度计算（移动平均）**：
```typescript
class SpeedCalculator {
  private samples: Array<{bytes: number; timestamp: number}> = [];
  private windowSize = 10; // 10 秒窗口

  addSample(bytesUploaded: number) {
    const now = Date.now();
    this.samples.push({ bytes: bytesUploaded, timestamp: now });

    // 移除超过窗口的样本
    const cutoff = now - this.windowSize * 1000;
    this.samples = this.samples.filter(s => s.timestamp > cutoff);
  }

  getSpeed(): number {
    if (this.samples.length < 2) return 0;

    const first = this.samples[0];
    const last = this.samples[this.samples.length - 1];

    const bytesDiff = last.bytes - first.bytes;
    const timeDiff = (last.timestamp - first.timestamp) / 1000; // seconds

    return bytesDiff / timeDiff; // bytes/s
  }

  getRemainingTime(bytesRemaining: number): number {
    const speed = this.getSpeed();
    return speed > 0 ? bytesRemaining / speed : 0;
  }
}
```

**进度展示组件**：
```tsx
function UploadProgress({ task }: { task: UploadTask }) {
  return (
    <div>
      <div className="flex justify-between mb-1">
        <span>{task.file.name}</span>
        <span>{task.progress.toFixed(1)}%</span>
      </div>

      <Progress value={task.progress} />

      <div className="flex justify-between text-sm text-gray-500 mt-1">
        <span>{formatSpeed(task.speed)}</span>
        <span>剩余 {formatTime(task.remaining)}</span>
      </div>
    </div>
  );
}
```

## 3. 文件预览技术选型

### 3.1 图片预览

**决策：react-photo-view**

**选择理由**：
- 轻量级（~30KB gzipped），性能优秀
- 支持缩放、旋转、拖拽，触摸手势
- 与 Radix UI 兼容性好（可在 Dialog 中使用）
- TypeScript 支持完善

**使用示例**：
```tsx
import { PhotoProvider, PhotoView } from 'react-photo-view';
import 'react-photo-view/dist/react-photo-view.css';

function ImageGallery({ images }: { images: FileObject[] }) {
  return (
    <PhotoProvider>
      <div className="grid grid-cols-4 gap-4">
        {images.map((img) => (
          <PhotoView key={img.key} src={img.presignedUrl}>
            <img
              src={img.thumbnailUrl}
              alt={img.name}
              className="cursor-pointer rounded-md"
            />
          </PhotoView>
        ))}
      </div>
    </PhotoProvider>
  );
}
```

**缩略图策略**：
- 使用 MinIO `GetObject` 带 `Range` header 获取前 100KB（适用于 JPEG 渐进式）
- 或使用 MinIO Image Transformation API（如果启用）
- 前端懒加载：react-window + Intersection Observer

### 3.2 视频播放

**决策：原生 HTML5 video + plyr.js（可选）**

**选择理由**：
- 原生 video 标签浏览器支持好，无需额外依赖
- 支持 HTTP Range 请求，渐进式加载
- plyr.js 提供统一的播放器 UI（可选，11KB gzipped）

**实现示例**：
```tsx
function VideoPlayer({ file }: { file: FileObject }) {
  const videoUrl = file.presignedUrl; // 15 分钟有效期

  return (
    <video
      controls
      preload="metadata" // 仅加载元数据
      className="w-full max-h-[600px]"
      onError={(e) => {
        // 处理视频格式不支持
        showError('视频格式不支持，请下载后使用播放器查看');
      }}
    >
      <source src={videoUrl} type={file.contentType} />
      您的浏览器不支持视频播放
    </video>
  );
}
```

**HTTP Range 验证**：
- MinIO 默认支持 `Range` header
- 浏览器自动发送 `Range: bytes=0-` 请求
- 服务器返回 `206 Partial Content`，支持视频拖拽

### 3.3 代码高亮

**决策：Monaco Editor（只读模式）**

**选择理由**：
- VS Code 同款编辑器，语法高亮效果最好
- 支持 60+ 编程语言，自动检测
- 按需加载语言支持，减少打包体积

**轻量级配置**：
```tsx
import Editor from '@monaco-editor/react';

function CodeViewer({ file, code }: { file: FileObject; code: string }) {
  const language = detectLanguage(file.name); // .py -> python, .go -> go

  return (
    <Editor
      height="600px"
      language={language}
      value={code}
      theme="vs-dark"
      options={{
        readOnly: true,
        minimap: { enabled: false }, // 禁用缩略图
        scrollBeyondLastLine: false,
        fontSize: 14,
        lineNumbers: 'on',
        wordWrap: 'on',
      }}
      loading={<Skeleton className="h-[600px]" />}
    />
  );
}

function detectLanguage(filename: string): string {
  const ext = filename.split('.').pop()?.toLowerCase();
  const langMap: Record<string, string> = {
    py: 'python', js: 'javascript', ts: 'typescript',
    go: 'go', java: 'java', cpp: 'cpp', c: 'c',
    rs: 'rust', sh: 'shell', yml: 'yaml', yaml: 'yaml',
    json: 'json', xml: 'xml', html: 'html', css: 'css',
  };
  return langMap[ext || ''] || 'plaintext';
}
```

**按需加载（减少打包体积）**：
```typescript
// vite.config.ts
export default defineConfig({
  plugins: [
    monacoEditorPlugin({
      // 仅加载常用语言
      languages: ['javascript', 'typescript', 'python', 'go', 'java', 'cpp', 'rust'],
    }),
  ],
});
```

### 3.4 Markdown 渲染

**决策：react-markdown + remark-gfm**

**选择理由**：
- 官方推荐，稳定可靠（~15KB gzipped）
- remark-gfm 支持 GitHub 风格（表格、任务列表、删除线）
- 安全：默认过滤 XSS，不执行危险 HTML
- 可扩展：支持自定义组件（代码块语法高亮）

**实现示例**：
```tsx
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

function MarkdownViewer({ content }: { content: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      className="prose prose-sm max-w-none dark:prose-invert"
      components={{
        // 自定义代码块渲染
        code({ node, inline, className, children, ...props }) {
          const match = /language-(\w+)/.exec(className || '');
          return !inline && match ? (
            <SyntaxHighlighter
              style={vscDarkPlus}
              language={match[1]}
              PreTag="div"
              {...props}
            >
              {String(children).replace(/\n$/, '')}
            </SyntaxHighlighter>
          ) : (
            <code className={className} {...props}>
              {children}
            </code>
          );
        },
      }}
    >
      {content}
    </ReactMarkdown>
  );
}
```

**样式配置**：
```css
/* TailwindCSS prose 插件 */
@tailwindcss/typography

/* 或自定义样式 */
.markdown-body {
  h1, h2, h3 { @apply font-bold my-4; }
  a { @apply text-blue-600 underline; }
  table { @apply border-collapse w-full; }
  th, td { @apply border px-4 py-2; }
  code { @apply bg-gray-100 rounded px-1; }
  pre { @apply bg-gray-900 p-4 rounded overflow-x-auto; }
}
```

## 4. 安全策略生成

### MinIO IAM 策略结构

**策略元素**：
1. **Version**：固定为 "2012-10-17"（AWS 策略版本）
2. **Statement**：策略声明数组
   - **Effect**：Allow/Deny
   - **Principal**：授权主体（用户 ARN）
   - **Action**：允许的操作列表
   - **Resource**：资源 ARN（Bucket/Object）
   - **Condition**：条件约束（可选）

### 权限级别映射

**只读（ReadOnly）**：
```json
{
  "Action": [
    "s3:GetObject",
    "s3:GetObjectVersion",
    "s3:ListBucket",
    "s3:GetBucketLocation"
  ]
}
```

**只写（WriteOnly）**：
```json
{
  "Action": [
    "s3:PutObject",
    "s3:DeleteObject",
    "s3:DeleteObjectVersion"
  ]
}
```

**读写（ReadWrite）**：
```json
{
  "Action": [
    "s3:GetObject",
    "s3:GetObjectVersion",
    "s3:PutObject",
    "s3:DeleteObject",
    "s3:DeleteObjectVersion",
    "s3:ListBucket",
    "s3:GetBucketLocation"
  ]
}
```

### 前缀级权限实现

**资源定义**：
```json
{
  "Resource": [
    "arn:aws:s3:::my-bucket/team-a/*",  // 仅访问 team-a/ 前缀
    "arn:aws:s3:::my-bucket"            // 列出 Bucket（需要 ListBucket 权限）
  ],
  "Condition": {
    "StringLike": {
      "s3:prefix": ["team-a/*"]  // 列出时仅显示 team-a/ 前缀的对象
    }
  }
}
```

**多前缀支持**：
```go
func GenerateMultiPrefixPolicy(username, bucket string, prefixes []string, permission PermissionType) (string, error) {
    statements := []map[string]interface{}{}

    for _, prefix := range prefixes {
        resource := fmt.Sprintf("arn:aws:s3:::%s/%s/*", bucket, prefix)
        statements = append(statements, map[string]interface{}{
            "Effect": "Allow",
            "Principal": map[string]interface{}{
                "AWS": []string{fmt.Sprintf("arn:aws:iam::minio:user/%s", username)},
            },
            "Action":   getActionsForPermission(permission),
            "Resource": []string{resource},
            "Condition": map[string]interface{}{
                "StringLike": map[string]interface{}{
                    "s3:prefix": []string{fmt.Sprintf("%s/*", prefix)},
                },
            },
        })
    }

    policy := map[string]interface{}{
        "Version":   "2012-10-17",
        "Statement": statements,
    }

    return json.Marshal(policy)
}
```

### 策略验证

**验证步骤**：
1. **语法验证**：JSON 格式是否正确
2. **语义验证**：Action、Resource 是否有效
3. **冲突检测**：多个 Statement 是否冲突（Allow vs Deny）
4. **测试验证**：创建测试用户，尝试访问资源

**测试函数**：
```go
func TestPolicy(ctx context.Context, client *minio.Client, username, bucket, object string) error {
    // 使用用户凭据创建临时客户端
    userClient, err := minio.New(endpoint, &minio.Options{
        Creds: credentials.NewStaticV4(username, userSecret, ""),
    })
    if err != nil {
        return err
    }

    // 尝试获取对象
    _, err = userClient.GetObject(ctx, bucket, object, minio.GetObjectOptions{})
    return err
}
```

## 5. 性能优化

### 5.1 文件列表虚拟滚动

**决策：react-window**

**选择理由**：
- 轻量级（~7KB gzipped），性能优秀
- 支持固定高度和动态高度列表
- 与 TailwindCSS 集成良好

**实现示例**：
```tsx
import { FixedSizeList as List } from 'react-window';

function FileList({ files }: { files: FileObject[] }) {
  const Row = ({ index, style }: { index: number; style: React.CSSProperties }) => {
    const file = files[index];
    return (
      <div style={style} className="flex items-center px-4 hover:bg-gray-50">
        <FileIcon type={file.contentType} />
        <span className="ml-2">{file.name}</span>
        <span className="ml-auto text-gray-500">{formatSize(file.size)}</span>
      </div>
    );
  };

  return (
    <List
      height={600}
      itemCount={files.length}
      itemSize={48} // 每行高度 48px
      width="100%"
    >
      {Row}
    </List>
  );
}
```

### 5.2 图片懒加载和缩略图

**懒加载策略**：
```tsx
import { useInView } from 'react-intersection-observer';

function LazyImage({ src, alt }: { src: string; alt: string }) {
  const { ref, inView } = useInView({
    triggerOnce: true, // 仅触发一次
    threshold: 0.1,
  });

  return (
    <div ref={ref} className="relative w-full h-48 bg-gray-200">
      {inView ? (
        <img
          src={src}
          alt={alt}
          loading="lazy"
          className="w-full h-full object-cover"
        />
      ) : (
        <Skeleton className="w-full h-full" />
      )}
    </div>
  );
}
```

**缩略图生成**：
- **方案 A**：MinIO Image Transformation（需要启用 `mc admin config set myminio api image_service=on`）
- **方案 B**：客户端缩放（使用 Canvas API，适合少量图片）
- **方案 C**：独立缩略图服务（使用 sharp/ImageMagick，适合大量图片）

**推荐方案 A**：
```typescript
function getThumbnailUrl(presignedUrl: string, width: number, height: number): string {
  const url = new URL(presignedUrl);
  url.searchParams.set('width', width.toString());
  url.searchParams.set('height', height.toString());
  url.searchParams.set('mode', 'contain'); // 保持宽高比
  return url.toString();
}
```

### 5.3 Presigned URL 缓存

**缓存策略**：
- **短期缓存**：15 分钟（预览链接），使用 React Query
- **长期缓存**：7 天（分享链接），存储在数据库

**React Query 配置**：
```typescript
function useFilePreviewUrl(bucket: string, objectKey: string) {
  return useQuery({
    queryKey: ['preview-url', bucket, objectKey],
    queryFn: () => minioApi.getPreviewUrl(bucket, objectKey),
    staleTime: 10 * 60 * 1000, // 10 分钟内认为数据新鲜
    cacheTime: 15 * 60 * 1000, // 15 分钟后清除缓存
    refetchOnWindowFocus: false, // 窗口聚焦时不重新获取
  });
}
```

**避免频繁签名**：
```go
type PresignedCache struct {
    cache *cache.Cache // ttlcache
}

func (c *PresignedCache) GetOrCreate(ctx context.Context, bucket, object string, expires time.Duration) (string, error) {
    key := fmt.Sprintf("%s/%s", bucket, object)

    // 从缓存获取
    if cached, found := c.cache.Get(key); found {
        return cached.(string), nil
    }

    // 生成新 URL
    url, err := generatePresignedURL(ctx, bucket, object, expires)
    if err != nil {
        return "", err
    }

    // 缓存 URL（过期时间为 URL 过期时间的 80%）
    c.cache.Set(key, url, expires*8/10)
    return url, nil
}
```

## 6. 集成测试环境

### 决策：testcontainers-go + MinIO 官方镜像

**选择理由**：
- testcontainers-go 提供 Go 原生的容器管理 API
- MinIO 官方镜像轻量级（~80MB），启动快（< 5s）
- 支持自定义配置（根用户、Bucket、策略）
- 测试隔离，每个测试套件独立容器

### MinIO 容器配置

**启动配置**：
```go
package miniotest

import (
    "context"
    "fmt"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"
)

type MinIOContainer struct {
    Container testcontainers.Container
    Endpoint  string
    AccessKey string
    SecretKey string
}

func StartMinIOContainer(ctx context.Context) (*MinIOContainer, error) {
    accessKey := "minioadmin"
    secretKey := "minioadmin"

    req := testcontainers.ContainerRequest{
        Image:        "minio/minio:latest",
        ExposedPorts: []string{"9000/tcp"},
        Env: map[string]string{
            "MINIO_ROOT_USER":     accessKey,
            "MINIO_ROOT_PASSWORD": secretKey,
        },
        Cmd: []string{"server", "/data"},
        WaitingFor: wait.ForHTTP("/minio/health/live").
            WithPort("9000/tcp").
            WithStartupTimeout(30 * time.Second),
    }

    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    if err != nil {
        return nil, err
    }

    host, err := container.Host(ctx)
    if err != nil {
        return nil, err
    }

    port, err := container.MappedPort(ctx, "9000")
    if err != nil {
        return nil, err
    }

    return &MinIOContainer{
        Container: container,
        Endpoint:  fmt.Sprintf("%s:%s", host, port.Port()),
        AccessKey: accessKey,
        SecretKey: secretKey,
    }, nil
}

func (c *MinIOContainer) Terminate(ctx context.Context) error {
    return c.Container.Terminate(ctx)
}
```

### 测试数据初始化

**初始化脚本**：
```go
func SetupTestData(ctx context.Context, minioContainer *MinIOContainer) error {
    client, err := minio.New(minioContainer.Endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(minioContainer.AccessKey, minioContainer.SecretKey, ""),
        Secure: false,
    })
    if err != nil {
        return err
    }

    // 1. 创建测试 Bucket
    buckets := []string{"test-bucket", "team-a", "team-b"}
    for _, bucket := range buckets {
        err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
        if err != nil {
            return fmt.Errorf("create bucket %s: %w", bucket, err)
        }
    }

    // 2. 上传测试文件
    testFiles := map[string]string{
        "test.txt":    "Hello, MinIO!",
        "test.jpg":    loadTestImage(),
        "test.mp4":    loadTestVideo(),
        "README.md":   "# Test README",
        "script.py":   "print('Hello')",
    }

    for filename, content := range testFiles {
        _, err = client.PutObject(ctx, "test-bucket", filename,
            strings.NewReader(content), int64(len(content)),
            minio.PutObjectOptions{})
        if err != nil {
            return fmt.Errorf("upload file %s: %w", filename, err)
        }
    }

    // 3. 创建测试用户
    testUser := "testuser"
    testUserSecret := "testuser123"

    err = client.AddUser(ctx, testUser, testUserSecret)
    if err != nil {
        return fmt.Errorf("create user: %w", err)
    }

    // 4. 设置测试策略
    policy := generateReadOnlyPolicy(testUser, "test-bucket")
    err = client.SetBucketPolicy(ctx, "test-bucket", policy)
    if err != nil {
        return fmt.Errorf("set policy: %w", err)
    }

    return nil
}
```

### 清理策略

**测试后清理**：
```go
func TestWithMinIO(t *testing.T) {
    ctx := context.Background()

    // 启动容器
    minioContainer, err := StartMinIOContainer(ctx)
    require.NoError(t, err)
    defer minioContainer.Terminate(ctx) // 确保测试后关闭容器

    // 初始化测试数据
    err = SetupTestData(ctx, minioContainer)
    require.NoError(t, err)

    // 运行测试
    t.Run("CreateBucket", func(t *testing.T) {
        // 测试逻辑
    })

    t.Run("UploadFile", func(t *testing.T) {
        // 测试逻辑
    })
}
```

**并行测试支持**：
```go
func TestParallel(t *testing.T) {
    t.Run("TestA", func(t *testing.T) {
        t.Parallel()
        minioA, _ := StartMinIOContainer(context.Background())
        defer minioA.Terminate(context.Background())
        // 测试 A
    })

    t.Run("TestB", func(t *testing.T) {
        t.Parallel()
        minioB, _ := StartMinIOContainer(context.Background())
        defer minioB.Terminate(context.Background())
        // 测试 B
    })
}
```

## 研究总结

### 关键决策汇总

| 方面 | 决策 | 理由 |
|------|------|------|
| MinIO SDK | MinIO Go SDK v7 | 官方支持，功能完整，文档完善 |
| 文件上传 | tus-js-client | 断点续传标准，稳定可靠 |
| 图片预览 | react-photo-view | 轻量级，体验好，兼容 Radix UI |
| 视频播放 | 原生 video | 浏览器原生支持，HTTP Range 渐进式加载 |
| 代码高亮 | Monaco Editor | VS Code 同款，语法高亮最佳 |
| Markdown 渲染 | react-markdown + remark-gfm | 官方推荐，安全，支持 GitHub 风格 |
| 虚拟滚动 | react-window | 轻量级，性能优秀 |
| 集成测试 | testcontainers-go + MinIO | 容器化测试，隔离性好，可重复 |

### 风险评估

**低风险**：
- MinIO SDK 和 testcontainers-go：成熟稳定，文档完善
- 前端预览组件：开源项目活跃，社区支持好

**中等风险**：
- 大文件上传（GB 级）：需要充分测试断点续传和错误恢复
- 前缀级权限：MinIO 策略语法复杂，需要验证各种边界情况

**缓解措施**：
- 编写全面的集成测试（testcontainers MinIO）
- 在 quickstart.md 中包含手动验证步骤
- 在 data-model.md 中明确策略 JSON 结构

### 下一步

所有技术选型已完成，无需进一步研究。可以进入阶段 1：设计与契约。

---
*研究完成日期：2025-10-14*
*参考文档：[plan.md](./plan.md)、[spec.md](./spec.md)*
