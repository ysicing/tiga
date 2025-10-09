import { useEffect, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
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
import { ArrowLeft, Loader2 } from 'lucide-react';
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

export function HostEditPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  // Fetch host data from API
  const { data: hostResponse, isLoading, isError } = useQuery({
    queryKey: ['host', id],
    queryFn: async () => {
      if (!id) throw new Error('No host ID provided');
      return devopsAPI.vms.hosts.get(id);
    },
    enabled: !!id,
  });

  const host = (hostResponse as any)?.data;

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

  useEffect(() => {
    if (host) {
      setFormData({
        name: host.name,
        note: host.note || '',
        public_note: host.public_note || '',
        display_index: host.display_index || 0,
        hide_for_guest: host.hide_for_guest || false,
        cost: host.cost || 0,
        renewal_type: host.renewal_type || 'monthly',
        purchase_date: host.purchase_date ? host.purchase_date.split('T')[0] : '',
        expiry_date: host.expiry_date ? host.expiry_date.split('T')[0] : '',
        auto_renew: host.auto_renew || false,
        traffic_limit: host.traffic_limit || 0,
        group_name: host.group_name || '',
      });
    }
  }, [host]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!id) return;

    try {
      const data: any = await devopsAPI.vms.hosts.update(id, formData);
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

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (isError || !host) {
    return (
      <div className="text-center py-12">
        <p className="text-muted-foreground mb-4">
          {isError ? '加载失败' : '主机未找到'}
        </p>
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

            <div className="flex items-center justify-between py-2">
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

    </div>
  );
}
