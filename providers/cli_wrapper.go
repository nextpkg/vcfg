package providers

import (
	"log/slog"
	"maps"
	"strings"

	"github.com/knadh/koanf/v2"
)

// flattenMap 递归扁平化嵌套的map结构
func flattenMap(data map[string]any, prefix string, result map[string]any) {
	for key, value := range data {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if nestedMap, ok := value.(map[string]any); ok {
			// 递归处理嵌套的map
			flattenMap(nestedMap, fullKey, result)
		} else {
			// 叶子节点，直接设置值
			result[fullKey] = value
		}
	}
}

// CliProviderWrapper 包装 cliflagv3.Provider 来处理键名映射
type CliProviderWrapper struct {
	original koanf.Provider
	cmdName  string
	delim    string
}

// NewCliProviderWrapper 创建一个新的 CLI provider wrapper
func NewCliProviderWrapper(original koanf.Provider, cmdName, delim string) *CliProviderWrapper {
	return &CliProviderWrapper{
		original: original,
		cmdName:  cmdName,
		delim:    delim,
	}
}

// Read 实现 koanf.Provider 接口
func (w *CliProviderWrapper) Read() (map[string]any, error) {
	data, err := w.original.Read()
	if err != nil {
		return nil, err
	}

	// 处理键名映射，移除命令名前缀
	result := make(map[string]any)
	slog.Debug("cliProviderWrapper: original data", "data", data)
	slog.Debug("cliProviderWrapper: cmdName", "cmdName", w.cmdName, "delim", w.delim)

	// 如果分隔符为空，需要特殊处理：扁平化嵌套的map结构
	if w.delim == "" {
		slog.Debug("cliProviderWrapper: empty delimiter, flattening nested structure")

		// 递归扁平化嵌套的map结构
		flattenMap(data, "", result)

		// 移除命令名前缀的键，只保留实际的配置键
		// 扁平化后的键格式是: c.l.i.-.d.e.m.o.配置名
		cmdPrefixPattern := strings.Join(strings.Split(w.cmdName, ""), ".") + "."

		finalResult := make(map[string]any)
		for key, value := range result {
			if strings.HasPrefix(key, cmdPrefixPattern) {
				// 移除命令名前缀，得到实际的配置键名
				actualKey := strings.TrimPrefix(key, cmdPrefixPattern)
				// 移除点分隔符，恢复原始键名
				actualKey = strings.ReplaceAll(actualKey, ".", "")
				slog.Debug("cliProviderWrapper: mapping prefixed key", "from", key, "to", actualKey)
				finalResult[actualKey] = value
			}
		}

		slog.Debug("cliProviderWrapper: empty delimiter result", "result", finalResult)
		return finalResult, nil
	}

	// 检查是否有以命令名为键的嵌套 map
	if cmdData, exists := data[w.cmdName]; exists {
		if cmdMap, ok := cmdData.(map[string]any); ok {
			// 直接使用嵌套 map 的内容
			slog.Debug("cliProviderWrapper: found nested command data", "cmdData", cmdMap)
			maps.Copy(result, cmdMap)
		} else {
			// 如果不是 map，直接设置值
			result[w.cmdName] = cmdData
		}
	}

	// 处理其他键（不以命令名开头的）
	prefix := w.cmdName + w.delim
	for key, value := range data {
		if key == w.cmdName {
			// 跳过已处理的命令键
			continue
		}
		if strings.HasPrefix(key, prefix) {
			// 移除命令名前缀
			newKey := strings.TrimPrefix(key, prefix)
			slog.Debug("cliProviderWrapper: mapping key", "from", key, "to", newKey)
			result[newKey] = value
		} else {
			// 保持原键名
			slog.Debug("cliProviderWrapper: keeping key", "key", key)
			result[key] = value
		}
	}
	slog.Debug("cliProviderWrapper: result", "result", result)
	return result, nil
}

// ReadBytes 实现 koanf.Provider 接口
func (w *CliProviderWrapper) ReadBytes() ([]byte, error) {
	return w.original.ReadBytes()
}
