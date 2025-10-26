package models

import (
	"encoding/json"
	"fmt"
	"time"
)

// AuditEvent 统一的审计事件模型
// 用途：记录所有关键操作的审计日志，支持追溯和合规审查。不可修改和删除。
//
// 参考：.claude/specs/006-gitness-tiga/data-model.md 实体 4
//
//	.claude/specs/006-gitness-tiga/audit-unification.md 方案 A
type AuditEvent struct {
	// 基础字段
	ID        string `gorm:"type:varchar(255);primaryKey" json:"id"`                                                                 // UUID
	Timestamp int64  `gorm:"not null;index:idx_audit_events_timestamp;index:idx_audit_events_composite,priority:3" json:"timestamp"` // Unix 毫秒时间戳

	// 操作信息
	Action       Action        `gorm:"type:varchar(64);not null;index:idx_audit_events_action;index:idx_audit_events_composite,priority:2" json:"action"`
	ResourceType ResourceType  `gorm:"type:varchar(64);not null;index:idx_audit_events_resource_type;index:idx_audit_events_composite,priority:1" json:"resource_type"`
	Resource     Resource      `gorm:"type:text;serializer:json" json:"resource"`                                                  // JSON 序列化
	Subsystem    SubsystemType `gorm:"type:varchar(32);not null;index:idx_audit_events_subsystem;default:'http'" json:"subsystem"` // 子系统标识

	// 操作主体
	User      Principal `gorm:"type:text;serializer:json" json:"user"`
	SpacePath string    `gorm:"type:varchar(512)" json:"space_path,omitempty"` // 空间路径（可选）

	// 差异对象（变更前后）
	DiffObject DiffObject `gorm:"type:text;serializer:json" json:"diff_object,omitempty"`

	// 客户端信息
	ClientIP      string `gorm:"type:varchar(45);index:idx_audit_events_client_ip" json:"client_ip"`
	UserAgent     string `gorm:"type:text" json:"user_agent,omitempty"`
	RequestMethod string `gorm:"type:varchar(16)" json:"request_method,omitempty"` // GET, POST, etc.
	RequestID     string `gorm:"type:varchar(128);index:idx_audit_events_request_id" json:"request_id,omitempty"`

	// 自定义数据
	Data map[string]string `gorm:"type:text;serializer:json" json:"data,omitempty"`

	// 时间戳（仅创建）
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

// TableName 指定表名
func (AuditEvent) TableName() string {
	return "audit_events"
}

// GetID 实现 audit.AuditLog 接口 - 返回审计日志 ID
func (ae *AuditEvent) GetID() string {
	return ae.ID
}

// SetCreatedAt 实现 audit.AuditLog 接口 - 设置创建时间
func (ae *AuditEvent) SetCreatedAt(t time.Time) {
	ae.CreatedAt = t
}

// Resource 资源定义
type Resource struct {
	Type       ResourceType      `json:"type"`
	Identifier string            `json:"identifier"`     // 资源 ID
	Data       map[string]string `json:"data,omitempty"` // 资源元数据（如 resourceName、clusterName）
}

// Principal 操作主体
type Principal struct {
	UID      string        `json:"uid"`
	Username string        `json:"username"`
	Type     PrincipalType `json:"type"`
}

// PrincipalType 主体类型枚举
type PrincipalType string

const (
	// PrincipalTypeUser 用户
	PrincipalTypeUser PrincipalType = "user"
	// PrincipalTypeService 服务账号
	PrincipalTypeService PrincipalType = "service"
	// PrincipalTypeAnonymous 匿名用户
	PrincipalTypeAnonymous PrincipalType = "anonymous"
	// PrincipalTypeSystem 系统操作
	PrincipalTypeSystem PrincipalType = "system"
)

// Validate 验证主体类型有效性
func (pt PrincipalType) Validate() error {
	switch pt {
	case PrincipalTypeUser, PrincipalTypeService, PrincipalTypeAnonymous, PrincipalTypeSystem:
		return nil
	default:
		return fmt.Errorf("invalid principal type: %s", pt)
	}
}

// String 返回字符串表示
func (pt PrincipalType) String() string {
	return string(pt)
}

// DiffObject 差异对象
type DiffObject struct {
	OldObject          string   `json:"old_object,omitempty"` // JSON 字符串，最大 64KB
	NewObject          string   `json:"new_object,omitempty"` // JSON 字符串，最大 64KB
	OldObjectTruncated bool     `json:"old_object_truncated"`
	NewObjectTruncated bool     `json:"new_object_truncated"`
	TruncatedFields    []string `json:"truncated_fields,omitempty"` // 被截断的字段列表
}

// HasDiff 判断是否有差异对象
func (d *DiffObject) HasDiff() bool {
	return d.OldObject != "" || d.NewObject != ""
}

// IsTruncated 判断是否有截断
func (d *DiffObject) IsTruncated() bool {
	return d.OldObjectTruncated || d.NewObjectTruncated
}

// Action 操作类型枚举
type Action string

const (
	// 基础 CRUD 操作
	ActionCreated Action = "created"
	ActionUpdated Action = "updated"
	ActionDeleted Action = "deleted"
	ActionRead    Action = "read" // 敏感资源读取操作

	// 状态变更操作
	ActionEnabled  Action = "enabled"
	ActionDisabled Action = "disabled"

	// 特殊操作（参考 Gitness）
	ActionBypassed  Action = "bypassed"  // 绕过检查
	ActionForcePush Action = "forcePush" // 强制推送

	// 认证操作
	ActionLogin  Action = "login"
	ActionLogout Action = "logout"

	// 权限操作
	ActionGranted Action = "granted"
	ActionRevoked Action = "revoked"

	// Host 管理操作 (T038)
	ActionAgentConnected    Action = "agent_connected"
	ActionAgentDisconnected Action = "agent_disconnected"
	ActionAgentReconnected  Action = "agent_reconnected"
	ActionTerminalCreated   Action = "terminal_created"
	ActionTerminalClosed    Action = "terminal_closed"
	ActionTerminalReplay    Action = "terminal_replay"
	ActionNodeCreated       Action = "node_created"
	ActionNodeUpdated       Action = "node_updated"
	ActionNodeDeleted       Action = "node_deleted"
	ActionSystemAlert       Action = "system_alert"
	ActionSystemError       Action = "system_error"
)

// Validate 验证操作类型有效性
func (a Action) Validate() error {
	switch a {
	case ActionCreated, ActionUpdated, ActionDeleted, ActionRead,
		ActionEnabled, ActionDisabled, ActionBypassed, ActionForcePush,
		ActionLogin, ActionLogout, ActionGranted, ActionRevoked,
		// Host 管理操作
		ActionAgentConnected, ActionAgentDisconnected, ActionAgentReconnected,
		ActionTerminalCreated, ActionTerminalClosed, ActionTerminalReplay,
		ActionNodeCreated, ActionNodeUpdated, ActionNodeDeleted,
		ActionSystemAlert, ActionSystemError:
		return nil
	default:
		return fmt.Errorf("invalid action: %s", a)
	}
}

// String 返回字符串表示
func (a Action) String() string {
	return string(a)
}

// ResourceType 资源类型枚举
type ResourceType string

const (
	// Kubernetes 资源
	ResourceTypeCluster    ResourceType = "cluster"
	ResourceTypePod        ResourceType = "pod"
	ResourceTypeDeployment ResourceType = "deployment"
	ResourceTypeService    ResourceType = "service"
	ResourceTypeConfigMap  ResourceType = "configMap"
	ResourceTypeSecret     ResourceType = "secret"

	// 数据库资源
	ResourceTypeDatabase         ResourceType = "database"
	ResourceTypeDatabaseInstance ResourceType = "databaseInstance"
	ResourceTypeDatabaseUser     ResourceType = "databaseUser"

	// 中间件资源
	ResourceTypeMinIO      ResourceType = "minio"
	ResourceTypeRedis      ResourceType = "redis"
	ResourceTypeMySQL      ResourceType = "mysql"
	ResourceTypePostgreSQL ResourceType = "postgresql"

	// 系统资源
	ResourceTypeUser     ResourceType = "user"
	ResourceTypeRole     ResourceType = "role"
	ResourceTypeInstance ResourceType = "instance"

	// Scheduler 资源
	ResourceTypeScheduledTask ResourceType = "scheduledTask"

	// Host 资源（T038）
	ResourceTypeHost ResourceType = "host"

	// Docker 资源（T036-T037）
	ResourceTypeDockerInstance  ResourceType = "docker_instance"
	ResourceTypeDockerContainer ResourceType = "docker_container"
	ResourceTypeDockerImage     ResourceType = "docker_image"
	ResourceTypeDockerNetwork   ResourceType = "docker_network"
	ResourceTypeDockerVolume    ResourceType = "docker_volume"
	ResourceTypeDockerSystem    ResourceType = "docker_system"
	ResourceTypeDockerRecording ResourceType = "docker_recording"
)

// Validate 验证资源类型有效性
func (rt ResourceType) Validate() error {
	switch rt {
	case ResourceTypeCluster, ResourceTypePod, ResourceTypeDeployment,
		ResourceTypeService, ResourceTypeConfigMap, ResourceTypeSecret,
		ResourceTypeDatabase, ResourceTypeDatabaseInstance, ResourceTypeDatabaseUser,
		ResourceTypeMinIO, ResourceTypeRedis, ResourceTypeMySQL, ResourceTypePostgreSQL,
		ResourceTypeUser, ResourceTypeRole, ResourceTypeInstance,
		ResourceTypeScheduledTask, ResourceTypeHost,
		// Docker 资源 (T036-T037)
		ResourceTypeDockerInstance, ResourceTypeDockerContainer, ResourceTypeDockerImage,
		ResourceTypeDockerNetwork, ResourceTypeDockerVolume, ResourceTypeDockerSystem,
		ResourceTypeDockerRecording:
		return nil
	default:
		return fmt.Errorf("invalid resource type: %s", rt)
	}
}

// String 返回字符串表示
func (rt ResourceType) String() string {
	return string(rt)
}

// Validate 验证 AuditEvent 数据有效性
func (ae *AuditEvent) Validate() error {
	// 验证必填字段
	if ae.ID == "" {
		return fmt.Errorf("id is required")
	}
	if ae.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be positive")
	}

	// 验证枚举
	if err := ae.Action.Validate(); err != nil {
		return err
	}
	if err := ae.ResourceType.Validate(); err != nil {
		return err
	}
	// T012: Validate Subsystem enum
	if err := ae.Subsystem.Validate(); err != nil {
		return err
	}
	if err := ae.User.Type.Validate(); err != nil {
		return err
	}

	// 验证用户信息
	if ae.User.UID == "" {
		return fmt.Errorf("user.uid is required")
	}
	if ae.User.Username == "" {
		return fmt.Errorf("user.username is required")
	}

	// 验证资源信息
	if ae.Resource.Type != ae.ResourceType {
		return fmt.Errorf("resource.type (%s) must match resource_type (%s)",
			ae.Resource.Type, ae.ResourceType)
	}
	if ae.Resource.Identifier == "" {
		return fmt.Errorf("resource.identifier is required")
	}

	// 验证对象大小（64KB 限制）
	const maxObjectSize = 64 * 1024
	if len(ae.DiffObject.OldObject) > maxObjectSize {
		return fmt.Errorf("old_object size (%d bytes) exceeds 64KB limit",
			len(ae.DiffObject.OldObject))
	}
	if len(ae.DiffObject.NewObject) > maxObjectSize {
		return fmt.Errorf("new_object size (%d bytes) exceeds 64KB limit",
			len(ae.DiffObject.NewObject))
	}

	// 验证 ClientIP
	if ae.ClientIP == "" {
		return fmt.Errorf("client_ip is required")
	}

	return nil
}

// MarshalOldObject 序列化 OldObject 到 JSON
func (ae *AuditEvent) MarshalOldObject(obj interface{}) error {
	if obj == nil {
		ae.DiffObject.OldObject = ""
		return nil
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal old object: %w", err)
	}

	ae.DiffObject.OldObject = string(data)
	return nil
}

// MarshalNewObject 序列化 NewObject 到 JSON
func (ae *AuditEvent) MarshalNewObject(obj interface{}) error {
	if obj == nil {
		ae.DiffObject.NewObject = ""
		return nil
	}

	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal new object: %w", err)
	}

	ae.DiffObject.NewObject = string(data)
	return nil
}

// UnmarshalOldObject 反序列化 OldObject
func (ae *AuditEvent) UnmarshalOldObject(target interface{}) error {
	if ae.DiffObject.OldObject == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(ae.DiffObject.OldObject), target); err != nil {
		return fmt.Errorf("failed to unmarshal old object: %w", err)
	}

	return nil
}

// UnmarshalNewObject 反序列化 NewObject
func (ae *AuditEvent) UnmarshalNewObject(target interface{}) error {
	if ae.DiffObject.NewObject == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(ae.DiffObject.NewObject), target); err != nil {
		return fmt.Errorf("failed to unmarshal new object: %w", err)
	}

	return nil
}

// SubsystemType 子系统类型枚举
// T012: 用于区分不同子系统的审计事件
//
// Reference: .claude/specs/006-gitness-tiga/audit-unification.md
//
// 统一维护所有 Tiga 子系统的审计类型
type SubsystemType string

const (
	// SubsystemHTTP HTTP API 审计（通用 API 操作）
	SubsystemHTTP SubsystemType = "http"

	// 存储和中间件子系统
	// SubsystemMinIO MinIO 对象存储操作审计
	SubsystemMinIO SubsystemType = "minio"
	// SubsystemDatabase 数据库管理操作审计
	SubsystemDatabase SubsystemType = "database"
	// SubsystemMiddleware 中间件管理操作审计（Redis、MySQL、PostgreSQL 等）
	SubsystemMiddleware SubsystemType = "middleware"

	// 容器和编排子系统
	// SubsystemKubernetes Kubernetes 集群管理审计
	SubsystemKubernetes SubsystemType = "kubernetes"
	// SubsystemDocker Docker 实例管理审计
	SubsystemDocker SubsystemType = "docker"

	// 主机管理子系统
	// SubsystemHost 主机监控和管理审计（VMs）
	SubsystemHost SubsystemType = "host"
	// SubsystemWebSSH SSH 终端会话审计
	SubsystemWebSSH SubsystemType = "webssh"

	// 系统管理子系统
	// SubsystemScheduler 调度器任务审计
	SubsystemScheduler SubsystemType = "scheduler"
	// SubsystemAlert 告警系统审计
	SubsystemAlert SubsystemType = "alert"
	// SubsystemAuth 认证和授权审计
	SubsystemAuth SubsystemType = "auth"

	// 其他子系统
	// SubsystemStorage 存储管理审计
	SubsystemStorage SubsystemType = "storage"
	// SubsystemWebServer Web 服务器管理审计
	SubsystemWebServer SubsystemType = "webserver"
)

// Validate 验证子系统类型有效性
func (st SubsystemType) Validate() error {
	switch st {
	case SubsystemHTTP, SubsystemMinIO, SubsystemDatabase, SubsystemScheduler,
		SubsystemKubernetes, SubsystemDocker, SubsystemHost, SubsystemWebSSH,
		SubsystemAlert, SubsystemAuth, SubsystemMiddleware, SubsystemStorage,
		SubsystemWebServer:
		return nil
	default:
		return fmt.Errorf("invalid subsystem type: %s", st)
	}
}

// String 返回字符串表示
func (st SubsystemType) String() string {
	return string(st)
}
