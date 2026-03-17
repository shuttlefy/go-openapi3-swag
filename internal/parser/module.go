package parser

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode"
)

// ModuleInfo 保存 go.mod 中的模块信息。
type ModuleInfo struct {
	Module  string            // 当前模块路径，如 "github.com/foo/bar"
	Require map[string]string // importPath → version
}

// ParseGoMod 解析 go.mod 文件，返回 ModuleInfo。
// 支持 require (...) 块和单行 require，自动去掉 // indirect 等行内注释。
func ParseGoMod(gomodPath string) (*ModuleInfo, error) {
	f, err := os.Open(gomodPath)
	if err != nil {
		return nil, fmt.Errorf("open go.mod %q: %w", gomodPath, err)
	}
	defer f.Close()

	info := &ModuleInfo{Require: make(map[string]string)}
	inRequireBlock := false

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 去掉行内注释
		if i := strings.Index(line, "//"); i != -1 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "module ") {
			info.Module = strings.TrimSpace(strings.TrimPrefix(line, "module "))
			continue
		}

		if line == "require (" {
			inRequireBlock = true
			continue
		}
		if inRequireBlock {
			if line == ")" {
				inRequireBlock = false
				continue
			}
			// "github.com/gin-gonic/gin v1.9.1"
			if parts := strings.Fields(line); len(parts) >= 2 {
				info.Require[parts[0]] = parts[1]
			}
			continue
		}
		if strings.HasPrefix(line, "require ") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "require "))
			if parts := strings.Fields(rest); len(parts) >= 2 {
				info.Require[parts[0]] = parts[1]
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan go.mod: %w", err)
	}
	return info, nil
}

// ModuleCacheDir 返回 Go 模块缓存目录。
// 优先使用 go env GOMODCACHE，fallback 到 $GOPATH/pkg/mod，再 fallback 到 ~/go/pkg/mod。
func ModuleCacheDir() string {
	if out, err := exec.Command("go", "env", "GOMODCACHE").Output(); err == nil {
		if dir := strings.TrimSpace(string(out)); dir != "" {
			return dir
		}
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		return filepath.Join(gopath, "pkg", "mod")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "go", "pkg", "mod")
}

// ResolvePackageDir 根据 importPath 在模块缓存中定位包目录。
// 使用最长前缀匹配从 info.Require 中找到对应模块，
// 返回 (dir, true) 表示找到，("", false) 表示未找到。
func ResolvePackageDir(importPath string, info *ModuleInfo, cacheDir string) (string, bool) {
	best, bestVer := "", ""
	for modPath, ver := range info.Require {
		if importPath == modPath || strings.HasPrefix(importPath, modPath+"/") {
			if len(modPath) > len(best) {
				best = modPath
				bestVer = ver
			}
		}
	}
	if best == "" {
		return "", false
	}
	subPkg := strings.TrimPrefix(strings.TrimPrefix(importPath, best), "/")
	dir := filepath.Join(cacheDir, escapeModulePath(best)+"@"+bestVer, filepath.FromSlash(subPkg))
	return dir, true
}

// escapeModulePath 将模块路径中的大写字母转换为 !小写，
// 符合 Go module 缓存目录命名规范（例如 "github.com/BurntSushi" → "github.com/!burnt!sushi"）。
func escapeModulePath(p string) string {
	var sb strings.Builder
	for _, r := range p {
		if unicode.IsUpper(r) {
			sb.WriteByte('!')
			sb.WriteRune(unicode.ToLower(r))
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
