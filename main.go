package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/LangSensei/swat/commander"
	"github.com/LangSensei/swat/mcp"
)

var version = "dev"

func main() {
	// Parse CLI flags
	runtimeName := "copilot"
	notifyBackend := "desktop"
	mcpOnly := false

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--version":
			fmt.Printf("swat %s\n", version)
			os.Exit(0)
		case "--mcp-only":
			mcpOnly = true
		case "--runtime":
			i++
			if i < len(os.Args) {
				runtimeName = os.Args[i]
			}
		case "--notify":
			i++
			if i < len(os.Args) {
				notifyBackend = os.Args[i]
			}
		}
	}

	cmdr := commander.New(runtimeName, notifyBackend)
	if !mcpOnly {
		go cmdr.BackgroundLoop(60 * time.Second)
		log.Println("SWAT Commander starting...")
	} else {
		log.Println("SWAT MCP server starting (no commander loop)...")
	}

	if err := mcp.Serve(cmdr, version); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
