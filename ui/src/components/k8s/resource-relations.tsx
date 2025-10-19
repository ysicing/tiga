import { useResourceRelations, ResourceRelation } from '@/services/k8s-api'
import { Loader2, ChevronRight, ArrowRight, ArrowUp, Link as LinkIcon } from 'lucide-react'
import { useNavigate } from 'react-router-dom'

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'

interface ResourceRelationsProps {
  clusterId: string
  namespace: string
  kind: string
  name: string
  className?: string
}

export function ResourceRelations({
  clusterId,
  namespace,
  kind,
  name,
  className,
}: ResourceRelationsProps) {
  const { data, isLoading, error } = useResourceRelations(
    clusterId,
    namespace,
    kind,
    name
  )

  const relations = data || []

  // Separate relations by type
  const owners = relations.filter((r: ResourceRelation) => r.type === 'owner')
  const owned = relations.filter((r: ResourceRelation) => r.type === 'owned')
  const references = relations.filter((r: ResourceRelation) => r.type === 'reference')

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>资源关系</CardTitle>
          <CardDescription>查看与此资源相关的其他资源</CardDescription>
        </CardHeader>
        <CardContent className="flex items-center justify-center py-12">
          <div className="text-center">
            <Loader2 className="mx-auto h-8 w-8 animate-spin text-muted-foreground" />
            <p className="text-sm text-muted-foreground mt-2">加载关系图...</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>资源关系</CardTitle>
          <CardDescription>查看与此资源相关的其他资源</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8">
            <p className="text-sm text-muted-foreground">
              无法加载资源关系：{error instanceof Error ? error.message : '未知错误'}
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (relations.length === 0) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>资源关系</CardTitle>
          <CardDescription>查看与此资源相关的其他资源</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8">
            <LinkIcon className="mx-auto h-12 w-12 text-muted-foreground mb-2" />
            <p className="text-sm text-muted-foreground">
              未找到相关资源
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={className}>
      <CardHeader>
        <CardTitle>资源关系</CardTitle>
        <CardDescription>
          显示与 {kind}/{name} 相关的资源
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Owners */}
        {owners.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium">
              <ArrowUp className="h-4 w-4" />
              <span>父资源 ({owners.length})</span>
            </div>
            <div className="ml-6 space-y-1">
              {owners.map((owner: ResourceRelation, idx: number) => (
                <RelationItem key={idx} relation={owner} clusterId={clusterId} />
              ))}
            </div>
          </div>
        )}

        {/* Current Resource */}
        <div className="flex items-center gap-2 p-3 bg-primary/10 border-l-4 border-primary rounded">
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <span className="font-medium">{kind}</span>
              <Badge variant="secondary">{name}</Badge>
              {namespace && (
                <Badge variant="outline" className="text-xs">
                  {namespace}
                </Badge>
              )}
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              当前资源
            </p>
          </div>
        </div>

        {/* Owned Resources */}
        {owned.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium">
              <ArrowRight className="h-4 w-4" />
              <span>子资源 ({owned.length})</span>
            </div>
            <div className="ml-6 space-y-1">
              {owned.map((child: ResourceRelation, idx: number) => (
                <RelationItem key={idx} relation={child} clusterId={clusterId} />
              ))}
            </div>
          </div>
        )}

        {/* References */}
        {references.length > 0 && (
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium">
              <LinkIcon className="h-4 w-4" />
              <span>引用资源 ({references.length})</span>
            </div>
            <div className="ml-6 space-y-1">
              {references.map((ref: ResourceRelation, idx: number) => (
                <RelationItem key={idx} relation={ref} clusterId={clusterId} />
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

interface RelationItemProps {
  relation: ResourceRelation
  clusterId: string
}

function RelationItem({ relation }: RelationItemProps) {
  const navigate = useNavigate()

  const handleClick = () => {
    const kind = relation.kind.toLowerCase()
    const namespace = relation.namespace || 'default'
    const name = relation.name

    // Navigate to resource detail page
    navigate(`/k8s/${kind}s/${namespace}/${name}`)
  }

  return (
    <div
      className="flex items-center gap-2 p-2 hover:bg-accent rounded cursor-pointer transition-colors"
      onClick={handleClick}
    >
      <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{relation.kind}</span>
          <span className="text-sm text-muted-foreground truncate">
            {relation.name}
          </span>
        </div>
      </div>
      {relation.namespace && (
        <Badge variant="outline" className="text-xs shrink-0">
          {relation.namespace}
        </Badge>
      )}
    </div>
  )
}
