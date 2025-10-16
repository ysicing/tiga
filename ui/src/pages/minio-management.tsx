import { useEffect, useState } from 'react'
import {
  ArrowLeft,
  Download,
  File,
  Folder,
  FolderOpen,
  MoreHorizontal,
  Plus,
  RefreshCw,
  Search,
  Trash2,
  Upload,
} from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import { toast } from 'sonner'

import { devopsAPI } from '@/lib/api-client'
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

interface Bucket {
  name: string
  creation_date: string
  size?: number
  object_count?: number
}

interface S3Object {
  key: string
  size: number
  last_modified: string
  is_dir: boolean
  etag?: string
}

export default function MinIOManagementPage() {
  const { instanceId } = useParams<{ instanceId: string }>()
  const navigate = useNavigate()
  const [buckets, setBuckets] = useState<Bucket[]>([])
  const [selectedBucket, setSelectedBucket] = useState<string | null>(null)
  const [objects, setObjects] = useState<S3Object[]>([])
  const [currentPath, setCurrentPath] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [searchTerm, setSearchTerm] = useState('')

  // Dialog states
  const [createBucketOpen, setCreateBucketOpen] = useState(false)
  const [uploadFileOpen, setUploadFileOpen] = useState(false)
  const [newBucketName, setNewBucketName] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)

  useEffect(() => {
    if (instanceId) {
      loadBuckets()
    }
  }, [instanceId])

  useEffect(() => {
    if (selectedBucket) {
      loadObjects(selectedBucket, currentPath)
    }
  }, [selectedBucket, currentPath])

  const loadBuckets = async () => {
    if (!instanceId) return
    setLoading(true)
    try {
      const response = (await devopsAPI.minio.listBuckets(instanceId)) as any
      setBuckets(response.buckets || [])
    } catch (error: any) {
      toast.error('Failed to load buckets', {
        description: error.message,
      })
    } finally {
      setLoading(false)
    }
  }

  const loadObjects = async (bucketName: string, prefix: string = '') => {
    if (!instanceId) return
    setLoading(true)
    try {
      const response = (await devopsAPI.minio.listObjects(
        instanceId,
        bucketName,
        prefix
      )) as any
      setObjects(response.objects || [])
    } catch (error: any) {
      toast.error('Failed to load objects', {
        description: error.message,
      })
    } finally {
      setLoading(false)
    }
  }

  const handleCreateBucket = async () => {
    if (!instanceId || !newBucketName) return

    try {
      await devopsAPI.minio.createBucket(instanceId, newBucketName)
      toast.success('Bucket created successfully')
      setNewBucketName('')
      setCreateBucketOpen(false)
      loadBuckets()
    } catch (error: any) {
      toast.error('Failed to create bucket', {
        description: error.message,
      })
    }
  }

  const handleDeleteBucket = async (bucketName: string) => {
    if (!instanceId) return

    if (!confirm(`Are you sure you want to delete bucket "${bucketName}"?`)) {
      return
    }

    try {
      await devopsAPI.minio.deleteBucket(instanceId, bucketName)
      toast.success('Bucket deleted successfully')
      if (selectedBucket === bucketName) {
        setSelectedBucket(null)
      }
      loadBuckets()
    } catch (error: any) {
      toast.error('Failed to delete bucket', {
        description: error.message,
      })
    }
  }

  const handleUploadFile = async () => {
    if (!instanceId || !selectedBucket || !selectedFile) return

    try {
      const objectName = currentPath
        ? `${currentPath}/${selectedFile.name}`
        : selectedFile.name
      await devopsAPI.minio.uploadObject(
        instanceId,
        selectedBucket,
        objectName,
        selectedFile
      )
      toast.success('File uploaded successfully')
      setSelectedFile(null)
      setUploadFileOpen(false)
      loadObjects(selectedBucket, currentPath)
    } catch (error: any) {
      toast.error('Failed to upload file', {
        description: error.message,
      })
    }
  }

  const handleDeleteObject = async (objectKey: string) => {
    if (!instanceId || !selectedBucket) return

    if (!confirm(`Are you sure you want to delete "${objectKey}"?`)) {
      return
    }

    try {
      await devopsAPI.minio.deleteObject(instanceId, selectedBucket, objectKey)
      toast.success('Object deleted successfully')
      loadObjects(selectedBucket, currentPath)
    } catch (error: any) {
      toast.error('Failed to delete object', {
        description: error.message,
      })
    }
  }

  const handleDownloadObject = async (objectKey: string) => {
    if (!instanceId || !selectedBucket) return

    try {
      const response = (await devopsAPI.minio.getObjectUrl(
        instanceId,
        selectedBucket,
        objectKey
      )) as any
      window.open(response.url, '_blank')
    } catch (error: any) {
      toast.error('Failed to download object', {
        description: error.message,
      })
    }
  }

  const handleNavigateToFolder = (folderKey: string) => {
    setCurrentPath(folderKey)
  }

  const handleNavigateUp = () => {
    const pathParts = currentPath.split('/').filter(Boolean)
    pathParts.pop()
    setCurrentPath(pathParts.join('/'))
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
    const i = Math.min(
      Math.floor(Math.log(bytes) / Math.log(k)),
      sizes.length - 1
    )
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
  }

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString()
  }

  const filteredBuckets = buckets.filter((bucket) =>
    bucket.name.toLowerCase().includes(searchTerm.toLowerCase())
  )

  const filteredObjects = objects.filter((obj) =>
    obj.key.toLowerCase().includes(searchTerm.toLowerCase())
  )

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(-1)}>
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <h1 className="text-3xl font-bold">MinIO Management</h1>
            <p className="text-muted-foreground">Instance ID: {instanceId}</p>
          </div>
        </div>
        <Button onClick={loadBuckets} variant="outline" size="icon">
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Buckets List */}
        <Card className="lg:col-span-1">
          <CardHeader>
            <div className="flex items-center justify-between">
              <CardTitle>Buckets</CardTitle>
              <Dialog
                open={createBucketOpen}
                onOpenChange={setCreateBucketOpen}
              >
                <DialogTrigger asChild>
                  <Button size="sm">
                    <Plus className="h-4 w-4 mr-2" />
                    New Bucket
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Create New Bucket</DialogTitle>
                    <DialogDescription>
                      Enter a name for your new S3 bucket
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="bucket-name">Bucket Name</Label>
                      <Input
                        id="bucket-name"
                        placeholder="my-bucket"
                        value={newBucketName}
                        onChange={(e) => setNewBucketName(e.target.value)}
                      />
                    </div>
                  </div>
                  <DialogFooter>
                    <Button
                      variant="outline"
                      onClick={() => setCreateBucketOpen(false)}
                    >
                      Cancel
                    </Button>
                    <Button
                      onClick={handleCreateBucket}
                      disabled={!newBucketName}
                    >
                      Create
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
            <CardDescription>Manage your storage buckets</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <div className="relative">
                <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Search buckets..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-8"
                />
              </div>
              <div className="space-y-1 max-h-[600px] overflow-y-auto">
                {filteredBuckets.map((bucket) => (
                  <div
                    key={bucket.name}
                    className={`flex items-center justify-between p-3 rounded-lg border cursor-pointer hover:bg-accent transition-colors ${
                      selectedBucket === bucket.name ? 'bg-accent' : ''
                    }`}
                    onClick={() => {
                      setSelectedBucket(bucket.name)
                      setCurrentPath('')
                    }}
                  >
                    <div className="flex items-center gap-3">
                      <Folder className="h-5 w-5 text-blue-500" />
                      <div>
                        <div className="font-medium">{bucket.name}</div>
                        <div className="text-xs text-muted-foreground">
                          {formatDate(bucket.creation_date)}
                        </div>
                      </div>
                    </div>
                    <DropdownMenu>
                      <DropdownMenuTrigger
                        asChild
                        onClick={(e) => e.stopPropagation()}
                      >
                        <Button variant="ghost" size="icon">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuLabel>Actions</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem
                          onClick={(e) => {
                            e.stopPropagation()
                            handleDeleteBucket(bucket.name)
                          }}
                          className="text-destructive"
                        >
                          <Trash2 className="h-4 w-4 mr-2" />
                          Delete Bucket
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                ))}
                {filteredBuckets.length === 0 && (
                  <div className="text-center py-8 text-muted-foreground">
                    No buckets found
                  </div>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Objects List */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>
                  {selectedBucket ? (
                    <div className="flex items-center gap-2">
                      <FolderOpen className="h-5 w-5" />
                      {selectedBucket}
                      {currentPath && ` / ${currentPath}`}
                    </div>
                  ) : (
                    'Select a bucket'
                  )}
                </CardTitle>
                <CardDescription>
                  {selectedBucket
                    ? 'Browse and manage objects in this bucket'
                    : 'Choose a bucket from the left panel to view its contents'}
                </CardDescription>
              </div>
              {selectedBucket && (
                <div className="flex gap-2">
                  {currentPath && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleNavigateUp}
                    >
                      <ArrowLeft className="h-4 w-4 mr-2" />
                      Back
                    </Button>
                  )}
                  <Dialog
                    open={uploadFileOpen}
                    onOpenChange={setUploadFileOpen}
                  >
                    <DialogTrigger asChild>
                      <Button size="sm">
                        <Upload className="h-4 w-4 mr-2" />
                        Upload
                      </Button>
                    </DialogTrigger>
                    <DialogContent>
                      <DialogHeader>
                        <DialogTitle>Upload File</DialogTitle>
                        <DialogDescription>
                          Upload a file to {selectedBucket}
                          {currentPath && ` / ${currentPath}`}
                        </DialogDescription>
                      </DialogHeader>
                      <div className="space-y-4">
                        <div className="space-y-2">
                          <Label htmlFor="file">File</Label>
                          <Input
                            id="file"
                            type="file"
                            onChange={(e) =>
                              setSelectedFile(e.target.files?.[0] || null)
                            }
                          />
                        </div>
                      </div>
                      <DialogFooter>
                        <Button
                          variant="outline"
                          onClick={() => setUploadFileOpen(false)}
                        >
                          Cancel
                        </Button>
                        <Button
                          onClick={handleUploadFile}
                          disabled={!selectedFile}
                        >
                          Upload
                        </Button>
                      </DialogFooter>
                    </DialogContent>
                  </Dialog>
                </div>
              )}
            </div>
          </CardHeader>
          <CardContent>
            {selectedBucket ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Size</TableHead>
                    <TableHead>Last Modified</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredObjects.map((obj) => (
                    <TableRow key={obj.key}>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          {obj.is_dir ? (
                            <>
                              <Folder className="h-4 w-4 text-blue-500" />
                              <button
                                className="hover:underline"
                                onClick={() => handleNavigateToFolder(obj.key)}
                              >
                                {obj.key.split('/').filter(Boolean).pop()}
                              </button>
                            </>
                          ) : (
                            <>
                              <File className="h-4 w-4 text-gray-500" />
                              <span>
                                {obj.key.split('/').filter(Boolean).pop()}
                              </span>
                            </>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        {obj.is_dir ? '-' : formatBytes(obj.size)}
                      </TableCell>
                      <TableCell>{formatDate(obj.last_modified)}</TableCell>
                      <TableCell className="text-right">
                        {!obj.is_dir && (
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
                                onClick={() => handleDownloadObject(obj.key)}
                              >
                                <Download className="h-4 w-4 mr-2" />
                                Download
                              </DropdownMenuItem>
                              <DropdownMenuItem
                                onClick={() => handleDeleteObject(obj.key)}
                                className="text-destructive"
                              >
                                <Trash2 className="h-4 w-4 mr-2" />
                                Delete
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {filteredObjects.length === 0 && (
                    <TableRow>
                      <TableCell
                        colSpan={4}
                        className="text-center py-8 text-muted-foreground"
                      >
                        No objects found in this bucket
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            ) : (
              <div className="flex items-center justify-center h-64 text-muted-foreground">
                <div className="text-center">
                  <FolderOpen className="h-12 w-12 mx-auto mb-4 opacity-50" />
                  <p>Select a bucket to view its contents</p>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
