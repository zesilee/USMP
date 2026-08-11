package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenConf 是 schemagen 消费的 gen.conf 子集（yang_path/modules；其余键忽略——
// split_count 等属结构体生成面）。
type GenConf struct {
	YangPaths []string
	Modules   []string
}

// ParseGenConfAt 解析 gen.conf，yang_path 相对 repoRoot 展开为可用路径。
func ParseGenConfAt(confPath, repoRoot string) (*GenConf, error) {
	f, err := os.Open(confPath)
	if err != nil {
		return nil, fmt.Errorf("schemagen: open %s: %w", confPath, err)
	}
	defer f.Close()
	conf := &GenConf{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "yang_path":
			for _, d := range strings.Split(val, ",") {
				if d = strings.TrimSpace(d); d != "" {
					conf.YangPaths = append(conf.YangPaths, filepath.Join(repoRoot, d))
				}
			}
		case "modules":
			conf.Modules = strings.Fields(val)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(conf.YangPaths) == 0 || len(conf.Modules) == 0 {
		return nil, fmt.Errorf("schemagen: %s 缺少 yang_path 或 modules", confPath)
	}
	return conf, nil
}
