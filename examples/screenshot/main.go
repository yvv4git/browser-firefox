package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/yvv4git/browser-firefox/examples/internal/bidi"
)

func main() {
	addr := flag.String("addr", "http://localhost:9222", "WebDriver BiDi endpoint")
	url := flag.String("url", "https://www.wikipedia.org", "page to load")
	output := flag.String("output", "screenshot.png", "output file")
	full := flag.Bool("full", false, "capture full page instead of viewport")
	flag.Parse()

	ctx := context.Background()
	c, err := bidi.Connect(ctx, *addr)
	if err != nil {
		log.Fatalf("Connect: %v", err)
	}
	defer c.Close(context.Background())

	contextID, err := c.NewPage(ctx)
	if err != nil {
		log.Fatalf("NewPage: %v", err)
	}
	defer c.ClosePage(context.Background(), contextID)

	if err := c.Navigate(ctx, contextID, *url); err != nil {
		log.Fatalf("Navigate: %v", err)
	}

	png, err := c.Screenshot(ctx, contextID, *full)
	if err != nil {
		log.Fatalf("Screenshot: %v", err)
	}
	if err := os.WriteFile(*output, png, 0o644); err != nil {
		log.Fatalf("WriteFile: %v", err)
	}
	log.Printf("saved %s (%d bytes)", *output, len(png))
}
