package config

import (
	"os"
	"path/filepath"
)

// DefaultURLs 是未显式传入 URL 时使用的内置测试链接。
var DefaultURLs = []string{
	"https://img.mcloud.139.com/material_prod/material_media/20221128/1669626861087.png",
	"https://wxhls.mcloud.139.com/hls/M068756c0040acdfc2749d3e70b04f183d/single/video/0/1080/ts/000000.ts",
}

// DefaultChunkSize 保持 Python 版本的 256KB 分块大小。
const DefaultChunkSize = 256 * 1024

// Paths 保存配置和统计文件位置。
type Paths struct {
	// ConfigDir 是配置和统计文件所在目录。
	ConfigDir string
	// ConfigFile 是 config.json 的完整路径。
	ConfigFile string
	// StatsFile 是 stats.json 的完整路径。
	StatsFile string
}

// ResolvePaths 按 Python 版本语义解析持久化路径，并支持 Docker/NAS 挂载目录。
func ResolvePaths() Paths {
	dir := os.Getenv("TRAFFIC_CONSUMER_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".traffic_consumer")
	}
	dir, _ = filepath.Abs(os.ExpandEnv(dir))
	return Paths{
		ConfigDir:  dir,
		ConfigFile: filepath.Join(dir, "config.json"),
		StatsFile:  filepath.Join(dir, "stats.json"),
	}
}
