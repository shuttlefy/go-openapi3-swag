package config

import (
	"flag"
	"strings"
)

type Config struct {
	ScanDirs   StringSlice
	OutputFile string
}

// 定义自定义类型
type StringSlice []string

// 实现 flag.Value 接口
func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *StringSlice) Set(value string) error {
	*s = strings.Split(value, ",")
	return nil
}

func ParseFlags() *Config {
	cfg := &Config{}

	flag.Var(&cfg.ScanDirs, "dirs", "directories to scan for Go source files (repeatable)")
	flag.StringVar(&cfg.OutputFile, "output", "./docs/openapi.json", "output file path, *.json or *.yaml")
	flag.Parse()

	if len(cfg.ScanDirs) == 0 {
		cfg.ScanDirs = []string{"."}
	}
	return cfg
}
