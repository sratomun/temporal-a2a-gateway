package main

// A2A Error Codes
// Standardized error codes for Agent-to-Agent protocol

// JSON-RPC 2.0 Standard Error Codes
const (
	// Standard JSON-RPC errors
	ErrorParseError     = -32700 // Invalid JSON was received by the server
	ErrorInvalidRequest = -32600 // The JSON sent is not a valid Request object
	ErrorMethodNotFound = -32601 // The method does not exist / is not available
	ErrorInvalidParams  = -32602 // Invalid method parameter(s)
	ErrorInternalError  = -32603 // Internal JSON-RPC error
)

// A2A Protocol Specific Error Codes (-32000 to -32099)
const (
	// Task Management Errors (-32000 to -32019)
	ErrorTaskNotFound      = -32000 // Task with specified ID does not exist
	ErrorTaskStateInvalid  = -32001 // Task is in invalid state for operation
	ErrorTaskCreationFailed = -32002 // Failed to create new task
	ErrorTaskUpdateFailed  = -32003 // Failed to update task status
	ErrorTaskCancelFailed  = -32004 // Failed to cancel task
	ErrorTaskTimeout       = -32005 // Task execution timeout
	ErrorTaskQuotaExceeded = -32006 // Task quota or rate limit exceeded
	
	// Agent Management Errors (-32020 to -32039)
	ErrorAgentNotFound        = -32020 // Agent with specified ID does not exist
	ErrorAgentUnavailable     = -32021 // Agent is not available or offline
	ErrorAgentIncompatible    = -32022 // Agent incompatible with request
	ErrorAgentRegistrationFailed = -32023 // Failed to register agent
	ErrorAgentCapabilityMismatch = -32024 // Agent lacks required capability
	ErrorAgentQuotaExceeded   = -32025 // Agent quota or rate limit exceeded
	
	// Authentication & Authorization Errors (-32040 to -32059)
	ErrorUnauthorized         = -32040 // Invalid or missing authentication
	ErrorForbidden           = -32041 // Insufficient permissions
	ErrorInvalidAPIKey       = -32042 // Invalid API key format or value
	ErrorAPIKeyExpired       = -32043 // API key has expired
	ErrorRateLimitExceeded   = -32044 // Rate limit exceeded
	ErrorQuotaExceeded       = -32045 // Usage quota exceeded
	
	// Service Integration Errors (-32060 to -32079)
	ErrorTemporalConnection  = -32060 // Failed to connect to Temporal
	ErrorRedisConnection     = -32061 // Failed to connect to Redis
	ErrorDatabaseConnection  = -32062 // Failed to connect to database
	ErrorAgentRegistryConnection = -32063 // Failed to connect to Agent Registry
	ErrorExternalServiceTimeout = -32064 // External service timeout
	ErrorExternalServiceError = -32065 // External service returned error
	
	// Validation Errors (-32080 to -32099)
	ErrorValidationFailed    = -32080 // General validation error
	ErrorInvalidMessageFormat = -32081 // Invalid A2A message format
	ErrorInvalidConfiguration = -32082 // Invalid configuration detected
	ErrorEnvironmentInvalid  = -32083 // Invalid environment setup
	ErrorSchemaValidation    = -32084 // Schema validation failed
)

// A2A Error represents a structured error response
type A2AError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error Code Metadata
type ErrorCodeInfo struct {
	Code        int    `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Recoverable bool   `json:"recoverable"`
}

// GetErrorInfo returns detailed information about an error code
func GetErrorInfo(code int) ErrorCodeInfo {
	switch code {
	// JSON-RPC Standard Errors
	case ErrorParseError:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Parse Error",
			Description: "Invalid JSON was received by the server",
			Category:    "protocol",
			Recoverable: false,
		}
	case ErrorInvalidRequest:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Invalid Request",
			Description: "The JSON sent is not a valid Request object",
			Category:    "protocol",
			Recoverable: false,
		}
	case ErrorMethodNotFound:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Method Not Found",
			Description: "The method does not exist or is not available",
			Category:    "protocol",
			Recoverable: false,
		}
	case ErrorInvalidParams:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Invalid Parameters",
			Description: "Invalid method parameter(s)",
			Category:    "protocol",
			Recoverable: false,
		}
	case ErrorInternalError:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Internal Error",
			Description: "Internal JSON-RPC error",
			Category:    "system",
			Recoverable: true,
		}
		
	// Task Management Errors
	case ErrorTaskNotFound:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Task Not Found",
			Description: "Task with specified ID does not exist",
			Category:    "task",
			Recoverable: false,
		}
	case ErrorTaskStateInvalid:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Invalid Task State",
			Description: "Task is in invalid state for the requested operation",
			Category:    "task",
			Recoverable: false,
		}
	case ErrorTaskCreationFailed:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Task Creation Failed",
			Description: "Failed to create new task",
			Category:    "task",
			Recoverable: true,
		}
	case ErrorTaskTimeout:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Task Timeout",
			Description: "Task execution exceeded timeout limit",
			Category:    "task",
			Recoverable: true,
		}
		
	// Agent Management Errors
	case ErrorAgentNotFound:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Agent Not Found",
			Description: "Agent with specified ID does not exist",
			Category:    "agent",
			Recoverable: false,
		}
	case ErrorAgentUnavailable:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Agent Unavailable",
			Description: "Agent is not available or offline",
			Category:    "agent",
			Recoverable: true,
		}
	case ErrorAgentCapabilityMismatch:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Agent Capability Mismatch",
			Description: "Agent lacks required capability for this task",
			Category:    "agent",
			Recoverable: false,
		}
		
	// Authentication & Authorization Errors
	case ErrorUnauthorized:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Unauthorized",
			Description: "Invalid or missing authentication",
			Category:    "auth",
			Recoverable: false,
		}
	case ErrorForbidden:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Forbidden",
			Description: "Insufficient permissions for this operation",
			Category:    "auth",
			Recoverable: false,
		}
	case ErrorRateLimitExceeded:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Rate Limit Exceeded",
			Description: "Rate limit exceeded, please try again later",
			Category:    "limits",
			Recoverable: true,
		}
		
	// Service Integration Errors
	case ErrorTemporalConnection:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Temporal Connection Error",
			Description: "Failed to connect to Temporal workflow service",
			Category:    "service",
			Recoverable: true,
		}
	case ErrorRedisConnection:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Redis Connection Error",
			Description: "Failed to connect to Redis cache service",
			Category:    "service",
			Recoverable: true,
		}
	case ErrorExternalServiceTimeout:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "External Service Timeout",
			Description: "External service request timed out",
			Category:    "service",
			Recoverable: true,
		}
		
	// Validation Errors
	case ErrorValidationFailed:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Validation Failed",
			Description: "Request validation failed",
			Category:    "validation",
			Recoverable: false,
		}
	case ErrorInvalidMessageFormat:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Invalid Message Format",
			Description: "Invalid A2A message format",
			Category:    "validation",
			Recoverable: false,
		}
	case ErrorEnvironmentInvalid:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Invalid Environment",
			Description: "Invalid environment configuration detected",
			Category:    "validation",
			Recoverable: false,
		}
		
	default:
		return ErrorCodeInfo{
			Code:        code,
			Title:       "Unknown Error",
			Description: "Unknown error code",
			Category:    "unknown",
			Recoverable: false,
		}
	}
}

// NewA2AError creates a new A2A error with optional data
func NewA2AError(code int, message string, data interface{}) *A2AError {
	return &A2AError{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewA2AErrorFromCode creates a new A2A error from a predefined error code
func NewA2AErrorFromCode(code int, data interface{}) *A2AError {
	info := GetErrorInfo(code)
	return &A2AError{
		Code:    code,
		Message: info.Description,
		Data:    data,
	}
}

// IsRecoverable returns true if the error is recoverable (client can retry)
func (e *A2AError) IsRecoverable() bool {
	info := GetErrorInfo(e.Code)
	return info.Recoverable
}

// GetCategory returns the error category
func (e *A2AError) GetCategory() string {
	info := GetErrorInfo(e.Code)
	return info.Category
}