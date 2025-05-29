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

// NewFileNotFoundError 创建文件未找到错误
func NewFileNotFoundError(filePath string, cause error) *ConfigError {
	return NewConfigError(ErrorTypeFileNotFound, filePath, "configuration file not found", cause)
}

// NewParseError 创建解析错误
func NewParseError(source, message string, cause error) *ConfigError {
	return NewConfigError(ErrorTypeParseFailure, source, message, cause)
}

// NewValidationError 创建验证错误
func NewValidationError(source, message string, cause error) *ConfigError {
	return NewConfigError(ErrorTypeValidationFailure, source, message, cause)
}

// NewWatchError 创建监听错误
func NewWatchError(source, message string, cause error) *ConfigError {
	return NewConfigError(ErrorTypeWatchFailure, source, message, cause)
}

// NewPluginError 创建插件错误
func NewPluginError(pluginName, message string, cause error) *ConfigError {
	return NewConfigError(ErrorTypePluginFailure, pluginName, message, cause)
}

// NewMergeError 创建合并错误
func NewMergeError(source, message string, cause error) *ConfigError {
	return NewConfigError(ErrorTypeMergeFailure, source, message, cause)
}

// ValidationErrors 验证错误集合
type ValidationErrors struct {
	Errors []ValidationError
}

// ValidationError 单个验证错误
type ValidationError struct {
	Field   string
	Value   interface{}
	Rule    string
	Message string
}

// Error 实现 error 接口
func (ve ValidationErrors) Error() string {
	if len(ve.Errors) == 0 {
		return "no validation errors"
	}

	if len(ve.Errors) == 1 {
		return ve.Errors[0].Error()
	}

	var messages []string
	for _, err := range ve.Errors {
		messages = append(messages, err.Error())
	}

	return fmt.Sprintf("multiple validation errors: %s", strings.Join(messages, "; "))
}

// Error 实现 error 接口
func (ve ValidationError) Error() string {
	if ve.Message != "" {
		return fmt.Sprintf("field '%s': %s", ve.Field, ve.Message)
	}

	if ve.Rule != "" {
		return fmt.Sprintf("field '%s' failed rule '%s' with value '%v'", ve.Field, ve.Rule, ve.Value)
	}

	return fmt.Sprintf("field '%s' validation failed with value '%v'", ve.Field, ve.Value)
}

// Add 添加验证错误
func (ve *ValidationErrors) Add(field, rule, message string, value interface{}) {
	ve.Errors = append(ve.Errors, ValidationError{
		Field:   field,
		Value:   value,
		Rule:    rule,
		Message: message,
	})
}

// HasErrors 检查是否有错误
func (ve ValidationErrors) HasErrors() bool {
	return len(ve.Errors) > 0
}
