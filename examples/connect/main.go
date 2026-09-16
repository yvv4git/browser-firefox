package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/yvv4git/browser-firefox/examples/internal/bidi"
)

func main() {
	addr := flag.String("addr", "http://localhost:9222", "WebDriver BiDi endpoint")
	flag.Parse()

	ctx := context.Background()
	c, err := bidi.Connect(ctx, *addr)
	if err != nil {
		log.Fatalf("Connect: %v", err)
	}
	defer c.Close(context.Background())

	status, err := c.Status(ctx)
	if err != nil {
		log.Fatalf("Status: %v", err)
	}
	fmt.Printf("status: ready=%v message=%s\n", status.Ready, status.Message)

	contexts, err := c.Contexts(ctx)
	if err != nil {
		log.Fatalf("Contexts: %v", err)
	}
	fmt.Printf("contexts: %d\n", len(contexts))
	for i, page := range contexts {
		fmt.Printf("  [%d] %s\n", i+1, page.URL)
	}
}
