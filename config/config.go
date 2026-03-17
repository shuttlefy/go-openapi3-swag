package config

import (
	"flag"
	"strings"
)

type Config struct {
	InputDirs  StringSlice
	OutputFile string
	OpenAPIVer string // "3.0.3"（默认）| "3.1.0"
	ParseDepth int    // 目录递归深度：0=仅当前目录，N=最多N层，-1=无限
	GoMod      string // go.mod 文件路径（默认 ./go.mod）
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

	flag.Var(&cfg.InputDirs, "dirs", "directories to scan for Go source files (comma-separated or repeatable)")
	flag.StringVar(&cfg.OutputFile, "output", "./docs/openapi.json", "output file path (*.json or *.yaml)")
	flag.IntVar(&cfg.ParseDepth, "depth", -1, "directory recursion depth: 0=root only, N=N levels, -1=unlimited")
	flag.StringVar(&cfg.OpenAPIVer, "openapi-ver", "3.0.3", "OpenAPI version: 3.0.3 | 3.1.0")
	flag.StringVar(&cfg.GoMod, "gomod", "go.mod", "path to go.mod file")
	flag.Parse()

	if len(cfg.InputDirs) == 0 {
		cfg.InputDirs = []string{"."}
	}
	return cfg
}
