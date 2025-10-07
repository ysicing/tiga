import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useHostStore } from '@/stores/host-store';
import { useHostMonitor } from '@/hooks/use-host-monitor';
import { HostCard } from '@/components/hosts/host-card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Plus, RefreshCw, Search, LayoutGrid, LayoutList } from 'lucide-react';
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { toast } from 'sonner';
import { devopsAPI } from '@/lib/api-client';

type HostFormData = {
  name: string;
  note?: string;
  public_note?: string;
  display_index: number;
  hide_for_guest: boolean;

  // Billing information
  cost: number;
  renewal_type: 'monthly' | 'yearly';
  purchase_date?: string;
  expiry_date?: string;
  auto_renew: boolean;
  traffic_limit: number;

  // Group (simple string grouping)
  group_name?: string;
};

// Calculate days until expiry
function getDaysUntilExpiry(expiryDate?: string): number | null {
  if (!expiryDate) return null;
  const expiry = new Date(expiryDate);
  const now = new Date();
  const diffTime = expiry.getTime() - now.getTime();
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
  return diffDays;
}

// Get expiry status with color
function getExpiryStatus(days: number | null): { text: string; color: string } {
  if (days === null) return { text: '未设置', color: 'text-muted-foreground' };
  if (days < 0) return { text: `已过期 ${Math.abs(days)} 天`, color: 'text-red-600 font-semibold' };
  if (days === 0) return { text: '今天到期', color: 'text-red-600 font-semibold' };
  if (days <= 7) return { text: `${days} 天后到期`, color: 'text-red-600' };
  if (days <= 30) return { text: `${days} 天后到期`, color: 'text-orange-600' };
  return { text: `${days} 天后到期`, color: 'text-muted-foreground' };
}

export function HostListPage() {
  const navigate = useNavigate();
  const { hosts, loading, setHosts, setLoading } = useHostStore();
  const { connected, reconnecting } = useHostMonitor({ autoConnect: true });
  const [searchTerm, setSearchTerm] = useState('');
  const [viewMode, setViewMode] = useState<'card' | 'table'>(() => {
    // Load from localStorage
    const saved = localStorage.getItem('host-list-view-mode');
    return (saved as 'card' | 'table') || 'card';
  });
  const [isCreateDialogOpen, setIsCreateDialogOpen] = useState(false);
  const [installCmdDialog, setInstallCmdDialog] = useState<{
    open: boolean;
    id?: string;
    cmd?: string;
  }>({ open: false });

  // Save view mode to localStorage
  useEffect(() => {
    localStorage.setItem('host-list-view-mode', viewMode);
  }, [viewMode]);

  const [formData, setFormData] = useState<HostFormData>({
    name: '',
    note: '',
    public_note: '',
    display_index: 0,
    hide_for_guest: false,
    cost: 0,
    renewal_type: 'monthly',
    purchase_date: '',
    expiry_date: '',
    auto_renew: false,
    traffic_limit: 0,
  });

  // Fetch hosts on mount
  useEffect(() => {
    fetchHosts();
  }, []);

  const fetchHosts = async () => {
    setLoading(true);
    try {
      const data: any = await devopsAPI.vms.hosts.list();
      if (data.code === 0) {
        setHosts(data.data.items || []);
      }
    } catch (error) {
      console.error('Failed to fetch hosts:', error);
      toast.error('获取主机列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleCreateHost = async (e: React.FormEvent) => {
    e.preventDefault();

    try {
      const data: any = await devopsAPI.vms.hosts.create(formData);
      if (data.code === 0) {
        toast.success('主机创建成功');
        setIsCreateDialogOpen(false);

        // Show agent install command
        setInstallCmdDialog({
          open: true,
          id: data.data.id,
          cmd: data.data.agent_install_cmd,
        });

        // Reset form
        setFormData({
          name: '',
          note: '',
          public_note: '',
          display_index: 0,
          hide_for_guest: false,
          cost: 0,
          renewal_type: 'monthly',
          purchase_date: '',
          expiry_date: '',
          auto_renew: false,
          traffic_limit: 0,
        });

        // Refresh list
        fetchHosts();
      } else {
        toast.error(data.message || '创建失败');
      }
    } catch (error) {
      console.error('Failed to create host:', error);
      toast.error('创建主机失败');
    }
  };

  const handleDeleteHost = async (id: string, name: string) => {
    if (!confirm(`确定要删除主机"${name}"吗？此操作不可恢复。`)) {
      return;
    }

    try {
      const data: any = await devopsAPI.vms.hosts.delete(id);
      if (data.code === 0) {
        toast.success('主机删除成功');
        fetchHosts();
      } else {
        toast.error(data.message || '删除失败');
      }
    } catch (error) {
      console.error('Failed to delete host:', error);
      toast.error('删除主机失败');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    toast.success('已复制到剪贴板');
  };

  const filteredHosts = hosts.filter((host) =>
    host.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    host.note?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  // Group hosts by group_name for card view
  const groupedHosts = filteredHosts.reduce((acc, host) => {
    const groupName = host.group_name || '未分类主机';
    if (!acc[groupName]) {
      acc[groupName] = [];
    }
    acc[groupName].push(host);
    return acc;
  }, {} as Record<string, typeof filteredHosts>);

  const stats = {
    total: hosts.length,
    online: hosts.filter((h) => h.online).length,
    offline: hosts.filter((h) => !h.online).length,
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">主机管理</h1>
          <p className="text-muted-foreground mt-1">
            管理和监控您的主机节点
          </p>
        </div>
        <Button onClick={() => setIsCreateDialogOpen(true)}>
          <Plus className="mr-2 h-4 w-4" />
          添加主机
        </Button>
      </div>

      {/* Stats Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <div className="rounded-lg border bg-card p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                总主机数
              </p>
              <p className="text-2xl font-bold">{stats.total}</p>
            </div>
          </div>
        </div>
        <div className="rounded-lg border bg-card p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                在线主机
              </p>
              <p className="text-2xl font-bold text-green-600">
                {stats.online}
              </p>
            </div>
          </div>
        </div>
        <div className="rounded-lg border bg-card p-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-muted-foreground">
                离线主机
              </p>
              <p className="text-2xl font-bold text-red-600">
                {stats.offline}
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索主机..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="pl-9"
          />
        </div>
        <ToggleGroup type="single" value={viewMode} onValueChange={(value) => value && setViewMode(value as 'card' | 'table')}>
          <ToggleGroupItem value="card" aria-label="卡片视图">
            <LayoutGrid className="h-4 w-4" />
          </ToggleGroupItem>
          <ToggleGroupItem value="table" aria-label="表格视图">
            <LayoutList className="h-4 w-4" />
          </ToggleGroupItem>
        </ToggleGroup>
        <Button variant="outline" onClick={fetchHosts} disabled={loading}>
          <RefreshCw className={`mr-2 h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          刷新
        </Button>
        <div className="flex items-center gap-2 text-sm">
          <div className={`h-2 w-2 rounded-full ${connected ? 'bg-green-500' : reconnecting ? 'bg-yellow-500 animate-pulse' : 'bg-red-500'}`} />
          <span className="text-muted-foreground">
            {connected ? '实时监控已连接' : reconnecting ? '重新连接中...' : '监控断开'}
          </span>
        </div>
      </div>

      {/* Host Views */}
      {filteredHosts.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-muted-foreground">
            {hosts.length === 0 ? '暂无主机，点击"添加主机"开始' : '没有找到匹配的主机'}
          </p>
        </div>
      ) : viewMode === 'card' ? (
        /* Grouped Card View */
        <div className="space-y-8">
          {Object.entries(groupedHosts).map(([groupName, groupHosts]) => (
            <div key={groupName} className="space-y-4">
              <div className="flex items-center gap-2">
                <h2 className="text-xl font-semibold">{groupName}</h2>
                <span className="text-sm text-muted-foreground">
                  ({groupHosts.length} 台主机)
                </span>
              </div>
              <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
                {groupHosts.map((host) => (
                  <div key={host.id} className="relative group">
                    <HostCard
                      host={host}
                      onClick={() => navigate(`/vms/hosts/${host.id}`)}
                    />
                    <div className="absolute top-4 right-4 opacity-0 group-hover:opacity-100 transition-opacity">
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteHost(host.id, host.name);
                        }}
                      >
                        删除
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      ) : (
        /* Table View */
        <div className="rounded-md border">
          <table className="w-full">
            <thead className="bg-muted/50">
              <tr className="border-b">
                <th className="px-4 py-3 text-left text-sm font-medium">状态</th>
                <th className="px-4 py-3 text-left text-sm font-medium">名称</th>
                <th className="px-4 py-3 text-left text-sm font-medium">分组</th>
                <th className="px-4 py-3 text-left text-sm font-medium">系统</th>
                <th className="px-4 py-3 text-right text-sm font-medium">CPU</th>
                <th className="px-4 py-3 text-right text-sm font-medium">内存</th>
                <th className="px-4 py-3 text-right text-sm font-medium">磁盘</th>
                <th className="px-4 py-3 text-right text-sm font-medium">网络 ↓/↑</th>
                <th className="px-4 py-3 text-left text-sm font-medium">到期时间</th>
                <th className="px-4 py-3 text-right text-sm font-medium">操作</th>
              </tr>
            </thead>
            <tbody>
              {filteredHosts.map((host) => {
                const state = host.current_state;
                const info = host.host_info;
                const daysUntilExpiry = getDaysUntilExpiry(host.expiry_date);
                const expiryStatus = getExpiryStatus(daysUntilExpiry);
                return (
                  <tr
                    key={host.id}
                    className="border-b hover:bg-muted/50 cursor-pointer"
                    onClick={() => navigate(`/vms/hosts/${host.id}`)}
                  >
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <div className={`h-2 w-2 rounded-full ${host.online ? 'bg-green-500' : 'bg-red-500'}`} />
                        <span className="text-sm">{host.online ? '在线' : '离线'}</span>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <div>
                        <div className="font-medium">{host.name}</div>
                        {host.note && (
                          <div className="text-xs text-muted-foreground">{host.note}</div>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-sm">{host.group_name || '-'}</td>
                    <td className="px-4 py-3 text-sm">
                      {info ? `${info.platform} ${info.arch}` : '-'}
                    </td>
                    <td className="px-4 py-3 text-right text-sm">
                      {state ? (
                        <span className={state.cpu_usage >= 80 ? 'text-red-600' : ''}>
                          {state.cpu_usage.toFixed(1)}%
                        </span>
                      ) : '-'}
                    </td>
                    <td className="px-4 py-3 text-right text-sm">
                      {state ? (
                        <span className={state.mem_usage >= 80 ? 'text-red-600' : ''}>
                          {state.mem_usage.toFixed(1)}%
                        </span>
                      ) : '-'}
                    </td>
                    <td className="px-4 py-3 text-right text-sm">
                      {state ? (
                        <span className={state.disk_usage >= 80 ? 'text-red-600' : ''}>
                          {state.disk_usage.toFixed(1)}%
                        </span>
                      ) : '-'}
                    </td>
                    <td className="px-4 py-3 text-right text-sm">
                      {state ? (
                        <span className="text-xs">
                          {(state.net_in_speed / 1024 / 1024).toFixed(1)}M /
                          {(state.net_out_speed / 1024 / 1024).toFixed(1)}M
                        </span>
                      ) : '-'}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <span className={expiryStatus.color}>
                        {expiryStatus.text}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteHost(host.id, host.name);
                        }}
                      >
                        删除
                      </Button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Host Dialog */}
      <Dialog open={isCreateDialogOpen} onOpenChange={setIsCreateDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <form onSubmit={handleCreateHost}>
            <DialogHeader>
              <DialogTitle>添加主机节点</DialogTitle>
              <DialogDescription>
                填写主机信息，创建后将获得Agent安装命令
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name">主机名称 *</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                  placeholder="web-server-01"
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="note">内部备注</Label>
                <Textarea
                  id="note"
                  value={formData.note}
                  onChange={(e) => setFormData({ ...formData, note: e.target.value })}
                  placeholder="管理员可见的备注信息"
                  rows={2}
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="public_note">公开备注</Label>
                <Textarea
                  id="public_note"
                  value={formData.public_note}
                  onChange={(e) => setFormData({ ...formData, public_note: e.target.value })}
                  placeholder="访客可见的备注信息"
                  rows={2}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="cost">费用 (¥)</Label>
                  <Input
                    id="cost"
                    type="number"
                    step="0.01"
                    value={formData.cost}
                    onChange={(e) => setFormData({ ...formData, cost: parseFloat(e.target.value) || 0 })}
                  />
                  <p className="text-xs text-muted-foreground">
                    根据续费周期，此费用为{formData.renewal_type === 'monthly' ? '月' : '年'}费用
                  </p>
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="renewal_type">续费周期</Label>
                  <Select
                    value={formData.renewal_type}
                    onValueChange={(value: 'monthly' | 'yearly') => setFormData({ ...formData, renewal_type: value })}
                  >
                    <SelectTrigger id="renewal_type">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="monthly">按月</SelectItem>
                      <SelectItem value="yearly">按年</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="purchase_date">购买日期</Label>
                  <Input
                    id="purchase_date"
                    type="date"
                    value={formData.purchase_date}
                    onChange={(e) => setFormData({ ...formData, purchase_date: e.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="expiry_date">到期时间</Label>
                  <Input
                    id="expiry_date"
                    type="date"
                    value={formData.expiry_date}
                    onChange={(e) => setFormData({ ...formData, expiry_date: e.target.value })}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="traffic_limit">流量限制 (GB)</Label>
                  <Input
                    id="traffic_limit"
                    type="number"
                    value={formData.traffic_limit}
                    onChange={(e) => setFormData({ ...formData, traffic_limit: parseInt(e.target.value) || 0 })}
                    placeholder="0 表示无限"
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="group_name">主机分组</Label>
                  <Input
                    id="group_name"
                    value={formData.group_name || ''}
                    onChange={(e) => setFormData({ ...formData, group_name: e.target.value })}
                    placeholder="例如：生产环境、测试环境"
                  />
                  <p className="text-xs text-muted-foreground">
                    用于分组管理主机
                  </p>
                </div>
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="auto_renew">自动续费</Label>
                  <p className="text-xs text-muted-foreground">
                    到期时自动续费，避免服务中断
                  </p>
                </div>
                <Switch
                  id="auto_renew"
                  checked={formData.auto_renew}
                  onCheckedChange={(checked) => setFormData({ ...formData, auto_renew: checked })}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="hide_for_guest">对访客隐藏</Label>
                  <p className="text-xs text-muted-foreground">
                    未登录用户无法查看此主机
                  </p>
                </div>
                <Switch
                  id="hide_for_guest"
                  checked={formData.hide_for_guest}
                  onCheckedChange={(checked) => setFormData({ ...formData, hide_for_guest: checked })}
                />
              </div>
            </div>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setIsCreateDialogOpen(false)}>
                取消
              </Button>
              <Button type="submit">创建</Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Agent Install Command Dialog */}
      <Dialog open={installCmdDialog.open} onOpenChange={(open) => setInstallCmdDialog({ ...installCmdDialog, open })}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>Agent 安装命令</DialogTitle>
            <DialogDescription>
              在目标主机上执行以下命令安装 Agent
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>主机 ID</Label>
              <div className="flex gap-2 mt-2">
                <Input value={installCmdDialog.id || ''} readOnly />
                <Button
                  variant="outline"
                  onClick={() => copyToClipboard(installCmdDialog.id || '')}
                >
                  复制
                </Button>
              </div>
            </div>

            <div>
              <Label>安装命令</Label>
              <div className="mt-2 relative">
                <Textarea
                  value={installCmdDialog.cmd || ''}
                  readOnly
                  rows={3}
                  className="font-mono text-sm"
                />
                <Button
                  variant="outline"
                  size="sm"
                  className="absolute top-2 right-2"
                  onClick={() => copyToClipboard(installCmdDialog.cmd || '')}
                >
                  复制命令
                </Button>
              </div>
            </div>

            <div className="text-sm text-muted-foreground">
              <p>执行命令后，Agent 将自动连接到服务器并开始上报监控数据。</p>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={() => setInstallCmdDialog({ open: false })}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
