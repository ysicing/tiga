package models

// T017: 定义安装配置数据模型

// InstallConfig 安装配置总览
type InstallConfig struct {
	Database DatabaseConfig `json:"database" yaml:"database"`
	Admin    AdminAccount   `json:"admin" yaml:"admin"`
	Settings SystemSettings `json:"settings" yaml:"settings"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Type     string `json:"type" yaml:"type" binding:"required,oneof=mysql postgresql sqlite"`
	Host     string `json:"host,omitempty" yaml:"host,omitempty"`
	Port     int    `json:"port,omitempty" yaml:"port,omitempty" binding:"omitempty,min=1,max=65535"`
	Database string `json:"database" yaml:"database" binding:"required"`
	Username string `json:"username,omitempty" yaml:"username,omitempty"`
	Password string `json:"password,omitempty" yaml:"password,omitempty"`
	SSLMode  string `json:"ssl_mode,omitempty" yaml:"ssl_mode,omitempty"`
	Charset  string `json:"charset,omitempty" yaml:"charset,omitempty"`
}

// CheckDatabaseRequest 数据库检查请求
type CheckDatabaseRequest struct {
	DatabaseConfig
	ConfirmReinstall bool `json:"confirm_reinstall"`
}

// AdminAccount 管理员账户
type AdminAccount struct {
	Username string `json:"username" yaml:"username" binding:"required,min=3,max=20,alphanum"`
	Password string `json:"password" yaml:"password" binding:"required,min=8"`
	Email    string `json:"email" yaml:"email" binding:"required,email"`
}

// SystemSettings 系统设置
type SystemSettings struct {
	AppName         string `json:"app_name" yaml:"app_name" binding:"required,min=1,max=50"`
	AppSubtitle     string `json:"app_subtitle" yaml:"app_subtitle" binding:"max=100"`
	Domain          string `json:"domain" yaml:"domain" binding:"required"`
	HTTPPort        int    `json:"http_port" yaml:"http_port" binding:"required,min=1,max=65535"`
	GRPCPort        int    `json:"grpc_port" yaml:"grpc_port" binding:"omitempty,min=1,max=65535"`
	Language        string `json:"language" yaml:"language" binding:"required,oneof=zh-CN en-US"`
	EnableAnalytics bool   `json:"enable_analytics" yaml:"enable_analytics"` // 是否允许匿名收集统计数据
}
