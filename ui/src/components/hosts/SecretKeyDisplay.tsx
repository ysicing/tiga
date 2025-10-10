import { useState } from 'react';
import { Eye, EyeOff, Copy, RefreshCw, Check } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import { toast } from 'sonner';
import { cn } from '@/lib/utils';
import { devopsAPI } from '@/lib/api-client';

interface SecretKeyDisplayProps {
  hostId: string;
  secretKey?: string;
  agentInstallCmd?: string;
  onKeyRegenerated?: (newKey: string, newCommand: string) => void;
}

export function SecretKeyDisplay({
  hostId,
  secretKey,
  agentInstallCmd,
  onKeyRegenerated,
}: SecretKeyDisplayProps) {
  const safeSecretKey = secretKey ?? '';
  const safeAgentInstallCmd = agentInstallCmd ?? '';
  const hasSecretKey = safeSecretKey.length > 0;
  const [showKey, setShowKey] = useState(false);
  const [copiedKey, setCopiedKey] = useState(false);
  const [copiedCommand, setCopiedCommand] = useState(false);
  const [regenerating, setRegenerating] = useState(false);

  // 复制到剪贴板
  const copyToClipboard = async (text: string, type: 'key' | 'command') => {
    try {
      await navigator.clipboard.writeText(text);
      if (type === 'key') {
        setCopiedKey(true);
        setTimeout(() => setCopiedKey(false), 2000);
      } else {
        setCopiedCommand(true);
        setTimeout(() => setCopiedCommand(false), 2000);
      }
      toast.success(`${type === 'key' ? '密钥' : '安装命令'}已复制到剪贴板`);
    } catch (err) {
      toast.error('无法复制到剪贴板');
    }
  };

  // 重置密钥
  const handleRegenerateKey = async () => {
    setRegenerating(true);
    try {
      const response: any = await devopsAPI.vms.hosts.regenerateSecretKey(hostId);

      // API client returns the full response including code and data
      if (response.code === 0) {
        toast.success('密钥已重置', {
          description: '旧 Agent 连接已断开，请使用新命令重新部署 Agent',
        });

        // 通知父组件密钥已更新
        if (onKeyRegenerated) {
          onKeyRegenerated(
            response.data.secret_key,
            response.data.agent_install_cmd
          );
        }
      } else {
        throw new Error(response.message || '重置失败');
      }
    } catch (error) {
      toast.error('重置失败', {
        description: error instanceof Error ? error.message : '未知错误',
      });
    } finally {
      setRegenerating(false);
    }
  };

  // 遮挡密钥显示
  const maskedKey =
    safeSecretKey.length > 8
      ? safeSecretKey.slice(0, 8) + '****' + safeSecretKey.slice(-8)
      : safeSecretKey || '未生成';

  return (
    <div className="space-y-4">
      {/* 密钥显示 */}
      <div className="space-y-2">
        <Label htmlFor="secret-key">密钥 (Secret Key)</Label>
        <div className="flex gap-2">
          <div className="relative flex-1">
            <Input
              id="secret-key"
              type="text"
              value={hasSecretKey ? (showKey ? safeSecretKey : maskedKey) : '未生成'}
              readOnly
              className="pr-24 font-mono text-sm"
            />
            <div className="absolute right-2 top-1/2 -translate-y-1/2 flex gap-1">
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={() => hasSecretKey && setShowKey(!showKey)}
                title={showKey ? '隐藏密钥' : '显示密钥'}
                disabled={!hasSecretKey}
              >
                {showKey ? (
                  <EyeOff className="h-4 w-4" />
                ) : (
                  <Eye className="h-4 w-4" />
                )}
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={() => copyToClipboard(safeSecretKey, 'key')}
                title="复制密钥"
                disabled={!hasSecretKey}
              >
                {copiedKey ? (
                  <Check className="h-4 w-4 text-green-500" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>

          {/* 重置密钥按钮 */}
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button
                type="button"
                variant="outline"
                size="sm"
                disabled={regenerating}
              >
                <RefreshCw
                  className={cn('h-4 w-4 mr-2', regenerating && 'animate-spin')}
                />
                重置密钥
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>确认重置密钥？</AlertDialogTitle>
                <AlertDialogDescription>
                  重置密钥后：
                  <ul className="list-disc list-inside mt-2 space-y-1">
                    <li>旧密钥将立即失效</li>
                    <li>当前 Agent 连接会被断开</li>
                    <li>需要使用新的安装命令重新部署 Agent</li>
                  </ul>
                  <p className="mt-2 text-red-600 font-medium">此操作不可撤销！</p>
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction onClick={handleRegenerateKey}>
                  确认重置
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </div>
        <p className="text-xs text-muted-foreground">
          密钥用于 Agent 连接认证，请妥善保管。如果密钥泄露，请立即重置。
        </p>
      </div>

      {/* 安装命令显示 */}
      <div className="space-y-2">
        <Label htmlFor="install-command">Agent 安装命令</Label>
        <div className="relative">
          <textarea
            id="install-command"
            value={safeAgentInstallCmd}
            readOnly
            rows={3}
            placeholder="暂无可用安装命令"
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono resize-none focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
          />
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="absolute right-2 top-2 h-7 w-7"
            onClick={() => copyToClipboard(safeAgentInstallCmd, 'command')}
            title="复制安装命令"
            disabled={!safeAgentInstallCmd}
          >
            {copiedCommand ? (
              <Check className="h-4 w-4 text-green-500" />
            ) : (
              <Copy className="h-4 w-4" />
            )}
          </Button>
        </div>
        <p className="text-xs text-muted-foreground">
          在目标主机上执行此命令即可安装并启动 Agent
        </p>
      </div>
    </div>
  );
}
