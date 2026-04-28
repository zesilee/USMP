//go:build generate
// +build generate

// fix_enum.go - 修复 ygot 生成的枚举值中包含非法字符的问题
// 不修改 YANG 模型，仅在生成的 Go 代码层面修复

package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run fix_enum.go <file>")
		os.Exit(1)
	}

	filename := os.Args[1]
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	text := string(content)

	// 修复标识符中的 | 字符，替换为 _OR_
	// 例如: HuaweiIfm_PortType_50|100GE -> HuaweiIfm_PortType_50_OR_100GE
	re := regexp.MustCompile(`(Huawei\w+)_(\d+)\|(\d+\w+)`)
	text = re.ReplaceAllString(text, "${1}_${2}_OR_${3}")

	// 修复映射值中的 | 字符（枚举值映射）
	// 这部分保持 YANG 原值不变，只修改变量名
	re2 := regexp.MustCompile(`(Huawei\w+)_FlexE_(\d+)\|(\d+\w)`)
	text = re2.ReplaceAllString(text, "${1}_FlexE_${2}_OR_${3}")

	err = os.WriteFile(filename, []byte(text), 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Fixed enum identifiers successfully")
}

// scanLines 读取文件所有行
func scanLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines 写入文件
func writeLines(filename string, lines []string) error {
	return os.WriteFile(filename, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}
