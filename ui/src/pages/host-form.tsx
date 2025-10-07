import React, { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Badge } from '@/components/ui/badge';
import { ArrowLeft, Save, X } from 'lucide-react';
import { devopsAPI } from '@/lib/api-client';

interface InstanceFormData {
  name: string;
  service_type: string;
  host: string;
  port: string;
  version: string;
  environment: string;
  description: string;
  tags: string[];
  config: Record<string, any>;
}

export default function InstanceFormPage() {
  const navigate = useNavigate();
  const { id } = useParams();
  const isEdit = !!id;

  const [formData, setFormData] = useState<InstanceFormData>({
    name: '',
    service_type: 'mysql',
    host: '',
    port: '',
    version: '',
    environment: 'production',
    description: '',
    tags: [],
    config: {},
  });

  const [newTag, setNewTag] = useState('');
  const [configKey, setConfigKey] = useState('');
  const [configValue, setConfigValue] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);

  const serviceTypes = [
    { value: 'mysql', label: 'MySQL', defaultPort: '3306' },
    { value: 'postgres', label: 'PostgreSQL', defaultPort: '5432' },
    { value: 'redis', label: 'Redis', defaultPort: '6379' },
    { value: 'minio', label: 'MinIO', defaultPort: '9000' },
    { value: 'docker', label: 'Docker', defaultPort: '2375' },
    { value: 'k8s', label: 'Kubernetes', defaultPort: '6443' },
    { value: 'caddy', label: 'Caddy', defaultPort: '2019' },
  ];

  const environments = [
    { value: 'dev', label: 'Development' },
    { value: 'test', label: 'Testing' },
    { value: 'staging', label: 'Staging' },
    { value: 'production', label: 'Production' },
  ];

  const handleServiceTypeChange = (value: string) => {
    const serviceType = serviceTypes.find((st) => st.value === value);
    setFormData({
      ...formData,
      service_type: value,
      port: serviceType?.defaultPort || '',
    });
  };

  const handleAddTag = () => {
    if (newTag && !formData.tags.includes(newTag)) {
      setFormData({ ...formData, tags: [...formData.tags, newTag] });
      setNewTag('');
    }
  };

  const handleRemoveTag = (tag: string) => {
    setFormData({ ...formData, tags: formData.tags.filter((t) => t !== tag) });
  };

  const handleAddConfig = () => {
    if (configKey && configValue) {
      setFormData({
        ...formData,
        config: { ...formData.config, [configKey]: configValue },
      });
      setConfigKey('');
      setConfigValue('');
    }
  };

  const handleRemoveConfig = (key: string) => {
    const newConfig = { ...formData.config };
    delete newConfig[key];
    setFormData({ ...formData, config: newConfig });
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    try {
      const data = {
        ...formData,
        port: parseInt(formData.port),
      };

      if (isEdit) {
        await devopsAPI.instances.update(id!, data);
      } else {
        await devopsAPI.instances.create(data);
      }

      navigate('/instances');
    } catch (error) {
      console.error('Failed to save instance:', error);
      // TODO: Show error toast
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/instances')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h2 className="text-3xl font-bold tracking-tight">
          {isEdit ? 'Edit Instance' : 'Create Instance'}
        </h2>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle>Basic Information</CardTitle>
            <CardDescription>Configure the instance basic settings</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="name">Instance Name *</Label>
                <Input
                  id="name"
                  placeholder="mysql-prod-01"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="service_type">Service Type *</Label>
                <Select value={formData.service_type} onValueChange={handleServiceTypeChange}>
                  <SelectTrigger id="service_type">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {serviceTypes.map((type) => (
                      <SelectItem key={type.value} value={type.value}>
                        {type.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="host">Host *</Label>
                <Input
                  id="host"
                  placeholder="10.0.1.10"
                  value={formData.host}
                  onChange={(e) => setFormData({ ...formData, host: e.target.value })}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="port">Port *</Label>
                <Input
                  id="port"
                  type="number"
                  placeholder="3306"
                  value={formData.port}
                  onChange={(e) => setFormData({ ...formData, port: e.target.value })}
                  required
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="version">Version</Label>
                <Input
                  id="version"
                  placeholder="8.0.35"
                  value={formData.version}
                  onChange={(e) => setFormData({ ...formData, version: e.target.value })}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="environment">Environment</Label>
                <Select
                  value={formData.environment}
                  onValueChange={(value) => setFormData({ ...formData, environment: value })}
                >
                  <SelectTrigger id="environment">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {environments.map((env) => (
                      <SelectItem key={env.value} value={env.value}>
                        {env.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                placeholder="Production MySQL database for main application"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                rows={3}
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Tags</CardTitle>
            <CardDescription>Add tags to organize instances</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Input
                placeholder="Add tag (e.g., production, critical)"
                value={newTag}
                onChange={(e) => setNewTag(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && (e.preventDefault(), handleAddTag())}
              />
              <Button type="button" variant="outline" onClick={handleAddTag}>
                Add
              </Button>
            </div>
            {formData.tags.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {formData.tags.map((tag) => (
                  <Badge key={tag} variant="secondary" className="flex items-center gap-1">
                    {tag}
                    <X
                      className="h-3 w-3 cursor-pointer"
                      onClick={() => handleRemoveTag(tag)}
                    />
                  </Badge>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Configuration</CardTitle>
            <CardDescription>Service-specific configuration parameters</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Input
                placeholder="Key (e.g., max_connections)"
                value={configKey}
                onChange={(e) => setConfigKey(e.target.value)}
              />
              <Input
                placeholder="Value (e.g., 1000)"
                value={configValue}
                onChange={(e) => setConfigValue(e.target.value)}
              />
              <Button type="button" variant="outline" onClick={handleAddConfig}>
                Add
              </Button>
            </div>
            {Object.entries(formData.config).length > 0 && (
              <div className="space-y-2">
                {Object.entries(formData.config).map(([key, value]) => (
                  <div key={key} className="flex items-center justify-between p-3 border rounded">
                    <div className="flex gap-4">
                      <span className="font-medium font-mono text-sm">{key}</span>
                      <span className="text-sm text-muted-foreground font-mono">{value}</span>
                    </div>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      onClick={() => handleRemoveConfig(key)}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        <div className="flex justify-end gap-4">
          <Button type="button" variant="outline" onClick={() => navigate('/instances')}>
            Cancel
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            <Save className="mr-2 h-4 w-4" />
            {isSubmitting ? 'Saving...' : isEdit ? 'Update Instance' : 'Create Instance'}
          </Button>
        </div>
      </form>
    </div>
  );
}
