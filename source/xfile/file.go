package xfile

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// FileSource 表示文件配置源
type FileSource struct {
	Path   string
	Format string // 例如 "json", "yaml", "toml"
}

// NewFileSource 创建一个新的文件配置源
func NewFileSource(path string) *FileSource {
	// 自动检测格式
	format := filepath.Ext(path)
	if format != "" {
		format = format[1:] // 移除点号
	}

	return &FileSource{
		Path:   path,
		Format: format,
	}
}

// Read 实现 Source 接口，从文件读取配置
func (f *FileSource) Read() (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(f.Path)

	if f.Format != "" {
		v.SetConfigType(f.Format)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", f.Path, err)
	}

	return v, nil
}

// Watch 实现 Source 接口，监听文件变更
func (f *FileSource) Watch() (func() error, <-chan *viper.Viper, error) {
	// 创建文件监听器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, fmt.Errorf("创建文件监听器失败: %w", err)
	}

	// 获取文件的绝对路径和目录
	absPath, err := filepath.Abs(f.Path)
	if err != nil {
		watcher.Close()
		return nil, nil, fmt.Errorf("获取配置文件绝对路径失败: %w", err)
	}

	dir := filepath.Dir(absPath)

	// 监听目录而不是文件本身，这样即使文件被重新创建也能捕获变更
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, nil, fmt.Errorf("添加目录 %s 到监听失败: %w", dir, err)
	}

	// 用于发送配置更新的通道
	configChan := make(chan *viper.Viper)

	// 启动 goroutine 监听文件变更
	go func() {
		filename := filepath.Base(absPath)
		var debounceTimer *time.Timer
		var mu sync.Mutex

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				// 检查是否是我们关注的文件
				if filepath.Base(event.Name) != filename {
					continue
				}

				// 检查是否是写入或创建事件
				if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
					continue
				}

				// 使用防抖动机制，避免频繁重新加载
				mu.Lock()
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(100*time.Millisecond, func() {
					// 重新读取配置
					v, err := f.Read()
					if err != nil {
						fmt.Printf("重新读取配置文件 %s 失败: %v\n", f.Path, err)
						return
					}

					// 发送新配置
					configChan <- v
				})
				mu.Unlock()

			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				fmt.Printf("文件监听错误: %v\n", err)
			}
		}
	}()

	// 返回停止函数和配置通道
	stopFunc := func() error {
		close(configChan)
		return watcher.Close()
	}

	return stopFunc, configChan, nil
}

// String 实现 Source 接口，返回源的描述
func (f *FileSource) String() string {
	return fmt.Sprintf("FileSource(%s)", f.Path)
}
