import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useHostStore } from '@/stores/host-store';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
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
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ArrowLeft } from 'lucide-react';
import { toast } from 'sonner';

type HostFormData = {
  name: string;
  note?: string;
  public_note?: string;
  display_index: number;
  hide_for_guest: boolean;

  // Billing information
  monthly_cost: number;
  yearly_cost: number;
  renewal_type: 'monthly' | 'yearly';
  traffic_limit: number;
  expiry_date?: string;

  // Group
  group_id?: string;
};

export function HostEditPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { hosts } = useHostStore();
  const host = hosts.find((h) => h.id === id);
  const [groups, setGroups] = useState<Array<{ id: string; name: string }>>([]);

  const [formData, setFormData] = useState<HostFormData>({
    name: '',
    note: '',
    public_note: '',
    display_index: 0,
    hide_for_guest: false,
    monthly_cost: 0,
    yearly_cost: 0,
    renewal_type: 'monthly',
    traffic_limit: 0,
    expiry_date: '',
  });

  useEffect(() => {
    fetchGroups();
  }, []);

  useEffect(() => {
    if (host) {
      setFormData({
        name: host.name,
        note: host.note || '',
        public_note: host.public_note || '',
        display_index: host.display_index || 0,
        hide_for_guest: host.hide_for_guest || false,
        monthly_cost: host.monthly_cost || 0,
        yearly_cost: host.yearly_cost || 0,
        renewal_type: host.renewal_type || 'monthly',
        traffic_limit: host.traffic_limit || 0,
        expiry_date: host.expiry_date ? host.expiry_date.split('T')[0] : '',
        group_id: host.group_id,
      });
    }
  }, [host]);

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
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!id) return;

    try {
      const response = await fetch(`/api/v1/vms/hosts/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify(formData),
      });

      const data = await response.json();
      if (data.code === 0) {
        toast.success('主机更新成功');
        navigate(`/vms/hosts/${id}`);
      } else {
        toast.error(data.message || '更新失败');
      }
    } catch (error) {
      console.error('Failed to update host:', error);
      toast.error('更新主机失败');
    }
  };

  if (!host) {
    return (
      <div className="text-center py-12">
        <p>主机未找到</p>
        <Button onClick={() => navigate('/vms/hosts')} className="mt-4">
          返回列表
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6 max-w-2xl">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate(`/vms/hosts/${id}`)}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div>
          <h1 className="text-3xl font-bold">编辑主机</h1>
          <p className="text-muted-foreground mt-1">{host.name}</p>
        </div>
      </div>

      {/* Edit Form */}
      <Card>
        <CardHeader>
          <CardTitle>主机配置</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="name">主机名称 *</Label>
              <Input
                id="name"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                required
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

            <div className="grid gap-2">
              <Label htmlFor="display_index">显示顺序</Label>
              <Input
                id="display_index"
                type="number"
                value={formData.display_index}
                onChange={(e) => setFormData({ ...formData, display_index: parseInt(e.target.value) })}
              />
              <p className="text-xs text-muted-foreground">
                数值越小，显示越靠前
              </p>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="monthly_cost">月费用 (¥)</Label>
                <Input
                  id="monthly_cost"
                  type="number"
                  step="0.01"
                  value={formData.monthly_cost}
                  onChange={(e) => setFormData({ ...formData, monthly_cost: parseFloat(e.target.value) || 0 })}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="yearly_cost">年费用 (¥)</Label>
                <Input
                  id="yearly_cost"
                  type="number"
                  step="0.01"
                  value={formData.yearly_cost}
                  onChange={(e) => setFormData({ ...formData, yearly_cost: parseFloat(e.target.value) || 0 })}
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
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
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div className="grid gap-2">
                <Label htmlFor="expiry_date">到期时间</Label>
                <Input
                  id="expiry_date"
                  type="date"
                  value={formData.expiry_date}
                  onChange={(e) => setFormData({ ...formData, expiry_date: e.target.value })}
                />
              </div>
              <div className="grid gap-2">
                <Label htmlFor="group_id">主机分组</Label>
                <Select
                  value={formData.group_id}
                  onValueChange={(value) => setFormData({ ...formData, group_id: value || undefined })}
                >
                  <SelectTrigger id="group_id">
                    <SelectValue placeholder="选择分组（可选）" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="">无分组</SelectItem>
                    {groups.map((group) => (
                      <SelectItem key={group.id} value={group.id}>
                        {group.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="flex items-center justify-between py-2">
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

            <div className="flex gap-4 pt-4">
              <Button type="submit">保存修改</Button>
              <Button
                type="button"
                variant="outline"
                onClick={() => navigate(`/vms/hosts/${id}`)}
              >
                取消
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="border-red-200 dark:border-red-900">
        <CardHeader>
          <CardTitle className="text-red-600 dark:text-red-400">危险操作</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <h3 className="font-medium mb-2">主机 ID</h3>
            <div className="flex gap-2">
              <Input value={host.id} readOnly className="font-mono text-sm" />
              <Button
                variant="outline"
                onClick={() => {
                  navigator.clipboard.writeText(host.id);
                  toast.success('已复制到剪贴板');
                }}
              >
                复制
              </Button>
            </div>
            <p className="text-xs text-muted-foreground mt-1">
              ID用于Agent连接认证，请勿泄露
            </p>
          </div>

          <div>
            <h3 className="font-medium mb-2">重置密钥</h3>
            <p className="text-sm text-muted-foreground mb-2">
              重置后需要重新安装Agent，旧的Agent将无法连接
            </p>
            <Button variant="destructive" disabled>
              重置密钥（开发中）
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
