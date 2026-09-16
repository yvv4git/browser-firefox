package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/yvv4git/browser-firefox/examples/internal/bidi"
)

func main() {
	addr := flag.String("addr", "http://localhost:9222", "WebDriver BiDi endpoint")
	output := flag.String("output", "check.png", "screenshot output file")
	flag.Parse()
	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(2)
	}
	url := flag.Arg(0)

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

	if err := c.Navigate(ctx, contextID, url); err != nil {
		log.Fatalf("Navigate: %v", err)
	}

	title, err := c.Evaluate(ctx, contextID, "document.title")
	if err != nil {
		log.Fatalf("Evaluate title: %v", err)
	}
	fmt.Printf("title: %v\n", title.Value)

	pageURL, err := c.Evaluate(ctx, contextID, "document.URL")
	if err != nil {
		log.Fatalf("Evaluate url: %v", err)
	}
	fmt.Printf("url: %v\n", pageURL.Value)

	htmlLen, err := c.Evaluate(ctx, contextID, "document.documentElement.outerHTML.length")
	if err != nil {
		log.Fatalf("Evaluate html size: %v", err)
	}
	fmt.Printf("html: %v bytes\n", htmlLen.Value)

	png, err := c.Screenshot(ctx, contextID, false)
	if err != nil {
		log.Fatalf("Screenshot: %v", err)
	}
	if err := os.WriteFile(*output, png, 0o644); err != nil {
		log.Fatalf("WriteFile: %v", err)
	}
	fmt.Printf("screenshot saved: %s\n", *output)
}
