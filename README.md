# vibesort

Vibe based sorting library

> ⚠️ This is a joke. Do not put it anywhere near production. It is non-deterministic,
> costs money per sort, and can be confidently wrong. That's the bit.

## Install

```sh
go get vibesort
```

## Usage

```go
package main

import (
	"fmt"

	"vibesort"
)

func main() {
	fruits := []string{"banana", "Apple", "cherry", "blueberry"}

	// Reads the API key from the OPENAI_API_KEY environment variable.
	sorted, err := vibesort.Sort(fruits, "alphabetically, ignoring case")
	if err != nil {
		panic(err)
	}
	fmt.Println(sorted) // [Apple banana blueberry cherry]
}
```

The descriptor is just a key — anything the model can reason about works:

```go
movies := []string{"The Matrix", "Barbie", "Jaws", "Oppenheimer"}
vibesort.Sort(movies, "by release year, oldest first")

peppers := []string{"jalapeño", "ghost pepper", "bell pepper", "habanero"}
vibesort.Sort(peppers, "mildest to spiciest")
```

It works on any type, since items are serialized to JSON before being sent:

```go
type Person struct {
	Name string
	Age  int
}
people := []Person{{"Alice", 30}, {"Bob", 25}}
vibesort.Sort(people, "youngest first")
```

## Configuration

The API key can come from the environment or be passed explicitly, and the
model can be overridden:

```go
sorted, err := vibesort.Sort(items, "by vibe",
	vibesort.WithAPIKey("sk-..."),
	vibesort.WithModel("gpt-4o"),
)
```

For full control over the client, context, and timeouts:

```go
client := vibesort.New(vibesort.WithModel("gpt-4o"))
sorted, err := vibesort.SortContext(ctx, client, items, "by vibe")
```

## How it works

1. Your items are serialized to a 0-indexed JSON array.
2. The array and your descriptor are sent to the model, which is asked to
   return the indices in sorted order.
3. We validate the returned permutation and reorder the original items — so the
   items you get back are exactly the ones you put in, just rearranged.

The original slice is never mutated.
