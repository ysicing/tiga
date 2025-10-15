import { useEffect, useState } from 'react'
import Editor from '@monaco-editor/react'
import {
  ArrowLeft,
  Check,
  Copy,
  Database,
  Download,
  Key,
  MoreHorizontal,
  Play,
  Plus,
  RefreshCw,
  Search,
  Trash2,
  User,
} from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'

import { devopsAPI } from '@/lib/api-client'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

interface DatabaseInfo {
  name: string
  charset?: string
  collation?: string
  size?: string
  table_count?: number
}

interface UserInfo {
  username: string
  host: string
  privileges?: string[]
}

interface QueryResult {
  columns: string[]
  rows: any[][]
  affected_rows?: number
  execution_time?: number
  error?: string
}

export default function DatabaseManagementPage() {
  const { instanceId } = useParams<{ instanceId: string }>()
  const navigate = useNavigate()

  // State
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [users, setUsers] = useState<UserInfo[]>([])
  const [selectedDatabase, setSelectedDatabase] = useState<string>('')
  const [sqlQuery, setSqlQuery] = useState<string>(
    'SELECT * FROM information_schema.tables LIMIT 10;'
  )
  const [queryResult, setQueryResult] = useState<QueryResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [executing, setExecuting] = useState(false)
  const [searchTerm, setSearchTerm] = useState('')
  const [copiedQuery, setCopiedQuery] = useState(false)

  // Dialog states
  const [createDbOpen, setCreateDbOpen] = useState(false)
  const [createUserOpen, setCreateUserOpen] = useState(false)
  const [newDbName, setNewDbName] = useState('')
  const [newDbCharset, setNewDbCharset] = useState('utf8mb4')
  const [newDbCollation, setNewDbCollation] = useState('utf8mb4_unicode_ci')
  const [newUsername, setNewUsername] = useState('')
  const [newUserPassword, setNewUserPassword] = useState('')
  const [newUserHost, setNewUserHost] = useState('%')

  useEffect(() => {
    if (instanceId) {
      loadDatabases()
      loadUsers()
    }
  }, [instanceId])

  const loadDatabases = async () => {
    if (!instanceId) return
    setLoading(true)
    try {
      const response = (await devopsAPI.database.listDatabases(
        instanceId
      )) as any
      setDatabases(response.databases || [])
      if (response.databases?.length > 0 && !selectedDatabase) {
        setSelectedDatabase(response.databases[0].name)
      }
    } catch (error: any) {
      toast.error('Failed to load databases', {
        description: error.message,
      })
    } finally {
      setLoading(false)
    }
  }

  const loadUsers = async () => {
    if (!instanceId) return
    try {
      const response = (await devopsAPI.database.listUsers(instanceId)) as any
      setUsers(response.users || [])
    } catch (error: any) {
      toast.error('Failed to load users', {
        description: error.message,
      })
    }
  }

  const handleCreateDatabase = async () => {
    if (!instanceId || !newDbName) return

    try {
      await devopsAPI.database.createDatabase(
        instanceId,
        newDbName,
        newDbCharset,
        newDbCollation
      )
      toast.success('Database created successfully')
      setNewDbName('')
      setCreateDbOpen(false)
      loadDatabases()
    } catch (error: any) {
      toast.error('Failed to create database', {
        description: error.message,
      })
    }
  }

  const handleDeleteDatabase = async (dbName: string) => {
    if (!instanceId) return

    if (!confirm(`Are you sure you want to delete database "${dbName}"?`)) {
      return
    }

    try {
      await devopsAPI.database.deleteDatabase(instanceId, dbName)
      toast.success('Database deleted successfully')
      if (selectedDatabase === dbName) {
        setSelectedDatabase('')
      }
      loadDatabases()
    } catch (error: any) {
      toast.error('Failed to delete database', {
        description: error.message,
      })
    }
  }

  const handleCreateUser = async () => {
    if (!instanceId || !newUsername || !newUserPassword) return

    try {
      await devopsAPI.database.createUser(
        instanceId,
        newUsername,
        newUserPassword,
        newUserHost
      )
      toast.success('User created successfully')
      setNewUsername('')
      setNewUserPassword('')
      setNewUserHost('%')
      setCreateUserOpen(false)
      loadUsers()
    } catch (error: any) {
      toast.error('Failed to create user', {
        description: error.message,
      })
    }
  }

  const handleDeleteUser = async (username: string) => {
    if (!instanceId) return

    if (!confirm(`Are you sure you want to delete user "${username}"?`)) {
      return
    }

    try {
      await devopsAPI.database.deleteUser(instanceId, username)
      toast.success('User deleted successfully')
      loadUsers()
    } catch (error: any) {
      toast.error('Failed to delete user', {
        description: error.message,
      })
    }
  }

  const handleExecuteQuery = async () => {
    if (!instanceId || !selectedDatabase || !sqlQuery.trim()) {
      toast.error('Please select a database and enter a query')
      return
    }

    setExecuting(true)
    setQueryResult(null)

    try {
      const response = (await devopsAPI.database.executeQuery(
        instanceId,
        selectedDatabase,
        sqlQuery
      )) as any
      setQueryResult(response)

      if (response.error) {
        toast.error('Query execution failed', {
          description: response.error,
        })
      } else {
        toast.success('Query executed successfully', {
          description: `Execution time: ${response.execution_time?.toFixed(2)}ms`,
        })
      }
    } catch (error: any) {
      toast.error('Failed to execute query', {
        description: error.message,
      })
      setQueryResult({
        columns: [],
        rows: [],
        error: error.message,
      })
    } finally {
      setExecuting(false)
    }
  }

  const handleCopyQuery = () => {
    navigator.clipboard.writeText(sqlQuery)
    setCopiedQuery(true)
    setTimeout(() => setCopiedQuery(false), 2000)
    toast.success('Query copied to clipboard')
  }

  const handleExportResults = () => {
    if (!queryResult || queryResult.rows.length === 0) return

    // Convert to CSV
    const csv = [
      queryResult.columns.join(','),
      ...queryResult.rows.map((row) =>
        row
          .map((cell) =>
            typeof cell === 'string' && cell.includes(',') ? `"${cell}"` : cell
          )
          .join(',')
      ),
    ].join('\n')

    const blob = new Blob([csv], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `query-result-${new Date().toISOString()}.csv`
    a.click()
    URL.revokeObjectURL(url)
    toast.success('Results exported')
  }

  const filteredDatabases = databases.filter((db) =>
    db.name.toLowerCase().includes(searchTerm.toLowerCase())
  )

  const filteredUsers = users.filter((user) =>
    user.username.toLowerCase().includes(searchTerm.toLowerCase())
  )

  // SQL Example queries
  const exampleQueries = [
    {
      name: 'Show Tables',
      query: 'SHOW TABLES;',
    },
    {
      name: 'Show Databases',
      query: 'SHOW DATABASES;',
    },
    {
      name: 'Table Info',
      query: 'SHOW TABLE STATUS;',
    },
    {
      name: 'Current User',
      query: 'SELECT USER(), CURRENT_USER();',
    },
  ]

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold">Database Management</h1>
            <p className="text-muted-foreground">Instance ID: {instanceId}</p>
          </div>
        </div>
        <Button
          onClick={() => {
            loadDatabases()
            loadUsers()
          }}
          variant="outline"
          size="icon"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      <Tabs defaultValue="query" className="space-y-4">
        <TabsList>
          <TabsTrigger value="query">SQL Editor</TabsTrigger>
          <TabsTrigger value="databases">Databases</TabsTrigger>
          <TabsTrigger value="users">Users</TabsTrigger>
        </TabsList>

        {/* SQL Editor Tab */}
        <TabsContent value="query" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>SQL Query Editor</CardTitle>
                  <CardDescription>
                    Execute SQL queries against your database
                  </CardDescription>
                </div>
                <div className="flex items-center gap-2">
                  <div className="flex items-center gap-2">
                    <Label htmlFor="db-select">Database:</Label>
                    <select
                      id="db-select"
                      value={selectedDatabase}
                      onChange={(e) => setSelectedDatabase(e.target.value)}
                      className="flex h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm transition-colors"
                    >
                      <option value="">Select database</option>
                      {databases.map((db) => (
                        <option key={db.name} value={db.name}>
                          {db.name}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Example Queries */}
              <div className="flex items-center gap-2 flex-wrap">
                <span className="text-sm text-muted-foreground">Examples:</span>
                {exampleQueries.map((example) => (
                  <Button
                    key={example.name}
                    variant="outline"
                    size="sm"
                    onClick={() => setSqlQuery(example.query)}
                  >
                    {example.name}
                  </Button>
                ))}
              </div>

              {/* SQL Editor */}
              <div className="border rounded-md overflow-hidden">
                <Editor
                  height="300px"
                  defaultLanguage="sql"
                  value={sqlQuery}
                  onChange={(value) => setSqlQuery(value || '')}
                  theme="vs-dark"
                  options={{
                    minimap: { enabled: false },
                    fontSize: 14,
                    lineNumbers: 'on',
                    scrollBeyondLastLine: false,
                    automaticLayout: true,
                  }}
                />
              </div>

              {/* Actions */}
              <div className="flex items-center justify-between">
                <div className="flex gap-2">
                  <Button
                    onClick={handleExecuteQuery}
                    disabled={!selectedDatabase || executing}
                  >
                    <Play className="h-4 w-4 mr-2" />
                    {executing ? 'Executing...' : 'Execute Query'}
                  </Button>
                  <Button variant="outline" onClick={handleCopyQuery}>
                    {copiedQuery ? (
                      <Check className="h-4 w-4 mr-2" />
                    ) : (
                      <Copy className="h-4 w-4 mr-2" />
                    )}
                    Copy
                  </Button>
                </div>
                {queryResult && queryResult.rows.length > 0 && (
                  <Button variant="outline" onClick={handleExportResults}>
                    <Download className="h-4 w-4 mr-2" />
                    Export CSV
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Query Results */}
          {queryResult && (
            <Card>
              <CardHeader>
                <CardTitle>
                  {queryResult.error ? (
                    <span className="text-destructive">Query Error</span>
                  ) : (
                    'Query Results'
                  )}
                </CardTitle>
                <CardDescription>
                  {queryResult.error
                    ? queryResult.error
                    : queryResult.affected_rows !== undefined
                      ? `${queryResult.affected_rows} row(s) affected`
                      : `${queryResult.rows.length} row(s) returned in ${queryResult.execution_time?.toFixed(2)}ms`}
                </CardDescription>
              </CardHeader>
              <CardContent>
                {!queryResult.error && queryResult.rows.length > 0 && (
                  <div className="border rounded-md max-h-[500px] overflow-auto">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          {queryResult.columns.map((col, index) => (
                            <TableHead key={index}>{col}</TableHead>
                          ))}
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {queryResult.rows.map((row, rowIndex) => (
                          <TableRow key={rowIndex}>
                            {row.map((cell, cellIndex) => (
                              <TableCell key={cellIndex}>
                                {cell === null ? (
                                  <span className="text-muted-foreground italic">
                                    NULL
                                  </span>
                                ) : (
                                  String(cell)
                                )}
                              </TableCell>
                            ))}
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                )}
                {!queryResult.error && queryResult.rows.length === 0 && (
                  <div className="text-center py-8 text-muted-foreground">
                    No results returned
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* Databases Tab */}
        <TabsContent value="databases">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Databases</CardTitle>
                  <CardDescription>
                    Manage databases on this instance
                  </CardDescription>
                </div>
                <Dialog open={createDbOpen} onOpenChange={setCreateDbOpen}>
                  <DialogTrigger asChild>
                    <Button>
                      <Plus className="h-4 w-4 mr-2" />
                      Create Database
                    </Button>
                  </DialogTrigger>
                  <DialogContent>
                    <DialogHeader>
                      <DialogTitle>Create New Database</DialogTitle>
                      <DialogDescription>
                        Enter details for the new database
                      </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4">
                      <div className="space-y-2">
                        <Label htmlFor="db-name">Database Name</Label>
                        <Input
                          id="db-name"
                          placeholder="my_database"
                          value={newDbName}
                          onChange={(e) => setNewDbName(e.target.value)}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="charset">Character Set</Label>
                        <Input
                          id="charset"
                          placeholder="utf8mb4"
                          value={newDbCharset}
                          onChange={(e) => setNewDbCharset(e.target.value)}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="collation">Collation</Label>
                        <Input
                          id="collation"
                          placeholder="utf8mb4_unicode_ci"
                          value={newDbCollation}
                          onChange={(e) => setNewDbCollation(e.target.value)}
                        />
                      </div>
                    </div>
                    <DialogFooter>
                      <Button
                        variant="outline"
                        onClick={() => setCreateDbOpen(false)}
                      >
                        Cancel
                      </Button>
                      <Button
                        onClick={handleCreateDatabase}
                        disabled={!newDbName}
                      >
                        Create
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="relative">
                  <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search databases..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="pl-8"
                  />
                </div>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Charset</TableHead>
                      <TableHead>Collation</TableHead>
                      <TableHead>Tables</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredDatabases.map((db) => (
                      <TableRow key={db.name}>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <Database className="h-4 w-4" />
                            <span className="font-medium">{db.name}</span>
                          </div>
                        </TableCell>
                        <TableCell>{db.charset || '-'}</TableCell>
                        <TableCell>{db.collation || '-'}</TableCell>
                        <TableCell>{db.table_count || '-'}</TableCell>
                        <TableCell className="text-right">
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuLabel>Actions</DropdownMenuLabel>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem
                                onClick={() => {
                                  setSelectedDatabase(db.name)
                                  navigate(`#query`)
                                }}
                              >
                                <Play className="h-4 w-4 mr-2" />
                                Query
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => handleDeleteDatabase(db.name)}
                                className="text-destructive"
                              >
                                <Trash2 className="h-4 w-4 mr-2" />
                                Delete
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))}
                    {filteredDatabases.length === 0 && (
                      <TableRow>
                        <TableCell
                          colSpan={5}
                          className="text-center py-8 text-muted-foreground"
                        >
                          No databases found
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Users Tab */}
        <TabsContent value="users">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Database Users</CardTitle>
                  <CardDescription>
                    Manage database user accounts
                  </CardDescription>
                </div>
                <Dialog open={createUserOpen} onOpenChange={setCreateUserOpen}>
                  <DialogTrigger asChild>
                    <Button>
                      <Plus className="h-4 w-4 mr-2" />
                      Create User
                    </Button>
                  </DialogTrigger>
                  <DialogContent>
                    <DialogHeader>
                      <DialogTitle>Create New User</DialogTitle>
                      <DialogDescription>
                        Enter details for the new database user
                      </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4">
                      <div className="space-y-2">
                        <Label htmlFor="username">Username</Label>
                        <Input
                          id="username"
                          placeholder="myuser"
                          value={newUsername}
                          onChange={(e) => setNewUsername(e.target.value)}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="password">Password</Label>
                        <Input
                          id="password"
                          type="password"
                          placeholder="••••••••"
                          value={newUserPassword}
                          onChange={(e) => setNewUserPassword(e.target.value)}
                        />
                      </div>
                      <div className="space-y-2">
                        <Label htmlFor="host">Host</Label>
                        <Input
                          id="host"
                          placeholder="%"
                          value={newUserHost}
                          onChange={(e) => setNewUserHost(e.target.value)}
                        />
                      </div>
                    </div>
                    <DialogFooter>
                      <Button
                        variant="outline"
                        onClick={() => setCreateUserOpen(false)}
                      >
                        Cancel
                      </Button>
                      <Button
                        onClick={handleCreateUser}
                        disabled={!newUsername || !newUserPassword}
                      >
                        Create
                      </Button>
                    </DialogFooter>
                  </DialogContent>
                </Dialog>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="relative">
                  <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search users..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="pl-8"
                  />
                </div>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Username</TableHead>
                      <TableHead>Host</TableHead>
                      <TableHead>Privileges</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {filteredUsers.map((user) => (
                      <TableRow key={`${user.username}@${user.host}`}>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <User className="h-4 w-4" />
                            <span className="font-medium">{user.username}</span>
                          </div>
                        </TableCell>
                        <TableCell>{user.host}</TableCell>
                        <TableCell>
                          {user.privileges && user.privileges.length > 0 ? (
                            <div className="flex gap-1 flex-wrap">
                              {user.privileges.slice(0, 3).map((priv, idx) => (
                                <Badge key={idx} variant="secondary">
                                  {priv}
                                </Badge>
                              ))}
                              {user.privileges.length > 3 && (
                                <Badge variant="outline">
                                  +{user.privileges.length - 3} more
                                </Badge>
                              )}
                            </div>
                          ) : (
                            <span className="text-muted-foreground">
                              No privileges
                            </span>
                          )}
                        </TableCell>
                        <TableCell className="text-right">
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" size="icon">
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuLabel>Actions</DropdownMenuLabel>
                              <DropdownMenuSeparator />
                              <DropdownMenuItem>
                                <Key className="h-4 w-4 mr-2" />
                                Manage Privileges
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => handleDeleteUser(user.username)}
                                className="text-destructive"
                              >
                                <Trash2 className="h-4 w-4 mr-2" />
                                Delete
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))}
                    {filteredUsers.length === 0 && (
                      <TableRow>
                        <TableCell
                          colSpan={4}
                          className="text-center py-8 text-muted-foreground"
                        >
                          No users found
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
