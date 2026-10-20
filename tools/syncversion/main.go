// Command syncversion 把 -version 指定的版本号写入项目根 mygo.json 的
// version 字段，保持文件的其余内容与格式不变。
// 由 `task sync:version` 调用（见 Taskfile.yml）。
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
)

func main() {
	version := flag.String("version", "", "version to write into mygo.json")
	flag.Parse()
	if *version == "" {
		fmt.Fprintln(os.Stderr, "syncversion: -version is required")
		os.Exit(2)
	}
	re := regexp.MustCompile(`("version"\s*:\s*")[^"]*(")`)
	data, err := os.ReadFile("mygo.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "syncversion: %v\n", err)
		os.Exit(1)
	}
	if !re.Match(data) {
		fmt.Fprintln(os.Stderr, "syncversion: no version field in mygo.json")
		os.Exit(1)
	}
	data = re.ReplaceAll(data, []byte("${1}"+*version+"${2}"))
	if err := os.WriteFile("mygo.json", data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "syncversion: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("mygo.json version -> %s\n", *version)
}
