package main

import (
	"log"

	"github.com/shuttlefy/go-openapi3-swag/config"
	"github.com/shuttlefy/go-openapi3-swag/internal/builder"
	"github.com/shuttlefy/go-openapi3-swag/internal/extractor"
	"github.com/shuttlefy/go-openapi3-swag/internal/output"
	"github.com/shuttlefy/go-openapi3-swag/internal/parser"
)

func main() {
	cfg := config.ParseFlags()

	// 1. Parse
	p := &parser.GoParser{MaxDepth: cfg.ParseDepth}
	files, err := p.Parse(cfg.InputDirs)
	if err != nil {
		log.Fatalf("parse: %v", err)
	}
	log.Printf("parsed %d files", len(files))

	// 2. Extract
	e := &extractor.GoExtractor{}
	result, err := e.Extract(files)
	if err != nil {
		log.Fatalf("extract: %v", err)
	}

	// 3. Build（始终启用模块缓存懒加载）
	b := builder.NewBuilder()
	modInfo, err := parser.ParseGoMod(cfg.GoMod)
	if err != nil {
		log.Printf("warn: parse go.mod: %v (module loader disabled)", err)
	} else {
		b.SetLoader(builder.NewModuleLoader(modInfo, parser.ModuleCacheDir()))
	}
	doc, err := b.Build(result, files)
	if err != nil {
		log.Fatalf("build: %v", err)
	}

	// 4. Write
	if err := output.Write(doc, cfg.OutputFile); err != nil {
		log.Fatalf("write: %v", err)
	}
	log.Printf("wrote %s", cfg.OutputFile)
}
