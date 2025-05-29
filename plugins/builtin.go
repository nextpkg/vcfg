package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nextpkg/vcfg"
)

// BackupPlugin 配置备份插件
type BackupPlugin[T any] struct {
	backupDir  string
	maxBackups int
}

// NewBackupPlugin 创建备份插件
func NewBackupPlugin[T any](backupDir string, maxBackups int) *BackupPlugin[T] {
	return &BackupPlugin[T]{
		backupDir:  backupDir,
		maxBackups: maxBackups,
	}
}

func (p *BackupPlugin[T]) Name() string {
	return "backup"
}

func (p *BackupPlugin[T]) Initialize(ctx context.Context, manager *vcfg.ConfigManager[T]) error {
	// 确保备份目录存在
	if err := os.MkdirAll(p.backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	return nil
}

func (p *BackupPlugin[T]) OnConfigLoaded(ctx context.Context, config *T) error {
	return p.createBackup(config)
}

func (p *BackupPlugin[T]) OnConfigChanged(ctx context.Context, oldConfig, newConfig *T) error {
	return p.createBackup(newConfig)
}

func (p *BackupPlugin[T]) Shutdown(ctx context.Context) error {
	return nil
}

func (p *BackupPlugin[T]) createBackup(config *T) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// 确保备份目录存在
	if err := os.MkdirAll(p.backupDir, 0755); err != nil {
		return fmt.Errorf("failed to ensure backup directory: %w", err)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("config_backup_%s.json", timestamp)
	filePath := filepath.Join(p.backupDir, filename)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// 使用临时文件确保原子写入
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	// 原子重命名
	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile) // 清理临时文件
		return fmt.Errorf("failed to finalize backup file: %w", err)
	}

	// 清理旧备份
	return p.cleanupOldBackups()
}

func (p *BackupPlugin[T]) cleanupOldBackups() error {
	if p.maxBackups <= 0 {
		return nil // 不限制备份数量
	}

	files, err := filepath.Glob(filepath.Join(p.backupDir, "config_backup_*.json"))
	if err != nil {
		return fmt.Errorf("failed to list backup files: %w", err)
	}

	if len(files) <= p.maxBackups {
		return nil
	}

	// 按文件名排序（时间戳排序）
	// filepath.Glob 已经返回排序的结果

	// 删除最旧的文件
	for i := 0; i < len(files)-p.maxBackups; i++ {
		if err := os.Remove(files[i]); err != nil {
			// 记录错误但继续删除其他文件
			fmt.Printf("Warning: failed to remove old backup %s: %v\n", files[i], err)
		}
	}

	return nil
}

// MetricsPlugin 配置指标收集插件
type MetricsPlugin[T any] struct {
	loadCount   int64
	changeCount int64
	lastLoaded  time.Time
	lastChanged time.Time
}

// NewMetricsPlugin 创建指标插件
func NewMetricsPlugin[T any]() *MetricsPlugin[T] {
	return &MetricsPlugin[T]{}
}

func (p *MetricsPlugin[T]) Name() string {
	return "metrics"
}

func (p *MetricsPlugin[T]) Initialize(ctx context.Context, manager *vcfg.ConfigManager[T]) error {
	return nil
}

func (p *MetricsPlugin[T]) OnConfigLoaded(ctx context.Context, config *T) error {
	p.loadCount++
	p.lastLoaded = time.Now()
	fmt.Printf("[Metrics] Config loaded. Total loads: %d\n", p.loadCount)
	return nil
}

func (p *MetricsPlugin[T]) OnConfigChanged(ctx context.Context, oldConfig, newConfig *T) error {
	p.changeCount++
	p.lastChanged = time.Now()
	fmt.Printf("[Metrics] Config changed. Total changes: %d\n", p.changeCount)
	return nil
}

func (p *MetricsPlugin[T]) Shutdown(ctx context.Context) error {
	fmt.Printf("[Metrics] Plugin shutdown. Final stats - Loads: %d, Changes: %d\n", p.loadCount, p.changeCount)
	return nil
}

// GetStats 获取统计信息
func (p *MetricsPlugin[T]) GetStats() map[string]any {
	return map[string]any{
		"load_count":   p.loadCount,
		"change_count": p.changeCount,
		"last_loaded":  p.lastLoaded,
		"last_changed": p.lastChanged,
	}
}

// ValidationPlugin 增强验证插件
type ValidationPlugin[T any] struct {
	validators []func(*T) error
}

// NewValidationPlugin 创建验证插件
func NewValidationPlugin[T any]() *ValidationPlugin[T] {
	return &ValidationPlugin[T]{
		validators: make([]func(*T) error, 0),
	}
}

func (p *ValidationPlugin[T]) Name() string {
	return "validation"
}

func (p *ValidationPlugin[T]) Initialize(ctx context.Context, manager *vcfg.ConfigManager[T]) error {
	return nil
}

func (p *ValidationPlugin[T]) OnConfigLoaded(ctx context.Context, config *T) error {
	return p.validate(config)
}

func (p *ValidationPlugin[T]) OnConfigChanged(ctx context.Context, oldConfig, newConfig *T) error {
	return p.validate(newConfig)
}

func (p *ValidationPlugin[T]) Shutdown(ctx context.Context) error {
	return nil
}

// AddValidator 添加自定义验证器
func (p *ValidationPlugin[T]) AddValidator(validator func(*T) error) {
	p.validators = append(p.validators, validator)
}

func (p *ValidationPlugin[T]) validate(config *T) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	for i, validator := range p.validators {
		if validator == nil {
			continue
		}
		if err := validator(config); err != nil {
			return fmt.Errorf("validator %d failed: %w", i, err)
		}
	}
	return nil
}
