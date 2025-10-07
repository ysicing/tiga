import { z } from 'zod'

// T031-T034: 前端 Zod 验证 Schema

// Database Type Enum
export const DatabaseTypeSchema = z.enum(['mysql', 'postgresql', 'sqlite'])
export type DatabaseType = z.infer<typeof DatabaseTypeSchema>

// Database Configuration Schema
export const DatabaseConfigSchema = z.object({
  type: DatabaseTypeSchema,
  host: z.string().optional(),
  port: z.number().min(1).max(65535).optional(),
  database: z.string().min(1, 'Database name is required'),
  username: z.string().optional(),
  password: z.string().optional(),
  ssl_mode: z.string().optional(),
  charset: z.string().optional(),
}).superRefine((data, ctx) => {
  // MySQL and PostgreSQL require host, port, and username
  if (data.type !== 'sqlite') {
    if (!data.host) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['host'],
        message: 'Host is required for MySQL/PostgreSQL',
      })
    }
    if (!data.port) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['port'],
        message: 'Port is required for MySQL/PostgreSQL',
      })
    }
    if (!data.username) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ['username'],
        message: 'Username is required for MySQL/PostgreSQL',
      })
    }
  }
})

export type DatabaseConfig = z.infer<typeof DatabaseConfigSchema>

// Admin Account Schema
export const AdminAccountSchema = z.object({
  username: z.string()
    .min(3, 'Username must be at least 3 characters')
    .max(20, 'Username must be at most 20 characters')
    .regex(/^[a-zA-Z0-9_]+$/, 'Username must contain only letters, numbers, and underscores'),
  password: z.string()
    .min(8, 'Password must be at least 8 characters')
    .regex(/[a-z]/, 'Password must contain at least one lowercase letter')
    .regex(/[A-Z]/, 'Password must contain at least one uppercase letter')
    .regex(/[0-9]/, 'Password must contain at least one number'),
  confirm_password: z.string(),
  email: z.string().email('Invalid email format'),
}).refine((data) => data.password === data.confirm_password, {
  message: 'Passwords do not match',
  path: ['confirm_password'],
})

export type AdminAccount = z.infer<typeof AdminAccountSchema>

// System Settings Schema
export const LanguageSchema = z.enum(['zh-CN', 'en-US'])
export type Language = z.infer<typeof LanguageSchema>

export const SystemSettingsSchema = z.object({
  app_name: z.string()
    .min(1, 'App name is required')
    .max(50, 'App name must be at most 50 characters'),
  app_subtitle: z.string()
    .max(100, 'App subtitle must be at most 100 characters')
    .optional(),
  domain: z.string().min(1, 'Domain is required'),
  http_port: z.number()
    .min(1, 'Port must be between 1 and 65535')
    .max(65535, 'Port must be between 1 and 65535'),
  language: LanguageSchema,
  enable_analytics: z.boolean(),
})

export type SystemSettings = z.infer<typeof SystemSettingsSchema>

// Helper function to get default system settings
export const getDefaultSystemSettings = (): SystemSettings => ({
  app_name: 'Tiga Dashboard',
  app_subtitle: '',
  domain: 'localhost',
  http_port: 12306,
  language: 'zh-CN',
  enable_analytics: false,
})

// Complete Install Config
export const InstallConfigSchema = z.object({
  database: DatabaseConfigSchema,
  admin: AdminAccountSchema,
  settings: SystemSettingsSchema,
})

export type InstallConfig = z.infer<typeof InstallConfigSchema>

// API Response Types
export interface CheckDBResponse {
  success: boolean
  has_existing_data?: boolean
  schema_version?: string
  can_upgrade?: boolean
  error?: string
}

export interface ValidateResponse {
  valid: boolean
  errors?: Record<string, string>
}

export interface FinalizeResponse {
  success: boolean
  message?: string
  session_token?: string
  redirect_url?: string // 重定向URL（包含端口）
  needs_restart?: boolean // 是否需要重启
  restart_message?: string // 重启提示信息
  error?: string
}

export interface StatusResponse {
  installed: boolean
  redirect_to: string
}
