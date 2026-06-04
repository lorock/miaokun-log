package cache

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Cache struct {
	Dir string
	TTL time.Duration
}

func New(dir string) (*Cache, error) {
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "miaokun-cache")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("创建缓存目录失败: %w", err)
	}
	return &Cache{Dir: dir, TTL: 24 * time.Hour}, nil
}

func (c *Cache) Key(src string, size int64, modTime time.Time) string {
	h := sha256.New()
	h.Write([]byte(src))
	h.Write([]byte(fmt.Sprintf("%d-%d", size, modTime.UnixNano())))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Cache) Get(src string, size int64, modTime time.Time) (string, error) {
	key := c.Key(src, size, modTime)
	cached := filepath.Join(c.Dir, key+".log")

	if fi, err := os.Stat(cached); err == nil {
		if time.Since(fi.ModTime()) < c.TTL {
			return cached, nil
		}
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return "", fmt.Errorf("打开源文件失败: %w", err)
	}
	defer srcFile.Close()

	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return "", fmt.Errorf("创建 gzip reader 失败: %w", err)
	}
	defer gzReader.Close()

	outFile, err := os.Create(cached)
	if err != nil {
		return "", fmt.Errorf("创建缓存文件失败: %w", err)
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, gzReader); err != nil {
		return "", fmt.Errorf("解压失败: %w", err)
	}

	return cached, nil
}
