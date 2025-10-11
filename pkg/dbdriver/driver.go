package dbdriver

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotConnected indicates the driver has not established a connection.
	ErrNotConnected = errors.New("database driver is not connected")
	// ErrUnsupportedOperation is returned when the underlying engine does not support an operation.
	ErrUnsupportedOperation = errors.New("operation not supported by this driver")
	// ErrRowLimitExceeded is returned when query results exceed the maximum row limit.
	ErrRowLimitExceeded = errors.New("query result exceeds maximum row limit")
)

// ConnectionConfig contains connection parameters shared across drivers.
type ConnectionConfig struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	SSLMode         string
	Params          map[string]string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DatabaseInfo represents metadata about a logical database/schema.
type DatabaseInfo struct {
	Name       string                 `json:"name"`
	Charset    string                 `json:"charset,omitempty"`
	Collation  string                 `json:"collation,omitempty"`
	Owner      string                 `json:"owner,omitempty"`
	SizeBytes  int64                  `json:"size_bytes,omitempty"`
	TableCount int                    `json:"table_count,omitempty"`
	KeyCount   int                    `json:"key_count,omitempty"`
	Extra      map[string]interface{} `json:"extra,omitempty"`
}

// UserInfo represents metadata about a database-level user.
type UserInfo struct {
	Username    string                 `json:"username"`
	Host        string                 `json:"host,omitempty"`
	Roles       []string               `json:"roles,omitempty"`
	Permissions []string               `json:"permissions,omitempty"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

// CreateDatabaseOptions captures provider-specific options when creating databases.
type CreateDatabaseOptions struct {
	Name      string
	Charset   string
	Collation string
	Owner     string
}

// CreateUserOptions captures parameters for creating database users.
type CreateUserOptions struct {
	Username string
	Password string
	Host     string
	Roles    []string
	Extra    map[string]interface{}
}

// QueryRequest describes an arbitrary query/command to execute.
type QueryRequest struct {
	Database string
	Query    string
	Limit    int
	Args     []any
}

// QueryResult contains the response from executing a query/command.
type QueryResult struct {
	Columns       []string                 `json:"columns,omitempty"`
	Rows          []map[string]interface{} `json:"rows,omitempty"`
	AffectedRows  int64                    `json:"affected_rows,omitempty"`
	RowCount      int                      `json:"row_count,omitempty"`
	ExecutionTime time.Duration            `json:"execution_time,omitempty"`
	RawResponse   interface{}              `json:"raw_response,omitempty"`
}

// DatabaseDriver standardises how different engines are managed.
type DatabaseDriver interface {
	Connect(ctx context.Context, cfg ConnectionConfig) error
	Disconnect(ctx context.Context) error
	Ping(ctx context.Context) error

	ListDatabases(ctx context.Context) ([]DatabaseInfo, error)
	CreateDatabase(ctx context.Context, opts CreateDatabaseOptions) error
	DeleteDatabase(ctx context.Context, name string, opts map[string]interface{}) error

	ListUsers(ctx context.Context) ([]UserInfo, error)
	CreateUser(ctx context.Context, opts CreateUserOptions) error
	DeleteUser(ctx context.Context, username string, opts map[string]interface{}) error
	UpdateUserPassword(ctx context.Context, username, password string, opts map[string]interface{}) error

	ExecuteQuery(ctx context.Context, req QueryRequest) (*QueryResult, error)

	GetVersion(ctx context.Context) (string, error)
	GetUptime(ctx context.Context) (time.Duration, error)
}
