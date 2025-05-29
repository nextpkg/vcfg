package vcfg

import (
	"fmt"
	"strings"
)

// ErrorType 错误类型枚举
type ErrorType int

const (
	ErrorTypeUnknown ErrorType = iota
	ErrorTypeFileNotFound
	ErrorTypeParseFailure
	ErrorTypeValidationFailure
	ErrorTypeWatchFailure
	ErrorTypePluginFailure
	ErrorTypeMergeFailure
)

// String 返回错误类型的字符串表示
func (et ErrorType) String() string {
	switch et {
	case ErrorTypeFileNotFound:
		return "FileNotFound"
	case ErrorTypeParseFailure:
		return "ParseFailure"
	case ErrorTypeValidationFailure:
		return "ValidationFailure"
	case ErrorTypeWatchFailure:
		return "WatchFailure"
	case ErrorTypePluginFailure:
		return "PluginFailure"
	case ErrorTypeMergeFailure:
		return "MergeFailure"
	default:
		return "Unknown"
	}
}

// ConfigError 配置错误结构
type ConfigError struct {
	Type    ErrorType
	Source  string
	Message string
	Cause   error
}

// Error 实现 error 接口
func (e *ConfigError) Error() string {
	var parts []string

	if e.Type != ErrorTypeUnknown {
		parts = append(parts, fmt.Sprintf("[%s]", e.Type.String()))
	}

	if e.Source != "" {
		parts = append(parts, fmt.Sprintf("source=%s", e.Source))
	}

	if e.Message != "" {
		parts = append(parts, e.Message)
	}

	result := strings.Join(parts, " ")

	if e.Cause != nil {
		result += fmt.Sprintf(": %v", e.Cause)
	}

	return result
}

// Unwrap 返回底层错误
func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// Is 检查错误类型
func (e *ConfigError) Is(target error) bool {
	if ce, ok := target.(*ConfigError); ok {
		return e.Type == ce.Type
	}
	return false
}

// NewConfigError 创建新的配置错误
func NewConfigError(errType ErrorType, source, message string, cause error) *ConfigError {
	return &ConfigError{
		Type:    errType,
		Source:  source,
		Message: message,
		Cause:   cause,
	}
}

// 便捷的错误创建函数

// NewParseError 创建解析错误
func NewParseError(source, message string, cause error) *ConfigError {
	return NewConfigError(ErrorTypeParseFailure, source, message, cause)
}

// NewValidationError 创建验证错误
func NewValidationError(source, message string, cause error) *ConfigError {
	return NewConfigError(ErrorTypeValidationFailure, source, message, cause)
}
