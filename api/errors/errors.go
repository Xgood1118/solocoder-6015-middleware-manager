package errors

import (
  "log"
  "net/http"
  
  "github.com/gin-gonic/gin"
)

// APIError represents a standardized error response
type APIError struct {
  Code    int    `json:"code"`
  Message string `json:"message"`
  Details string `json:"details,omitempty"` // Optional detailed error info
}

// HandleAPIError logs and returns a standardized error response
func HandleAPIError(c *gin.Context, statusCode int, message string, err error) {
  // Log the error with details
  if err != nil {
    log.Printf("API Error: %s - %v", message, err)
    
    // Respond to the client with error details
    c.JSON(statusCode, APIError{
      Code:    statusCode,
      Message: message,
      Details: err.Error(),
    })
  } else {
    // Log just the message if no specific error is provided
    log.Printf("API Error: %s", message)
    
    // Respond to the client without error details
    c.JSON(statusCode, APIError{
      Code:    statusCode,
      Message: message,
    })
  }
}

// NotFound handles not found errors (404)
func NotFound(c *gin.Context, resourceType, id string) {
  message := resourceType + " not found"
  if id != "" {
    message += ": " + id
  }
  
  HandleAPIError(c, http.StatusNotFound, message, nil)
}

// BadRequest handles bad request errors (400)
func BadRequest(c *gin.Context, message string, err error) {
  HandleAPIError(c, http.StatusBadRequest, message, err)
}

// ServerError handles internal server errors (500)
func ServerError(c *gin.Context, message string, err error) {
  HandleAPIError(c, http.StatusInternalServerError, message, err)
}

// Unauthorized handles unauthorized errors (401)
func Unauthorized(c *gin.Context, message string) {
  if message == "" {
    message = "Unauthorized access"
  }
  HandleAPIError(c, http.StatusUnauthorized, message, nil)
}

// Forbidden handles forbidden errors (403)
func Forbidden(c *gin.Context, message string) {
  if message == "" {
    message = "Access forbidden"
  }
  HandleAPIError(c, http.StatusForbidden, message, nil)
}

// Conflict handles conflict errors (409)
func Conflict(c *gin.Context, message string, err error) {
  HandleAPIError(c, http.StatusConflict, message, err)
}

// UnprocessableEntity handles validation errors (422)
func UnprocessableEntity(c *gin.Context, message string, err error) {
  HandleAPIError(c, http.StatusUnprocessableEntity, message, err)
}

// ServiceUnavailable handles service unavailable errors (503)
func ServiceUnavailable(c *gin.Context, message string, err error) {
  HandleAPIError(c, http.StatusServiceUnavailable, message, err)
}