# wnram — An In-Memory Go Library for WordNet

A Go library for accessing [Open English WordNet][https://en-word.net] (and Princeton WordNet compatible datasets).

## Implementation Overview

This library is a native Go parser for WordNet that stores the entire database in RAM.
This approach gives faster lookup times; the dataset sits at roughly 80–90 MB of RAM,
which is acceptable for most server environments. Parsing the full data files takes
around two seconds on a modest laptop.

## Supported Features

- Lookup by term, with automatic morphological normalization (plurals, verb forms, comparatives)
- Synonyms
- All relation types (Antonym, Hyponym, Hypernym, Attribute, Entailment, etc.)
- Iteration over the full database, optionally filtered by part of speech
- Lemmatization — find the canonical form of a word
- Morphology — derive a lemma from inflected input text

## Example Usage

```go
import (
    "log"

    "github.com/coreruleset/wnram"
)

func main() {
    wn, err := wnram.New("./path/to/wordnet/data")
    if err != nil {
        log.Fatalf("failed to load wordnet: %s", err)
    }

    // Look up "yummy" restricted to adjectives
    found, err := wn.Lookup(wnram.Criteria{
        Matching: "yummy",
        POS:      wnram.PartOfSpeechList{wnram.Adjective},
    })
    if err != nil {
        log.Fatalf("%s", err)
    }

    // Dump details about each matching synset to console
    for _, f := range found {
        f.Dump()
    }
}
```

### Iterating the database

```go
err := wn.Iterate(wnram.PartOfSpeechList{wnram.Noun}, func(l wnram.Lookup) error {
    fmt.Println(l.Word(), "—", l.Gloss())
    return nil
})
```

### Querying relations

```go
found, _ := wn.Lookup(wnram.Criteria{Matching: "good", POS: wnram.PartOfSpeechList{wnram.Adjective}})
for _, f := range found {
    for _, a := range f.Related(wnram.Antonym) {
        fmt.Println("antonym:", a.Word())
    }
}
```

## Developer Guide

### Prerequisites

- Go 1.26 or later

### Getting the WordNet data files

The library ships with [Open English WordNet][https://github.com/globalwordnet/english-wordnet] data in the `data/` folder.
If you need to update or replace the dataset, place the standard WordNet `data.*`,
`*.exc`, and index files in `data/` and point `wnram.New()` at it.

### Running the tests

```sh
go test ./...
```

With coverage:

```sh
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out   # open coverage report in browser
```

### Linting

The project uses [golangci-lint](https://golangci-lint.run):

```sh
golangci-lint run
```
