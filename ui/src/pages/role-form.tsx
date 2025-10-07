import React, { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Checkbox } from '@/components/ui/checkbox';
import { ArrowLeft, Save } from 'lucide-react';
import { devopsAPI } from '@/lib/api-client';

interface RoleFormData {
  name: string;
  description: string;
  permissions: string[];
}

const AVAILABLE_PERMISSIONS = [
  { category: 'Instances', permissions: [
    { id: 'instances:read', label: 'View Instances' },
    { id: 'instances:create', label: 'Create Instances' },
    { id: 'instances:update', label: 'Update Instances' },
    { id: 'instances:delete', label: 'Delete Instances' },
  ]},
  { category: 'Metrics', permissions: [
    { id: 'metrics:read', label: 'View Metrics' },
    { id: 'metrics:create', label: 'Create Metrics' },
  ]},
  { category: 'Alerts', permissions: [
    { id: 'alerts:read', label: 'View Alerts' },
    { id: 'alerts:create', label: 'Create Alert Rules' },
    { id: 'alerts:update', label: 'Update Alert Rules' },
    { id: 'alerts:delete', label: 'Delete Alert Rules' },
    { id: 'alerts:acknowledge', label: 'Acknowledge Alerts' },
    { id: 'alerts:resolve', label: 'Resolve Alerts' },
  ]},
  { category: 'Users', permissions: [
    { id: 'users:read', label: 'View Users' },
    { id: 'users:create', label: 'Create Users' },
    { id: 'users:update', label: 'Update Users' },
    { id: 'users:delete', label: 'Delete Users' },
  ]},
  { category: 'Roles', permissions: [
    { id: 'roles:read', label: 'View Roles' },
    { id: 'roles:create', label: 'Create Roles' },
    { id: 'roles:update', label: 'Update Roles' },
    { id: 'roles:delete', label: 'Delete Roles' },
  ]},
  { category: 'Audit', permissions: [
    { id: 'audit:read', label: 'View Audit Logs' },
  ]},
];

export default function RoleFormPage() {
  const navigate = useNavigate();
  const { id } = useParams();
  const isEdit = !!id;

  const [formData, setFormData] = useState<RoleFormData>({
    name: '',
    description: '',
    permissions: [],
  });

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (isEdit) {
      const fetchRole = async () => {
        try {
          const response: any = await devopsAPI.roles.get(id!);
          const role = response.data;
          setFormData({
            name: role.name,
            description: role.description || '',
            permissions: role.permissions || [],
          });
        } catch (error) {
          console.error('Failed to fetch role:', error);
        }
      };

      fetchRole();
    }
  }, [id, isEdit]);

  const handlePermissionToggle = (permissionId: string) => {
    const newPermissions = formData.permissions.includes(permissionId)
      ? formData.permissions.filter((p) => p !== permissionId)
      : [...formData.permissions, permissionId];

    setFormData({ ...formData, permissions: newPermissions });
  };

  const handleSelectAll = (categoryPermissions: { id: string; label: string }[]) => {
    const categoryIds = categoryPermissions.map((p) => p.id);
    const allSelected = categoryIds.every((id) => formData.permissions.includes(id));

    if (allSelected) {
      // Deselect all in this category
      setFormData({
        ...formData,
        permissions: formData.permissions.filter((p) => !categoryIds.includes(p)),
      });
    } else {
      // Select all in this category
      const newPermissions = [...new Set([...formData.permissions, ...categoryIds])];
      setFormData({ ...formData, permissions: newPermissions });
    }
  };

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.name.trim()) {
      newErrors.name = 'Role name is required';
    } else if (formData.name.length < 3) {
      newErrors.name = 'Role name must be at least 3 characters';
    }

    if (formData.permissions.length === 0) {
      newErrors.permissions = 'At least one permission must be selected';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);

    try {
      const data = {
        name: formData.name,
        description: formData.description,
        permissions: formData.permissions,
      };

      if (isEdit) {
        await devopsAPI.roles.update(id!, data);
      } else {
        await devopsAPI.roles.create(data);
      }

      navigate('/roles');
    } catch (error: any) {
      console.error('Failed to save role:', error);
      setErrors({ submit: error.message || 'Failed to save role' });
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/roles')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h2 className="text-3xl font-bold tracking-tight">
          {isEdit ? 'Edit Role' : 'Create Role'}
        </h2>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle>Role Information</CardTitle>
            <CardDescription>Basic role details</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Role Name *</Label>
              <Input
                id="name"
                placeholder="developer"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                className={errors.name ? 'border-red-500' : ''}
              />
              {errors.name && (
                <p className="text-sm text-red-500">{errors.name}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea
                id="description"
                placeholder="Describe the purpose and scope of this role"
                value={formData.description}
                onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                rows={3}
              />
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Permissions</CardTitle>
            <CardDescription>Select permissions for this role</CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            {AVAILABLE_PERMISSIONS.map((category) => {
              const categoryIds = category.permissions.map((p) => p.id);
              const allSelected = categoryIds.every((id) => formData.permissions.includes(id));

              return (
                <div key={category.category} className="space-y-3">
                  <div className="flex items-center justify-between">
                    <h4 className="text-sm font-medium">{category.category}</h4>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => handleSelectAll(category.permissions)}
                    >
                      {allSelected ? 'Deselect All' : 'Select All'}
                    </Button>
                  </div>
                  <div className="grid gap-3 md:grid-cols-2 lg:grid-cols-3">
                    {category.permissions.map((permission) => (
                      <div key={permission.id} className="flex items-center space-x-2">
                        <Checkbox
                          id={permission.id}
                          checked={formData.permissions.includes(permission.id)}
                          onCheckedChange={() => handlePermissionToggle(permission.id)}
                        />
                        <label
                          htmlFor={permission.id}
                          className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer"
                        >
                          {permission.label}
                        </label>
                      </div>
                    ))}
                  </div>
                </div>
              );
            })}
            {errors.permissions && (
              <p className="text-sm text-red-500">{errors.permissions}</p>
            )}
          </CardContent>
        </Card>

        {errors.submit && (
          <div className="p-3 border border-red-500 bg-red-50 text-red-500 rounded">
            {errors.submit}
          </div>
        )}

        <div className="flex justify-end gap-4">
          <Button type="button" variant="outline" onClick={() => navigate('/roles')}>
            Cancel
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            <Save className="mr-2 h-4 w-4" />
            {isSubmitting ? 'Saving...' : isEdit ? 'Update Role' : 'Create Role'}
          </Button>
        </div>
      </form>
    </div>
  );
}
