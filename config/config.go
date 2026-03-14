package config

import "flag"

type Config struct {
	InputDirs    []string
	OutputFile   string
	OutputFormat string // "json" | "yaml"
	Title        string
	Version      string
	OpenAPIVer   string // "3.0.3" | "3.1.0" | "3.2.0"
}

func ParseFlags() *Config {
	cfg := &Config{}

	var dir string
	flag.StringVar(&dir, "dir", ".", "directory to scan for Go source files")
	flag.StringVar(&cfg.OutputFile, "output", "./openapi.json", "output file path")
	flag.StringVar(&cfg.OutputFormat, "format", "json", "output format: json or yaml")
	flag.StringVar(&cfg.Title, "title", "", "API title (overridden by annotation)")
	flag.StringVar(&cfg.Version, "version", "", "API version (overridden by annotation)")
	flag.StringVar(&cfg.OpenAPIVer, "openapi", "3.0.3", "OpenAPI version: 3.0.3, 3.1.0, 3.2.0")
	flag.Parse()

	cfg.InputDirs = []string{dir}
	if args := flag.Args(); len(args) > 0 {
		cfg.InputDirs = args
	}

	return cfg
}
