import { useEffect, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Plus, Pencil, Trash2, FolderOpen } from 'lucide-react';
import { toast } from 'sonner';

interface HostGroup {
  id: number;
  name: string;
  description?: string;
  host_count?: number;
  created_at: string;
  updated_at?: string;
}

type GroupFormData = {
  name: string;
  description: string;
};

export function HostGroupsPage() {
  const [groups, setGroups] = useState<HostGroup[]>([]);
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<HostGroup | null>(null);

  const [formData, setFormData] = useState<GroupFormData>({
    name: '',
    description: '',
  });

  useEffect(() => {
    fetchGroups();
  }, []);

  const fetchGroups = async () => {
    try {
      const response = await fetch('/api/v1/vms/host-groups', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`,
        },
      });
      const data = await response.json();
      if (data.code === 0) {
        setGroups(data.data.items || []);
      }
    } catch (error) {
      console.error('Failed to fetch groups:', error);
      toast.error('获取分组列表失败');
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      const response = await fetch('/api/v1/vms/host-groups', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify(formData),
      });

      const data = await response.json();
      if (data.code === 0) {
        toast.success('分组创建成功');
        setIsCreateDialogOpen(false);
        setFormData({ name: '', description: '' });
        fetchGroups();
      } else {
        toast.error(data.message || '创建失败');
      }
    } catch (error) {
      console.error('Failed to create group:', error);
      toast.error('创建分组失败');
    }
  };

  const handleEdit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingGroup) return;

    try {
      const response = await fetch(`/api/v1/vms/host-groups/${editingGroup.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify(formData),
      });

      const data = await response.json();
      if (data.code === 0) {
        toast.success('分组更新成功');
        setIsEditDialogOpen(false);
        setEditingGroup(null);
        setFormData({ name: '', description: '' });
        fetchGroups();
      } else {
        toast.error(data.message || '更新失败');
      }
    } catch (error) {
      console.error('Failed to update group:', error);
      toast.error('更新分组失败');
    }
  };

  const handleDelete = async (id: number, name: string) => {
    if (!confirm(`确定要删除分组"${name}"吗？此操作不可恢复。`)) {
      return;
    }

    try {
      const response = await fetch(`/api/v1/vms/host-groups/${id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`,
        },
      });

      const data = await response.json();
      if (data.code === 0) {
        toast.success('分组删除成功');
        fetchGroups();
      } else {
        toast.error(data.message || '删除失败');
      }
    } catch (error) {
      console.error('Failed to delete group:', error);
      toast.error('删除分组失败');
    }
  };

  const openEditDialog = (group: HostGroup) => {
    setEditingGroup(group);
    setFormData({
      name: group.name,
      description: group.description || '',
    });
    setIsEditDialogOpen(true);
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">主机分组</h1>
          <p className="text-muted-foreground mt-1">
            管理主机分组，便于批量管理和查看
          </p>
        </div>
        <Button onClick={() => setIsCreateDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          创建分组
        </Button>
      </div>

      {/* Stats Card */}
      <Card>
        <CardHeader>
          <CardTitle>分组概览</CardTitle>
          <CardDescription>当前共有 {groups.length} 个主机分组</CardDescription>
        </CardHeader>
      </Card>

      {/* Groups Table */}
      <Card>
        <CardHeader>
          <CardTitle>分组列表</CardTitle>
        </CardHeader>
        <CardContent>
          {groups.length === 0 ? (
            <div className="text-center py-12">
              <FolderOpen className="mx-auto h-12 w-12 text-muted-foreground" />
              <p className="mt-4 text-muted-foreground">
                暂无分组，点击"创建分组"开始
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>分组名称</TableHead>
                  <TableHead>描述</TableHead>
                  <TableHead className="text-center">主机数量</TableHead>
                  <TableHead>创建时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {groups.map((group) => (
                  <TableRow key={group.id}>
                    <TableCell className="font-medium">{group.name}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {group.description || '-'}
                    </TableCell>
                    <TableCell className="text-center">
                      <span className="inline-flex items-center rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-medium text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                        {group.host_count || 0}
                      </span>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {new Date(group.created_at).toLocaleDateString('zh-CN')}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => openEditDialog(group)}
                        >
                          <Pencil className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleDelete(group.id, group.name)}
                        >
                          <Trash2 className="h-4 w-4 text-red-600" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Create Dialog */}
      <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
        <DialogContent>
          <form onSubmit={handleCreate}>
            <DialogHeader>
              <DialogTitle>创建主机分组</DialogTitle>
              <DialogDescription>
                创建新的主机分组以便于管理
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="create-name">分组名称 *</Label>
                <Input
                  id="create-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                  placeholder="生产环境"
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="create-description">描述</Label>
                <Textarea
                  id="create-description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="分组描述信息"
                  rows={3}
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setIsCreateDialogOpen(false);
                  setFormData({ name: '', description: '' });
                }}
              >
                取消
              </Button>
              <Button type="submit">创建</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog open={isEditDialogOpen} onOpenChange={setIsEditDialogOpen}>
        <DialogContent>
          <form onSubmit={handleEdit}>
            <DialogHeader>
              <DialogTitle>编辑主机分组</DialogTitle>
              <DialogDescription>
                修改分组信息
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="edit-name">分组名称 *</Label>
                <Input
                  id="edit-name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="edit-description">描述</Label>
                <Textarea
                  id="edit-description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  rows={3}
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setIsEditDialogOpen(false);
                  setEditingGroup(null);
                  setFormData({ name: '', description: '' });
                }}
              >
                取消
              </Button>
              <Button type="submit">保存</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
