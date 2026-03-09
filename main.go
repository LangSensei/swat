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
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("swat %s\n", version)
		os.Exit(0)
	}

	mcpOnly := len(os.Args) > 1 && os.Args[1] == "--mcp-only"

	cmdr := commander.New("~/.swat")
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
