import React, { useEffect, useState } from 'react'
import { ArrowLeft, Save, X } from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'

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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface UserFormData {
  username: string
  email: string
  full_name: string
  password: string
  confirm_password: string
  status: string
  role_ids: string[]
}

interface Role {
  id: string
  name: string
  description: string
}

export default function UserFormPage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const isEdit = !!id

  const [formData, setFormData] = useState<UserFormData>({
    username: '',
    email: '',
    full_name: '',
    password: '',
    confirm_password: '',
    status: 'active',
    role_ids: [],
  })

  const [availableRoles, setAvailableRoles] = useState<Role[]>([])
  const [selectedRoles, setSelectedRoles] = useState<Role[]>([])
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errors, setErrors] = useState<Record<string, string>>({})

  const statusOptions = [
    { value: 'active', label: 'Active' },
    { value: 'inactive', label: 'Inactive' },
    { value: 'suspended', label: 'Suspended' },
  ]

  useEffect(() => {
    const fetchRoles = async () => {
      try {
        const response: any = await devopsAPI.roles.list()
        setAvailableRoles(response.data || [])
      } catch (error) {
        console.error('Failed to fetch roles:', error)
      }
    }

    fetchRoles()

    if (isEdit) {
      const fetchUser = async () => {
        try {
          const response: any = await devopsAPI.users.get(id!)
          const user = response.data
          setFormData({
            username: user.username,
            email: user.email,
            full_name: user.full_name || '',
            password: '',
            confirm_password: '',
            status: user.status,
            role_ids: user.role_ids || [],
          })

          // Set selected roles
          if (user.role_ids?.length) {
            const roles = availableRoles.filter((r) =>
              user.role_ids.includes(r.id)
            )
            setSelectedRoles(roles)
          }
        } catch (error) {
          console.error('Failed to fetch user:', error)
        }
      }

      fetchUser()
    }
  }, [id, isEdit])

  const handleAddRole = (roleId: string) => {
    const role = availableRoles.find((r) => r.id === roleId)
    if (role && !selectedRoles.find((r) => r.id === roleId)) {
      const newSelectedRoles = [...selectedRoles, role]
      setSelectedRoles(newSelectedRoles)
      setFormData({
        ...formData,
        role_ids: newSelectedRoles.map((r) => r.id),
      })
    }
  }

  const handleRemoveRole = (roleId: string) => {
    const newSelectedRoles = selectedRoles.filter((r) => r.id !== roleId)
    setSelectedRoles(newSelectedRoles)
    setFormData({
      ...formData,
      role_ids: newSelectedRoles.map((r) => r.id),
    })
  }

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {}

    if (!formData.username.trim()) {
      newErrors.username = 'Username is required'
    } else if (formData.username.length < 3) {
      newErrors.username = 'Username must be at least 3 characters'
    }

    if (!formData.email.trim()) {
      newErrors.email = 'Email is required'
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      newErrors.email = 'Invalid email format'
    }

    if (!isEdit) {
      if (!formData.password) {
        newErrors.password = 'Password is required'
      } else if (formData.password.length < 8) {
        newErrors.password = 'Password must be at least 8 characters'
      }

      if (formData.password !== formData.confirm_password) {
        newErrors.confirm_password = 'Passwords do not match'
      }
    } else if (formData.password) {
      if (formData.password.length < 8) {
        newErrors.password = 'Password must be at least 8 characters'
      }
      if (formData.password !== formData.confirm_password) {
        newErrors.confirm_password = 'Passwords do not match'
      }
    }

    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!validateForm()) {
      return
    }

    setIsSubmitting(true)

    try {
      const data: any = {
        username: formData.username,
        email: formData.email,
        full_name: formData.full_name,
        status: formData.status,
      }

      if (formData.password) {
        data.password = formData.password
      }

      if (isEdit) {
        await devopsAPI.users.update(id!, data)
        // Update roles separately
        if (formData.role_ids.length > 0) {
          await devopsAPI.users.assignRoles(id!, formData.role_ids)
        }
      } else {
        const response: any = await devopsAPI.users.create(data)
        // Assign roles after user creation
        if (formData.role_ids.length > 0 && response.data?.id) {
          await devopsAPI.users.assignRoles(response.data.id, formData.role_ids)
        }
      }

      navigate('/users')
    } catch (error: any) {
      console.error('Failed to save user:', error)
      setErrors({ submit: error.message || 'Failed to save user' })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/users')}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h2 className="text-3xl font-bold tracking-tight">
          {isEdit ? 'Edit User' : 'Create User'}
        </h2>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        <Card>
          <CardHeader>
            <CardTitle>User Information</CardTitle>
            <CardDescription>Basic user account details</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="username">Username *</Label>
                <Input
                  id="username"
                  placeholder="johndoe"
                  value={formData.username}
                  onChange={(e) =>
                    setFormData({ ...formData, username: e.target.value })
                  }
                  disabled={isEdit}
                  className={errors.username ? 'border-red-500' : ''}
                />
                {errors.username && (
                  <p className="text-sm text-red-500">{errors.username}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="email">Email *</Label>
                <Input
                  id="email"
                  type="email"
                  placeholder="john@example.com"
                  value={formData.email}
                  onChange={(e) =>
                    setFormData({ ...formData, email: e.target.value })
                  }
                  className={errors.email ? 'border-red-500' : ''}
                />
                {errors.email && (
                  <p className="text-sm text-red-500">{errors.email}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="full_name">Full Name</Label>
                <Input
                  id="full_name"
                  placeholder="John Doe"
                  value={formData.full_name}
                  onChange={(e) =>
                    setFormData({ ...formData, full_name: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="status">Status</Label>
                <Select
                  value={formData.status}
                  onValueChange={(value) =>
                    setFormData({ ...formData, status: value })
                  }
                >
                  <SelectTrigger id="status">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {statusOptions.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="password">
                  {isEdit ? 'New Password (optional)' : 'Password *'}
                </Label>
                <Input
                  id="password"
                  type="password"
                  placeholder={
                    isEdit ? 'Leave blank to keep current password' : '••••••••'
                  }
                  value={formData.password}
                  onChange={(e) =>
                    setFormData({ ...formData, password: e.target.value })
                  }
                  className={errors.password ? 'border-red-500' : ''}
                />
                {errors.password && (
                  <p className="text-sm text-red-500">{errors.password}</p>
                )}
              </div>

              <div className="space-y-2">
                <Label htmlFor="confirm_password">
                  Confirm Password {!isEdit && '*'}
                </Label>
                <Input
                  id="confirm_password"
                  type="password"
                  placeholder="••••••••"
                  value={formData.confirm_password}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      confirm_password: e.target.value,
                    })
                  }
                  className={errors.confirm_password ? 'border-red-500' : ''}
                />
                {errors.confirm_password && (
                  <p className="text-sm text-red-500">
                    {errors.confirm_password}
                  </p>
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Roles</CardTitle>
            <CardDescription>Assign roles to the user</CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex gap-2">
              <Select onValueChange={handleAddRole}>
                <SelectTrigger className="flex-1">
                  <SelectValue placeholder="Select a role to add" />
                </SelectTrigger>
                <SelectContent>
                  {availableRoles
                    .filter(
                      (role) => !selectedRoles.find((r) => r.id === role.id)
                    )
                    .map((role) => (
                      <SelectItem key={role.id} value={role.id}>
                        {role.name}
                      </SelectItem>
                    ))}
                </SelectContent>
              </Select>
            </div>
            {selectedRoles.length > 0 && (
              <div className="flex flex-wrap gap-2">
                {selectedRoles.map((role) => (
                  <Badge
                    key={role.id}
                    variant="secondary"
                    className="flex items-center gap-1"
                  >
                    {role.name}
                    <X
                      className="h-3 w-3 cursor-pointer"
                      onClick={() => handleRemoveRole(role.id)}
                    />
                  </Badge>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {errors.submit && (
          <div className="p-3 border border-red-500 bg-red-50 text-red-500 rounded">
            {errors.submit}
          </div>
        )}

        <div className="flex justify-end gap-4">
          <Button
            type="button"
            variant="outline"
            onClick={() => navigate('/users')}
          >
            Cancel
          </Button>
          <Button type="submit" disabled={isSubmitting}>
            <Save className="mr-2 h-4 w-4" />
            {isSubmitting
              ? 'Saving...'
              : isEdit
                ? 'Update User'
                : 'Create User'}
          </Button>
        </div>
      </form>
    </div>
  )
}
