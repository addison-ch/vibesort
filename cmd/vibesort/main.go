package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"vibesort"
)

func main() {
	vibe := flag.String("vibe", "main character energy", "vibe prompt used for sorting")
	model := flag.String("model", "", "optional OpenAI model override")
	flag.Parse()

	items := flag.Args()
	if len(items) == 0 {
		log.Fatal("pass items to sort, e.g. vibesort -vibe \"spiciest\" ramen pho udon")
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	opts := []vibesort.Option{}
	if *model != "" {
		opts = append(opts, vibesort.WithModel(*model))
	}

	client, err := vibesort.NewClient(apiKey, opts...)
	if err != nil {
		log.Fatal(err)
	}

	sorted, err := client.SortStrings(context.Background(), items, *vibe)
	if err != nil {
		log.Fatal(err)
	}

	for i, item := range sorted {
		fmt.Printf("%d. %s\n", i+1, item)
	}
}
