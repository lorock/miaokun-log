package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	DefaultPaths []string `mapstructure:"default_paths"`
	SinceDays    float64  `mapstructure:"since_days"`
	CacheDir     string   `mapstructure:"cache_dir"`
}

var cfg *Config

func Load(file string) error {
	v := viper.New()
	v.SetConfigName("miaokun")
	v.SetConfigType("yaml")

	if file != "" {
		v.SetConfigFile(file)
	} else {
		home, err := os.UserHomeDir()
		if err == nil {
			v.AddConfigPath(filepath.Join(home, ".config"))
			v.AddConfigPath(home)
		}
		v.AddConfigPath(".")
	}

	v.SetDefault("default_paths", []string{"/var/log", "/opt/logs"})
	v.SetDefault("since_days", 3)
	v.SetDefault("cache_dir", "/tmp/miaokun-cache")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("读取配置文件失败: %w", err)
		}
	}

	cfg = &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	return nil
}

func Get() *Config {
	if cfg == nil {
		cfg = &Config{
			DefaultPaths: []string{"/var/log", "/opt/logs"},
			SinceDays:    3,
			CacheDir:     "/tmp/miaokun-cache",
		}
	}
	return cfg
}
