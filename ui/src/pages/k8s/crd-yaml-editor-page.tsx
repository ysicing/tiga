import { useState } from 'react'
import { useApplyCRDResourceYAML } from '@/services/k8s-api'
import { IconCheck, IconCode } from '@tabler/icons-react'
import Editor from '@monaco-editor/react'
import { useParams } from 'react-router-dom'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

const EXAMPLE_DEPLOYMENT = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80`

const EXAMPLE_SERVICE = `apiVersion: v1
kind: Service
metadata:
  name: nginx-service
  namespace: default
spec:
  selector:
    app: nginx
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80
  type: ClusterIP`

const EXAMPLE_CONFIGMAP = `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-config
  namespace: default
data:
  config.yml: |
    key: value
    nested:
      key: value`

export function CRDYAMLEditorPage() {
  const { clusterId } = useParams<{ clusterId: string }>()
  const [yaml, setYaml] = useState<string>('')
  const [activeTab, setActiveTab] = useState<string>('editor')

  const applyMutation = useApplyCRDResourceYAML()

  const handleApply = async () => {
    if (!clusterId) {
      toast.error('缺少集群 ID')
      return
    }

    if (!yaml.trim()) {
      toast.error('请输入 YAML 内容')
      return
    }

    try {
      const result = await applyMutation.mutateAsync({ clusterId, yaml })
      const data = result.data
      toast.success(
        `资源已应用: ${data.kind}/${data.name}${data.namespace ? ` (${data.namespace})` : ''}`
      )
      // Don't clear YAML after successful apply, user might want to modify and reapply
    } catch (err) {
      const error = err as { response?: { data?: { error?: string } } }
      toast.error(
        `应用失败: ${error?.response?.data?.error || (err as Error).message}`
      )
    }
  }

  const loadExample = (example: string) => {
    setYaml(example)
    setActiveTab('editor')
  }

  if (!clusterId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-destructive">缺少集群 ID</CardTitle>
          <CardDescription>请从集群列表选择一个集群</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Kubernetes YAML 编辑器</h1>
          <p className="text-muted-foreground mt-2">
            创建和应用 Kubernetes 资源
          </p>
        </div>
        <Button
          onClick={handleApply}
          disabled={applyMutation.isPending || !yaml.trim()}
        >
          <IconCheck className="w-4 h-4 mr-2" />
          {applyMutation.isPending ? '应用中...' : '应用 YAML'}
        </Button>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="editor">
            <IconCode className="w-4 h-4 mr-2" />
            编辑器
          </TabsTrigger>
          <TabsTrigger value="examples">示例模板</TabsTrigger>
        </TabsList>

        <TabsContent value="editor" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>YAML 编辑器</CardTitle>
              <CardDescription>
                编写或粘贴 Kubernetes YAML 配置
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="border rounded-md overflow-hidden">
                <Editor
                  height="600px"
                  defaultLanguage="yaml"
                  value={yaml}
                  onChange={(value) => setYaml(value || '')}
                  theme="vs-dark"
                  options={{
                    minimap: { enabled: false },
                    fontSize: 14,
                    lineNumbers: 'on',
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                    tabSize: 2,
                  }}
                />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="examples" className="space-y-4">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            <Card className="cursor-pointer hover:shadow-lg transition-shadow">
              <CardHeader>
                <CardTitle className="text-lg">Deployment</CardTitle>
                <CardDescription>
                  创建一个基本的 Nginx Deployment
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => loadExample(EXAMPLE_DEPLOYMENT)}
                >
                  使用此模板
                </Button>
              </CardContent>
            </Card>

            <Card className="cursor-pointer hover:shadow-lg transition-shadow">
              <CardHeader>
                <CardTitle className="text-lg">Service</CardTitle>
                <CardDescription>创建一个 ClusterIP Service</CardDescription>
              </CardHeader>
              <CardContent>
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => loadExample(EXAMPLE_SERVICE)}
                >
                  使用此模板
                </Button>
              </CardContent>
            </Card>

            <Card className="cursor-pointer hover:shadow-lg transition-shadow">
              <CardHeader>
                <CardTitle className="text-lg">ConfigMap</CardTitle>
                <CardDescription>创建一个配置映射</CardDescription>
              </CardHeader>
              <CardContent>
                <Button
                  variant="outline"
                  className="w-full"
                  onClick={() => loadExample(EXAMPLE_CONFIGMAP)}
                >
                  使用此模板
                </Button>
              </CardContent>
            </Card>
          </div>

          <Card>
            <CardHeader>
              <CardTitle>使用说明</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-sm text-muted-foreground">
              <p>• 选择一个示例模板开始编辑</p>
              <p>• 修改 YAML 配置以满足您的需求</p>
              <p>• 点击"应用 YAML"按钮创建或更新资源</p>
              <p>• 支持所有标准 Kubernetes 资源类型和 CRDs</p>
              <p>
                • 如果资源已存在，将执行更新操作（类似 kubectl apply）
              </p>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
