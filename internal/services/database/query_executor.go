package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ysicing/tiga/internal/models"
	"github.com/ysicing/tiga/pkg/dbdriver"

	dbrepo "github.com/ysicing/tiga/internal/repository/database"
)

// QueryExecutor executes SQL/Redis queries with safety controls.
type QueryExecutor struct {
	manager          *DatabaseManager
	querySessionRepo *dbrepo.QuerySessionRepository
	securityFilter   *SecurityFilter
	timeout          time.Duration
	maxResultBytes   int64
}

// QueryExecutorConfig allows overriding default timeout and result size limits.
type QueryExecutorConfig struct {
	Timeout        time.Duration
	MaxResultBytes int64
}

func defaultQueryExecutorConfig() QueryExecutorConfig {
	return QueryExecutorConfig{
		Timeout:        30 * time.Second,
		MaxResultBytes: 10 * 1024 * 1024,
	}
}

// NewQueryExecutor constructs a QueryExecutor.
func NewQueryExecutor(
	manager *DatabaseManager,
	querySessionRepo *dbrepo.QuerySessionRepository,
	securityFilter *SecurityFilter,
) *QueryExecutor {
	return NewQueryExecutorWithConfig(manager, querySessionRepo, securityFilter, nil)
}

// NewQueryExecutorWithConfig constructs a QueryExecutor with custom limits.
func NewQueryExecutorWithConfig(
	manager *DatabaseManager,
	querySessionRepo *dbrepo.QuerySessionRepository,
	securityFilter *SecurityFilter,
	cfg *QueryExecutorConfig,
) *QueryExecutor {
	options := defaultQueryExecutorConfig()
	if cfg != nil {
		if cfg.Timeout > 0 {
			options.Timeout = cfg.Timeout
		}
		if cfg.MaxResultBytes > 0 {
			options.MaxResultBytes = cfg.MaxResultBytes
		}
	}

	if securityFilter == nil {
		securityFilter = NewSecurityFilter()
	}
	return &QueryExecutor{
		manager:          manager,
		querySessionRepo: querySessionRepo,
		securityFilter:   securityFilter,
		timeout:          options.Timeout,
		maxResultBytes:   options.MaxResultBytes,
	}
}

// QueryExecutionRequest encapsulates input for a query execution.
type QueryExecutionRequest struct {
	InstanceID   uuid.UUID
	ExecutedBy   string
	DatabaseName string
	Query        string
	Limit        int
	ClientIP     string
}

// QueryExecutionResponse contains execution results.
type QueryExecutionResponse struct {
	Columns       []string                 `json:"columns,omitempty"`
	Rows          []map[string]interface{} `json:"rows,omitempty"`
	AffectedRows  int64                    `json:"affected_rows,omitempty"`
	RowCount      int                      `json:"row_count,omitempty"`
	ExecutionTime time.Duration            `json:"execution_time"`
	Truncated     bool                     `json:"truncated"`
	Message       string                   `json:"message,omitempty"`
}

// ExecuteQuery runs a query/command against the target instance.
func (e *QueryExecutor) ExecuteQuery(ctx context.Context, req QueryExecutionRequest) (*QueryExecutionResponse, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, errors.New("query cannot be empty")
	}

	start := time.Now()

	driver, instance, err := e.manager.GetConnectedDriver(ctx, req.InstanceID)
	if err != nil {
		return nil, err
	}

	queryType := detectQueryType(instance.Type, req.Query)

	switch normalizeDriverType(instance.Type) {
	case "mysql", "postgresql":
		if err := e.securityFilter.ValidateSQL(req.Query); err != nil {
			return nil, err
		}
	case "redis":
		if err := e.securityFilter.ValidateRedisCommand(req.Query); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported instance type: %s", instance.Type)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	driverReq := dbdriver.QueryRequest{
		Database: req.DatabaseName,
		Query:    req.Query,
		Limit:    req.Limit,
	}

	result, execErr := driver.ExecuteQuery(timeoutCtx, driverReq)
	duration := time.Since(start)

	session := &models.QuerySession{
		InstanceID:     req.InstanceID,
		ExecutedBy:     req.ExecutedBy,
		DatabaseName:   req.DatabaseName,
		QuerySQL:       req.Query,
		QueryType:      queryType,
		StartedAt:      start.UTC(),
		DurationMillis: int(duration / time.Millisecond),
		ClientIP:       req.ClientIP,
	}

	response := &QueryExecutionResponse{}

	if execErr != nil {
		session.Status = statusFromError(execErr, timeoutCtx.Err())
		session.ErrorMessage = execErr.Error()
		session.CompletedAt = timePtr(time.Now().UTC())
		_ = e.querySessionRepo.Create(ctx, session)
		return nil, execErr
	}

	if result == nil {
		result = &dbdriver.QueryResult{}
	}

	response.Columns = result.Columns
	response.Rows = result.Rows
	response.AffectedRows = result.AffectedRows
	response.RowCount = result.RowCount
	response.ExecutionTime = duration

	sizeBytes := estimateResultSize(result)
	session.BytesReturned = sizeBytes
	session.RowCount = result.RowCount
	session.CompletedAt = timePtr(time.Now().UTC())

	if sizeBytes > e.maxResultBytes {
		response.Truncated = true
		response.Message = fmt.Sprintf("result exceeded %d bytes and was truncated", e.maxResultBytes)
		response.Rows = nil
		response.Columns = nil
		session.Status = "truncated"
		session.BytesReturned = e.maxResultBytes
	} else {
		session.Status = "success"
	}

	if err := e.querySessionRepo.Create(ctx, session); err != nil {
		return response, fmt.Errorf("failed to record query session: %w", err)
	}

	return response, nil
}

func detectQueryType(instanceType, query string) string {
	switch normalizeDriverType(instanceType) {
	case "redis":
		return "REDIS_CMD"
	default:
		first := strings.ToUpper(extractFirstKeyword(query))
		switch first {
		case "SELECT":
			return "SELECT"
		case "INSERT":
			return "INSERT"
		case "UPDATE":
			return "UPDATE"
		case "DELETE":
			return "DELETE"
		default:
			return "OTHER"
		}
	}
}

func estimateResultSize(result *dbdriver.QueryResult) int64 {
	if result == nil {
		return 0
	}
	payload := map[string]interface{}{
		"columns":       result.Columns,
		"rows":          result.Rows,
		"affected_rows": result.AffectedRows,
		"row_count":     result.RowCount,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return 0
	}
	return int64(len(data))
}

func statusFromError(execErr error, ctxErr error) string {
	if errors.Is(ctxErr, context.DeadlineExceeded) || errors.Is(execErr, context.DeadlineExceeded) {
		return "timeout"
	}
	return "error"
}
