import { useEffect, useState } from 'react';
import { ServiceStatus } from '@/components/hosts/service-status';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Plus, Trash2, Edit, Search, RefreshCw } from 'lucide-react';
import { ServiceMonitor } from '@/stores/host-store';

type MonitorFormData = {
  name: string;
  type: 'HTTP' | 'TCP' | 'ICMP';
  target: string;
  interval: number;
  timeout: number;
  enabled: boolean;
  host_node_id?: number;
};

export function ServiceMonitorPage() {
  const [monitors, setMonitors] = useState<ServiceMonitor[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingMonitor, setEditingMonitor] = useState<ServiceMonitor | null>(null);
  const [formData, setFormData] = useState<MonitorFormData>({
    name: '',
    type: 'HTTP',
    target: '',
    interval: 60,
    timeout: 5,
    enabled: true,
  });

  useEffect(() => {
    fetchMonitors();
  }, []);

  const fetchMonitors = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/vms/service-monitors', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`,
        },
      });
      const data = await response.json();
      if (data.code === 0) {
        setMonitors(data.data.items || []);
      }
    } catch (error) {
      console.error('Failed to fetch monitors:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingMonitor(null);
    setFormData({
      name: '',
      type: 'HTTP',
      target: '',
      interval: 60,
      timeout: 5,
      enabled: true,
    });
    setIsDialogOpen(true);
  };

  const handleEdit = (monitor: ServiceMonitor) => {
    setEditingMonitor(monitor);
    setFormData({
      name: monitor.name,
      type: monitor.type as 'HTTP' | 'TCP' | 'ICMP',
      target: monitor.target,
      interval: monitor.interval || 60,
      timeout: monitor.timeout || 5,
      enabled: monitor.enabled,
      host_node_id: monitor.host_node_id,
    });
    setIsDialogOpen(true);
  };

  const handleDelete = async (id: number) => {
    if (!confirm('确定要删除此监控项吗?')) return;

    try {
      const response = await fetch(`/api/v1/vms/service-monitors/${id}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('token')}`,
        },
      });
      const data = await response.json();
      if (data.code === 0) {
        fetchMonitors();
      }
    } catch (error) {
      console.error('Failed to delete monitor:', error);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const url = editingMonitor
      ? `/api/v1/vms/service-monitors/${editingMonitor.id}`
      : '/api/v1/vms/service-monitors';

    const method = editingMonitor ? 'PUT' : 'POST';

    try {
      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify(formData),
      });
      const data = await response.json();
      if (data.code === 0) {
        setIsDialogOpen(false);
        fetchMonitors();
      }
    } catch (error) {
      console.error('Failed to save monitor:', error);
    }
  };

  const filteredMonitors = monitors.filter(
    (monitor) =>
      monitor.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      monitor.target.toLowerCase().includes(searchTerm.toLowerCase())
  );

  const stats = {
    total: monitors.length,
    enabled: monitors.filter((m) => m.enabled).length,
    disabled: monitors.filter((m) => !m.enabled).length,
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">服务监控</h1>
          <p className="text-muted-foreground mt-1">
            管理和配置服务探测监控
          </p>
        </div>
        <Button onClick={handleCreate}>
          <Plus className="mr-2 h-4 w-4" />
          添加监控
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              总监控项
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold">{stats.total}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              已启用
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-green-600">{stats.enabled}</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              已禁用
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-2xl font-bold text-gray-600">{stats.disabled}</p>
          </CardContent>
        </Card>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索监控项..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9"
          />
        </div>
        <Button variant="outline" onClick={fetchMonitors} disabled={loading}>
          <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
      </div>

      {/* Monitor Grid */}
      {filteredMonitors.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-muted-foreground">
            {monitors.length === 0
              ? '暂无监控项，点击"添加监控"开始'
              : '没有找到匹配的监控项'}
          </p>
        </div>
      ) : (
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {filteredMonitors.map((monitor) => (
            <div key={monitor.id} className="relative">
              <ServiceStatus monitor={monitor} />
              <div className="absolute top-4 right-4 flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleEdit(monitor)}
                >
                  <Edit className="h-4 w-4" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => handleDelete(monitor.id)}
                >
                  <Trash2 className="h-4 w-4 text-red-500" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Create/Edit Dialog */}
      <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>
                {editingMonitor ? '编辑监控项' : '添加监控项'}
              </DialogTitle>
              <DialogDescription>
                配置服务探测监控参数
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name">名称</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) =>
                    setFormData({ ...formData, name: e.target.value })
                  }
                  required
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="type">探测类型</Label>
                <Select
                  value={formData.type}
                  onValueChange={(value: 'HTTP' | 'TCP' | 'ICMP') =>
                    setFormData({ ...formData, type: value })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="HTTP">HTTP/HTTPS</SelectItem>
                    <SelectItem value="TCP">TCP</SelectItem>
                    <SelectItem value="ICMP">ICMP (Ping)</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="target">目标</Label>
                <Input
                  id="target"
                  placeholder={
                    formData.type === 'HTTP'
                      ? 'https://example.com'
                      : formData.type === 'TCP'
                      ? '192.168.1.1:8080'
                      : '192.168.1.1'
                  }
                  value={formData.target}
                  onChange={(e) =>
                    setFormData({ ...formData, target: e.target.value })
                  }
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="interval">间隔 (秒)</Label>
                  <Input
                    id="interval"
                    type="number"
                    min="10"
                    value={formData.interval}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        interval: parseInt(e.target.value),
                      })
                    }
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="timeout">超时 (秒)</Label>
                  <Input
                    id="timeout"
                    type="number"
                    min="1"
                    value={formData.timeout}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        timeout: parseInt(e.target.value),
                      })
                    }
                    required
                  />
                </div>
              </div>

              <div className="flex items-center justify-between">
                <Label htmlFor="enabled">启用监控</Label>
                <Switch
                  id="enabled"
                  checked={formData.enabled}
                  onCheckedChange={(checked) =>
                    setFormData({ ...formData, enabled: checked })
                  }
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setIsDialogOpen(false)}
              >
                取消
              </Button>
              <Button type="submit">
                {editingMonitor ? '保存' : '创建'}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
