import { useState, useEffect } from 'react'
import { Search, Loader2, FileText, Database, Server, Package, Filter, X } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'

import { useClusterSearch, SearchFilters, SearchResult } from '@/services/k8s-api'
import { useCluster } from '@/contexts/cluster-context'
import { Cluster } from '@/types/api'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { ClusterSelector } from '@/components/k8s/cluster-selector'

const RESOURCE_TYPES = [
  { value: 'Pod', label: 'Pods', icon: Package },
  { value: 'Deployment', label: 'Deployments', icon: Server },
  { value: 'Service', label: 'Services', icon: Database },
  { value: 'ConfigMap', label: 'ConfigMaps', icon: FileText },
]

const getMatchTypeBadge = (matchType: string) => {
  switch (matchType) {
    case 'exact':
      return <Badge className="bg-green-600">精确匹配</Badge>
    case 'name':
      return <Badge className="bg-blue-600">名称匹配</Badge>
    case 'label':
      return <Badge className="bg-purple-600">标签匹配</Badge>
    case 'annotation':
      return <Badge variant="secondary">注解匹配</Badge>
    default:
      return <Badge variant="outline">{matchType}</Badge>
  }
}

const getResourceIcon = (kind: string) => {
  const resourceType = RESOURCE_TYPES.find((r) => r.value === kind)
  return resourceType ? resourceType.icon : Package
}

export function SearchPage() {
  const navigate = useNavigate()
  const { currentCluster, clusters } = useCluster()

  const [searchQuery, setSearchQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [selectedTypes, setSelectedTypes] = useState<string[]>([])
  const [selectedNamespace, setSelectedNamespace] = useState<string>('')
  const [showFilters, setShowFilters] = useState(false)

  const cluster = clusters.find((c: Cluster) => c.name === currentCluster)

  // Debounce search query
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedQuery(searchQuery)
    }, 300)

    return () => clearTimeout(timer)
  }, [searchQuery])

  const filters: SearchFilters = {
    q: debouncedQuery,
    types: selectedTypes.length > 0 ? selectedTypes : undefined,
    namespace: selectedNamespace || undefined,
    limit: 50,
  }

  const { data, isLoading, error } = useClusterSearch(
    cluster?.id ? String(cluster.id) : undefined,
    filters,
    debouncedQuery.length > 0
  )

  const results = data || []

  // Group results by kind
  const resultsByKind = results.reduce((acc: Record<string, SearchResult[]>, result: SearchResult) => {
    if (!acc[result.kind]) {
      acc[result.kind] = []
    }
    acc[result.kind].push(result)
    return acc
  }, {} as Record<string, SearchResult[]>)

  const handleTypeToggle = (type: string) => {
    setSelectedTypes((prev) =>
      prev.includes(type)
        ? prev.filter((t) => t !== type)
        : [...prev, type]
    )
  }

  const handleClearFilters = () => {
    setSelectedTypes([])
    setSelectedNamespace('')
  }

  const handleResultClick = (result: SearchResult) => {
    const kind = result.kind.toLowerCase()
    const namespace = result.namespace || 'default'
    const name = result.name

    // Navigate to resource detail page
    navigate(`/k8s/${kind}s/${namespace}/${name}`)
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold">全局搜索</h1>
        <p className="text-muted-foreground mt-1">
          在集群中搜索 Kubernetes 资源
        </p>
      </div>

      {/* Cluster Selector */}
      <div className="flex items-center gap-4">
        <Label className="text-sm font-medium shrink-0">当前集群:</Label>
        <ClusterSelector />
      </div>

      {/* Search Bar */}
      <Card>
        <CardContent className="pt-6">
          <div className="space-y-4">
            <div className="flex gap-2">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="搜索资源（名称、标签、注解...）"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9"
                />
              </div>
              <Button
                variant={showFilters ? 'default' : 'outline'}
                onClick={() => setShowFilters(!showFilters)}
              >
                <Filter className="mr-2 h-4 w-4" />
                过滤器
              </Button>
              {(selectedTypes.length > 0 || selectedNamespace) && (
                <Button variant="ghost" onClick={handleClearFilters}>
                  <X className="mr-2 h-4 w-4" />
                  清除
                </Button>
              )}
            </div>

            {/* Filters */}
            {showFilters && (
              <div className="grid gap-4 md:grid-cols-2 border-t pt-4">
                <div className="space-y-2">
                  <Label>资源类型</Label>
                  <div className="grid grid-cols-2 gap-2">
                    {RESOURCE_TYPES.map((type) => (
                      <div
                        key={type.value}
                        className="flex items-center space-x-2"
                      >
                        <Checkbox
                          id={`type-${type.value}`}
                          checked={selectedTypes.includes(type.value)}
                          onCheckedChange={() => handleTypeToggle(type.value)}
                        />
                        <Label
                          htmlFor={`type-${type.value}`}
                          className="text-sm font-normal cursor-pointer"
                        >
                          {type.label}
                        </Label>
                      </div>
                    ))}
                  </div>
                </div>

                <div className="space-y-2">
                  <Label>命名空间</Label>
                  <Select
                    value={selectedNamespace}
                    onValueChange={setSelectedNamespace}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="所有命名空间" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="">所有命名空间</SelectItem>
                      <SelectItem value="default">default</SelectItem>
                      <SelectItem value="kube-system">kube-system</SelectItem>
                      <SelectItem value="kube-public">kube-public</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Search Results */}
      {!cluster ? (
        <Card>
          <CardContent className="flex items-center justify-center py-12">
            <div className="text-center">
              <Server className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold">未选择集群</h3>
              <p className="text-sm text-muted-foreground mt-1">
                请选择一个集群以开始搜索
              </p>
            </div>
          </CardContent>
        </Card>
      ) : !debouncedQuery ? (
        <Card>
          <CardContent className="flex items-center justify-center py-12">
            <div className="text-center">
              <Search className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold">开始搜索</h3>
              <p className="text-sm text-muted-foreground mt-1">
                输入关键词以搜索资源
              </p>
            </div>
          </CardContent>
        </Card>
      ) : isLoading ? (
        <Card>
          <CardContent className="flex items-center justify-center py-12">
            <div className="text-center">
              <Loader2 className="mx-auto h-12 w-12 text-muted-foreground animate-spin mb-4" />
              <h3 className="text-lg font-semibold">搜索中...</h3>
              <p className="text-sm text-muted-foreground mt-1">
                正在搜索集群 {cluster.name}
              </p>
            </div>
          </CardContent>
        </Card>
      ) : error ? (
        <Card>
          <CardContent className="py-12">
            <div className="text-center">
              <X className="mx-auto h-12 w-12 text-destructive mb-4" />
              <h3 className="text-lg font-semibold">搜索失败</h3>
              <p className="text-sm text-muted-foreground mt-1">
                {error instanceof Error ? error.message : '未知错误'}
              </p>
            </div>
          </CardContent>
        </Card>
      ) : results.length === 0 ? (
        <Card>
          <CardContent className="flex items-center justify-center py-12">
            <div className="text-center">
              <FileText className="mx-auto h-12 w-12 text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold">未找到结果</h3>
              <p className="text-sm text-muted-foreground mt-1">
                尝试使用不同的关键词或过滤器
              </p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-6">
          {/* Results Summary */}
          <div className="flex items-center justify-between">
            <h2 className="text-xl font-semibold">
              搜索结果 <Badge variant="secondary">{results.length}</Badge>
            </h2>
          </div>

          {/* Grouped Results */}
          {Object.entries(resultsByKind).map(([kind, kindResults]) => {
            const Icon = getResourceIcon(kind)
            const typedResults = kindResults as SearchResult[]

            return (
              <Card key={kind}>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Icon className="h-5 w-5" />
                    {kind}
                    <Badge variant="outline">{typedResults.length}</Badge>
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {typedResults.map((result: SearchResult, idx: number) => (
                      <div
                        key={idx}
                        className="flex items-center justify-between p-3 border rounded-lg hover:bg-accent cursor-pointer transition-colors"
                        onClick={() => handleResultClick(result)}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <h4 className="font-medium truncate">
                              {result.name}
                            </h4>
                            {result.namespace && (
                              <Badge variant="secondary" className="shrink-0">
                                {result.namespace}
                              </Badge>
                            )}
                            {getMatchTypeBadge(result.match_type)}
                          </div>
                          {result.labels && Object.keys(result.labels).length > 0 && (
                            <div className="flex flex-wrap gap-1 mt-1">
                              {Object.entries(result.labels).slice(0, 3).map(([key, value]) => (
                                <Badge
                                  key={key}
                                  variant="outline"
                                  className="text-xs"
                                >
                                  {key}={value}
                                </Badge>
                              ))}
                              {Object.keys(result.labels).length > 3 && (
                                <Badge variant="outline" className="text-xs">
                                  +{Object.keys(result.labels).length - 3}
                                </Badge>
                              )}
                            </div>
                          )}
                        </div>
                        <div className="text-sm text-muted-foreground shrink-0 ml-4">
                          {formatDistanceToNow(new Date(result.created), {
                            addSuffix: true,
                            locale: zhCN,
                          })}
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
