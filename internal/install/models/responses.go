package models

// T018: 定义 API 响应模型

// CheckDBResponse 数据库检查响应
type CheckDBResponse struct {
	Success         bool   `json:"success"`
	HasExistingData bool   `json:"has_existing_data,omitempty"`
	SchemaVersion   string `json:"schema_version,omitempty"`
	CanUpgrade      bool   `json:"can_upgrade,omitempty"`
	Error           string `json:"error,omitempty"`
}

// ValidateAdminResponse 管理员验证响应
type ValidateAdminResponse struct {
	Valid  bool              `json:"valid"`
	Errors map[string]string `json:"errors,omitempty"`
}

// ValidateSettingsResponse 系统设置验证响应
type ValidateSettingsResponse struct {
	Valid  bool              `json:"valid"`
	Errors map[string]string `json:"errors,omitempty"`
}

// FinalizeRequest 完成初始化请求
type FinalizeRequest struct {
	Database         DatabaseConfig `json:"database" binding:"required"`
	Admin            AdminAccount   `json:"admin" binding:"required"`
	Settings         SystemSettings `json:"settings" binding:"required"`
	ConfirmReinstall bool           `json:"confirm_reinstall"`
}

// FinalizeResponse 完成初始化响应
type FinalizeResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
	SessionToken   string `json:"session_token,omitempty"`
	RedirectURL    string `json:"redirect_url,omitempty"`    // 重定向URL（包含端口）
	NeedsRestart   bool   `json:"needs_restart,omitempty"`   // 是否需要重启
	RestartMessage string `json:"restart_message,omitempty"` // 重启提示信息
	Error          string `json:"error,omitempty"`
}

// StatusResponse 初始化状态响应
type StatusResponse struct {
	Installed  bool   `json:"installed"`
	RedirectTo string `json:"redirect_to"`
}
