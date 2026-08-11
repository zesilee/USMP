//go:build tools

// Package tools 用空白导入把构建工具锁进 go.mod（Go 1.22 兼容做法）。
//
// Go 1.24 的 tool 指令因交付要求钉死 Go 1.22 而不可用，改用本文件承载：
// 版本仍由 go.mod 统一锁定，调用方式为 go run <pkg>（见 Makefile gen-contract、
// scripts/gen-yang.sh）。构建标签 tools 永不满足，不会进任何产物。
package tools

import (
	_ "github.com/swaggo/swag/cmd/swag"
	_ "golang.org/x/tools/cmd/goimports"
)
