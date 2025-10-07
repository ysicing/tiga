package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SuccessResponse represents a successful API response
type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Success bool        `json:"success"`
	Error   string      `json:"error"`
	Message string      `json:"message,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination represents pagination metadata
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// RespondSuccess sends a success response
func RespondSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// RespondSuccessWithMessage sends a success response with a message
func RespondSuccessWithMessage(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// RespondCreated sends a 201 Created response
func RespondCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

// RespondNoContent sends a 204 No Content response
func RespondNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// RespondPaginated sends a paginated response
func RespondPaginated(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    data,
		Pagination: Pagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// RespondError sends an error response
func RespondError(c *gin.Context, statusCode int, err error) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error:   err.Error(),
	})
}

// RespondErrorWithMessage sends an error response with a custom message
func RespondErrorWithMessage(c *gin.Context, statusCode int, err error, message string) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error:   err.Error(),
		Message: message,
	})
}

// RespondErrorWithDetails sends an error response with details
func RespondErrorWithDetails(c *gin.Context, statusCode int, err error, details interface{}) {
	c.JSON(statusCode, ErrorResponse{
		Success: false,
		Error:   err.Error(),
		Details: details,
	})
}

// RespondBadRequest sends a 400 Bad Request response
func RespondBadRequest(c *gin.Context, err error) {
	RespondError(c, http.StatusBadRequest, err)
}

// RespondUnauthorized sends a 401 Unauthorized response
func RespondUnauthorized(c *gin.Context, err error) {
	RespondError(c, http.StatusUnauthorized, err)
}

// RespondForbidden sends a 403 Forbidden response
func RespondForbidden(c *gin.Context, err error) {
	RespondError(c, http.StatusForbidden, err)
}

// RespondNotFound sends a 404 Not Found response
func RespondNotFound(c *gin.Context, err error) {
	RespondError(c, http.StatusNotFound, err)
}

// RespondConflict sends a 409 Conflict response
func RespondConflict(c *gin.Context, err error) {
	RespondError(c, http.StatusConflict, err)
}

// RespondInternalError sends a 500 Internal Server Error response
func RespondInternalError(c *gin.Context, err error) {
	RespondError(c, http.StatusInternalServerError, err)
}

// BindJSON binds JSON request body and handles errors
func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		RespondBadRequest(c, err)
		return false
	}
	return true
}

// BindQuery binds query parameters and handles errors
func BindQuery(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindQuery(obj); err != nil {
		RespondBadRequest(c, err)
		return false
	}
	return true
}

// BindURI binds URI parameters and handles errors
func BindURI(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindUri(obj); err != nil {
		RespondBadRequest(c, err)
		return false
	}
	return true
}
