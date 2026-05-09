package main

import (
	"fmt"
	"os"
	"strings"

	"dungeon/engine"
)

func main() {
	cfg, err := engine.LoadConfig("config.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot load config.json:", err)
		os.Exit(1)
	}

	// read stdin fully
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}

	events := strings.ReplaceAll(sb.String(), "\r\n", "\n")

	out, err := engine.Process(cfg, events)
	if err != nil {
		fmt.Fprintln(os.Stderr, "processing error:", err)
		os.Exit(1)
	}
	fmt.Print(out)
}
