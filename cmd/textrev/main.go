// textrev 根据 JSON 修订计划批量修订来源目录中的 UTF-8 文本，
// 将结果写入一个全新的输出目录，并向标准输出打印 JSON 摘要。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"example.com/text-revision-plan/internal/revplan"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("textrev", flag.ContinueOnError)
	planPath := fs.String("plan", "", "JSON 修订计划路径（必填）")
	srcDir := fs.String("src", "", "来源目录（必填，只读不修改）")
	outDir := fs.String("out", "", "全新输出目录（必须尚不存在；check 模式可省略）")
	check := fs.Bool("check", false, "只校验计划，不创建输出目录")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "用法: textrev -plan plan.json -src 来源目录 -out 输出目录 [-check]\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *planPath == "" || *srcDir == "" || (!*check && *outDir == "") {
		fs.Usage()
		return 2
	}

	plan, err := revplan.LoadFile(*planPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		return 1
	}

	var summary *revplan.Summary
	if *check {
		summary, err = plan.ValidateOnly(*srcDir)
	} else {
		summary, err = plan.Execute(*srcDir, *outDir)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "修订计划校验/执行失败:")
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(summary); err != nil {
		fmt.Fprintln(os.Stderr, "错误: 输出摘要失败:", err)
		return 1
	}
	return 0
}
