package discover

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gitee.com/lorock/miaokun-log/pkg/types"
)

var candidateDirs = []string{
	"/var/log",
	"/var/log/app",
	"/var/log/nginx",
	"/var/log/mysql",
	"/var/log/postgresql",
	"/var/log/redis",
	"/var/log/kafka",
	"/opt/logs",
	"/opt/app/logs",
	"/usr/local/app/logs",
	"/tmp",
	"./logs",
}

type PathCandidate struct {
	Path      string `json:"path"`
	Exists    bool   `json:"exists"`
	FileCount int    `json:"file_count"`
	TotalSize int64  `json:"total_size"`
	IsDefault bool   `json:"is_default"`
}

func DiscoverPaths(sinceDays float64, defaultPaths ...string) []PathCandidate {
	var result []PathCandidate
	seen := make(map[string]bool)

	cutoff := time.Now().Add(-time.Duration(sinceDays * 24 * float64(time.Hour)))

	allDirs := candidateDirs
	if len(defaultPaths) > 0 {
		allDirs = append(defaultPaths, candidateDirs...)
	}

	defaultPathSet := make(map[string]bool)
	for _, p := range defaultPaths {
		absPath := p
		if !filepath.IsAbs(absPath) {
			if cwd, err := os.Getwd(); err == nil {
				absPath = filepath.Join(cwd, p)
			}
		}
		defaultPathSet[absPath] = true
	}

	if len(defaultPaths) == 0 {
		defaultPathSet["/var/log"] = true
		defaultPathSet["/opt/logs"] = true
	}

	for _, dir := range allDirs {
		absPath := dir
		if !filepath.IsAbs(absPath) {
			if cwd, err := os.Getwd(); err == nil {
				absPath = filepath.Join(cwd, dir)
			}
		}
		if seen[absPath] {
			continue
		}
		seen[absPath] = true

		info, err := os.Stat(absPath)
		if err != nil || !info.IsDir() {
			result = append(result, PathCandidate{
				Path:      absPath,
				Exists:    false,
				FileCount: 0,
				TotalSize: 0,
				IsDefault: defaultPathSet[absPath],
			})
			continue
		}

		count := 0
		var totalSize int64
		filepath.Walk(absPath, func(p string, fi os.FileInfo, werr error) error {
			if werr != nil || fi.IsDir() {
				return nil
			}
			name := strings.ToLower(fi.Name())
			if !strings.HasSuffix(name, ".log") &&
				!strings.HasSuffix(name, ".log.gz") &&
				!strings.Contains(name, ".log.") {
				return nil
			}
			if fi.ModTime().Before(cutoff) {
				return nil
			}
			count++
			totalSize += fi.Size()
			return nil
		})

		result = append(result, PathCandidate{
			Path:      absPath,
			Exists:    true,
			FileCount: count,
			TotalSize: totalSize,
			IsDefault: defaultPathSet[absPath],
		})
	}

	return result
}

func FindLogs(paths []string, sinceDays float64) ([]types.FileInfo, error) {
	var result []types.FileInfo
	cutoff := time.Now().Add(-time.Duration(sinceDays * 24 * float64(time.Hour)))

	for _, root := range paths {
		if _, err := os.Stat(root); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			name := strings.ToLower(info.Name())
			if !strings.HasSuffix(name, ".log") &&
				!strings.HasSuffix(name, ".log.gz") &&
				!strings.Contains(name, ".log.") {
				return nil
			}

			if info.ModTime().Before(cutoff) {
				return nil
			}

			result = append(result, types.FileInfo{
				Path:    path,
				Size:    info.Size(),
				ModTime: info.ModTime(),
				IsGzip:  strings.HasSuffix(name, ".gz"),
			})
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("遍历目录 %s 失败: %w", root, err)
		}
	}

	return result, nil
}
