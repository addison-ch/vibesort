# vibesort

A very serious Go sorting library that sorts by **vibe**, not outdated comparisons mechanisms

It sends your list to OpenAI, asks for a vibe ranking, and returns the reordered slice.

## Install

```bash
go get vibesort
```

## Library Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"vibesort"
)

func main() {
	client, err := vibesort.NewClient(os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	items := []string{"golang", "rust", "python", "javascript"}
	sorted, err := client.SortStrings(context.Background(), items, "most likely to start a startup this weekend")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(sorted)
}
```

## CLI Usage

```bash
go run ./cmd/vibesort -vibe "would win in a dance battle" compiler linker debugger profiler
```

Optional model override:

```bash
go run ./cmd/vibesort -model gpt-4.1 -vibe "chaotic neutral energy" alpha beta gamma
```

## Notes

- Default endpoint is OpenAI Chat Completions (`/v1/chat/completions`), but you can override with `WithBaseURL`.
