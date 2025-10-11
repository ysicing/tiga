import { useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { IconPlayerPlay, IconAlertCircle, IconCheck } from '@tabler/icons-react'
import { useExecuteQuery } from '@/services/database-api'
import { toast } from 'sonner'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

interface QueryConsoleProps {
  instanceId: number
  instanceType: string
}

export function QueryConsole({ instanceId, instanceType }: QueryConsoleProps) {
  const [query, setQuery] = useState('')
  const [database, setDatabase] = useState('')
  const [result, setResult] = useState<any>(null)

  const executeMutation = useExecuteQuery()

  const handleExecute = async () => {
    if (!query.trim()) {
      toast.error('请输入查询语句')
      return
    }

    if (!database && instanceType !== 'redis') {
      toast.error('请选择数据库')
      return
    }

    try {
      const res = await executeMutation.mutateAsync({
        instanceId,
        databaseName: database,
        query: query.trim(),
      })
      setResult(res.data)
      toast.success(`查询已执行，返回 ${res.data.row_count} 行`)
    } catch (error: any) {
      toast.error(error?.response?.data?.error || '查询执行出错')
      setResult({ error: error?.response?.data?.error || '查询执行出错' })
    }
  }

  const placeholder = instanceType === 'redis'
    ? 'GET key\nSET key value\nKEYS *'
    : instanceType === 'mysql'
    ? 'SELECT * FROM users LIMIT 10;\n\n-- DDL 操作已禁用'
    : 'SELECT * FROM users LIMIT 10;\n\n-- DDL operations are forbidden'

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>SQL 查询控制台</CardTitle>
          <CardDescription>
            执行 {instanceType.toUpperCase()} 查询。注意：DDL 操作和无 WHERE 条件的 UPDATE/DELETE 已被禁止。
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {instanceType !== 'redis' && (
            <div className="space-y-2">
              <label className="text-sm font-medium">数据库</label>
              <Select value={database} onValueChange={setDatabase}>
                <SelectTrigger>
                  <SelectValue placeholder="选择数据库" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="mysql">mysql</SelectItem>
                  <SelectItem value="information_schema">information_schema</SelectItem>
                </SelectContent>
              </Select>
            </div>
          )}

          <div className="space-y-2">
            <label className="text-sm font-medium">查询语句</label>
            <Textarea
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={placeholder}
              className="font-mono min-h-[200px]"
            />
          </div>

          <Button onClick={handleExecute} disabled={executeMutation.isPending}>
            <IconPlayerPlay className="w-4 h-4 mr-2" />
            {executeMutation.isPending ? '执行中...' : '执行'}
          </Button>
        </CardContent>
      </Card>

      {/* Results */}
      {result && (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>查询结果</CardTitle>
              {result.error ? (
                <Badge variant="destructive">
                  <IconAlertCircle className="w-3 h-3 mr-1" />
                  错误
                </Badge>
              ) : (
                <Badge className="bg-green-500">
                  <IconCheck className="w-3 h-3 mr-1" />
                  成功
                </Badge>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {result.error ? (
              <div className="text-destructive">{result.error}</div>
            ) : (
              <div className="space-y-4">
                <div className="text-sm text-muted-foreground">
                  返回 {result.row_count} 行 · 耗时 {result.execution_time}ms
                  {result.truncated && ' · 结果已截断 (超过 10MB 限制)'}
                </div>

                {result.rows && result.rows.length > 0 && (
                  <div className="overflow-auto max-h-[400px]">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          {result.columns?.map((col: string) => (
                            <TableHead key={col}>{col}</TableHead>
                          ))}
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {result.rows.map((row: any, idx: number) => (
                          <TableRow key={idx}>
                            {result.columns?.map((col: string) => (
                              <TableCell key={col}>{String(row[col] ?? '')}</TableCell>
                            ))}
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
