package main

import (
	"log"
	"time"

	"github.com/LangSensei/swat/commander"
	"github.com/LangSensei/swat/mcp"
)

func main() {
	cmdr := commander.New("~/.swat")
	go cmdr.BackgroundLoop(60 * time.Second)

	log.Println("SWAT Commander starting...")
	if err := mcp.Serve(cmdr); err != nil {
		log.Fatalf("MCP server error: %v", err)
	}
}
