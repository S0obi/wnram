package wnram

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

const PathToWordnetDataFiles = "./data"

func sourceCodeRelPath(suffix string) string {
	_, fileName, _, _ := runtime.Caller(1)
	return path.Join(path.Dir(fileName), suffix)
}

var wnInstance *Handle
var wnErr error

func init() {
	wnInstance, wnErr = New(sourceCodeRelPath(PathToWordnetDataFiles))
}

func TestParsing(t *testing.T) {
	if wnErr != nil {
		t.Fatalf("Can't initialize: %s", wnErr)
	}
}

func TestBasicLookup(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "good"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	gotAdjective := false
	for _, f := range found {
		if f.POS() == Adjective {
			gotAdjective = true
			break
		}
	}

	if !gotAdjective {
		t.Errorf("couldn't find basic adjective form for good")
	}
}

func TestPluralExceptionLookup(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "wolves"})
	if err != nil {
		t.Fatalf("%s", err)
	}

	gotNoun := false
	for _, f := range found {
		if f.POS() == Noun {
			gotNoun = true
			break
		}
	}

	if !gotNoun {
		t.Errorf("couldn't find exception plural noun form for wolves")
	}
}

func TestPluralLookup(t *testing.T) {
	tests := []struct {
		plural   string
		singular string
		pos      PartOfSpeech
	}{
		// Noun examples
		{"dogs", "dog", Noun},
		{"cars", "car", Noun},
		{"houses", "house", Noun},
		// Verb examples
		{"runs", "run", Verb},
		{"flies", "fly", Verb},
		{"plays", "play", Verb},
		// Adjective examples
		{"faster", "fast", Adjective},
		{"stronger", "strong", Adjective},
	}

	for _, tt := range tests {
		found, err := wnInstance.Lookup(Criteria{Matching: tt.plural})
		if err != nil {
			t.Errorf("Lookup(%q) failed: %v", tt.plural, err)
			continue
		}
		if len(found) == 0 {
			t.Errorf("couldn't find %v form for %q", tt.pos, tt.plural)
		}
	}
}

func TestLemma(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "awesome", POS: []PartOfSpeech{Adjective}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	if len(found) != 1 {
		for _, f := range found {
			f.Dump()
		}
		t.Fatalf("expected one synonym cluster for awesome, got %d", len(found))
	}

	if found[0].Lemma() != "amazing" {
		t.Errorf("incorrect lemma for awesome (%s)", found[0].Lemma())
	}
}

func setContains(haystack, needles []string) bool {
	for _, n := range needles {
		found := slices.Contains(haystack, n)
		if !found {
			return false
		}
	}
	return true
}

func TestSynonyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "yummy", POS: []PartOfSpeech{Adjective}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	if len(found) != 1 {
		for _, f := range found {
			f.Dump()
		}
		t.Fatalf("expected one synonym cluster for yummy, got %d", len(found))
	}

	syns := found[0].Synonyms()
	if !setContains(syns, []string{"delicious", "delectable"}) {
		t.Errorf("missing synonyms for yummy")
	}
}

func TestAntonyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "good", POS: []PartOfSpeech{Adjective}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	var antonyms []string
	for _, f := range found {
		as := f.Related(Antonym)
		for _, a := range as {
			antonyms = append(antonyms, a.Word())
		}
	}

	if !setContains(antonyms, []string{"bad", "evil"}) {
		t.Errorf("missing antonyms for good")
	}
}

func TestHypernyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "jab", POS: []PartOfSpeech{Noun}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	var hypernyms []string
	for _, f := range found {
		as := f.Related(Hypernym)
		for _, a := range as {
			hypernyms = append(hypernyms, a.Word())
		}
	}

	if !setContains(hypernyms, []string{"punch"}) {
		t.Errorf("missing hypernyms for jab (expected punch, got %v)", hypernyms)
	}
}

func TestHyponyms(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "food", POS: []PartOfSpeech{Noun}})
	if err != nil {
		t.Fatalf("%s", err)
	}

	var hyponyms []string
	for _, f := range found {
		as := f.Related(Hyponym)
		for _, a := range as {
			hyponyms = append(hyponyms, a.Word())
		}
	}

	expected := []string{"nutriment", "beverage", "foodstuff", "comestible"}
	if !setContains(hyponyms, expected) {
		t.Errorf("missing hyponyms for food (expected %v, got %v)", expected, hyponyms)
	}
}

func TestIterate(t *testing.T) {
	count := 0
	err := wnInstance.Iterate(PartOfSpeechList{Noun}, func(l Lookup) error {
		if l.POS() != Noun {
			t.Errorf("Iterate yielded non-noun entry: %s (%s)", l.Word(), l.POS())
		}
		count++
		return nil
	})

	if err != nil {
		t.Fatalf("Iterate failed: %v", err)
	}

	if count == 0 {
		t.Errorf("Iterate yielded no nouns")
	}
}

func TestWordbase(t *testing.T) {
	tests := []struct {
		word     string
		ender    int
		expected string
	}{
		// Noun suffixes
		{"dogs", 0, "dog"},
		{"buses", 1, "bus"},
		// Verb suffixes
		{"runs", 8, "run"},
		{"flies", 9, "fly"},
		// Adjective suffixes
		{"faster", 16, "fast"},  // "er" -> ""
		{"fastest", 17, "fast"}, // "est" -> ""
	}

	for _, tt := range tests {
		got := wordbase(tt.word, tt.ender)
		if got != tt.expected {
			t.Errorf("wordbase(%q, %d) = %q; want %q", tt.word, tt.ender, got, tt.expected)
		}
	}
}

func TestMorphword(t *testing.T) {
	tests := []struct {
		word     string
		pos      PartOfSpeech
		expected string
	}{
		// Noun cases
		{"dogs", Noun, "dog"},
		{"buses", Noun, "bus"},
		{"boxes", Noun, "box"},
		{"handful", Noun, "hand"},
		{"men", Noun, "man"},
		{"ladies", Noun, "lady"},
		{"fullness", Noun, ""}, // "ss" ending returns ""
		{"a", Noun, ""},        // too short returns ""
		// Verb cases
		{"runs", Verb, "run"},
		{"flies", Verb, "fly"},
		{"played", Verb, "play"},
		{"playing", Verb, "play"},
		// Adjective cases
		{"faster", Adjective, "fast"},
		{"fastest", Adjective, "fast"},
		{"stronger", Adjective, "strong"},
		{"strongest", Adjective, "strong"},
		// Adverb cases (should not change)
		{"quickly", Adverb, ""},
		{"slowly", Adverb, ""},
	}

	for _, tt := range tests {
		got := wnInstance.MorphWord(tt.word, tt.pos)
		if got != tt.expected {
			t.Errorf("morphword(%q, %v) = %q; want %q", tt.word, tt.pos, got, tt.expected)
		}
	}
}

func TestPartOfSpeechString(t *testing.T) {
	tests := []struct {
		pos  PartOfSpeech
		want string
	}{
		{Noun, "noun"},
		{Verb, "verb"},
		{Adjective, "adj"},
		{Adverb, "adv"},
		{PartOfSpeech(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.pos.String(); got != tt.want {
			t.Errorf("PartOfSpeech(%d).String() = %q, want %q", tt.pos, got, tt.want)
		}
	}
}

func TestLookupOutputMethods(t *testing.T) {
	found, err := wnInstance.Lookup(Criteria{Matching: "good", POS: PartOfSpeechList{Adjective}})
	if err != nil || len(found) == 0 {
		t.Fatal("failed to look up 'good'")
	}
	f := found[0]

	if s := f.String(); !strings.Contains(s, "good") {
		t.Errorf("String() = %q, want it to contain 'good'", s)
	}
	if f.Gloss() == "" {
		t.Error("Gloss() returned empty string")
	}
	if d := f.DumpStr(); !strings.Contains(d, "good") {
		t.Errorf("DumpStr() = %q, want it to contain 'good'", d)
	}
	f.Dump() // must not panic
}

func TestLookupEmptyString(t *testing.T) {
	if _, err := wnInstance.Lookup(Criteria{Matching: ""}); err == nil {
		t.Error("expected error for empty Matching string")
	}
}

func TestIterateCallbackError(t *testing.T) {
	sentinel := fmt.Errorf("stop iteration")
	err := wnInstance.Iterate(PartOfSpeechList{Noun}, func(Lookup) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestMorphWordNoMatch(t *testing.T) {
	for _, pos := range []PartOfSpeech{Verb, Adjective} {
		if got := wnInstance.MorphWord("zzzzz", pos); got != "" {
			t.Errorf("MorphWord(%q, %v) = %q, want empty string", "zzzzz", pos, got)
		}
	}
}

func TestNewSkipsHiddenAndBackupFiles(t *testing.T) {
	dir := t.TempDir()
	line := "00001234 00 a 01 able 0 000 | having the necessary means\n"
	if err := os.WriteFile(filepath.Join(dir, "data.test"), []byte(line), 0644); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".hidden", "backup~", "edit#"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("this must not be parsed\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := New(dir); err != nil {
		t.Errorf("New() should succeed and skip hidden/backup files, got: %v", err)
	}
}

func TestNewNonExistentDir(t *testing.T) {
	if _, err := New("/nonexistent/wordnet/path"); err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestNewBadDataFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "data.test"), []byte("00001234 00 X bad-pos\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir); err == nil {
		t.Error("expected error for invalid POS in data file")
	}
}

func TestNewBadExcFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "test.exc"), []byte("just-one-token\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir); err == nil {
		t.Error("expected error for malformed exception file")
	}
}

func TestNewOrphanCluster(t *testing.T) {
	dir := t.TempDir()
	// Semantic relation to offset 99999999 which is never defined → cluster with no words
	line := "00001234 00 a 01 able 0 001 ! 99999999 a 0000 | some gloss\n"
	if err := os.WriteFile(filepath.Join(dir, "data.test"), []byte(line), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir); err == nil {
		t.Error("expected consistency error for undefined cluster reference")
	}
}

func TestNewBogusSourceRelation(t *testing.T) {
	dir := t.TempDir()
	// Synset has 1 word; syntactic relation with source word index 2 (0-based 1) is out of bounds.
	// nature 0x0202 → source = uint8(2)-1 = 1, which is >= len(words) = 1.
	line := "00001234 00 a 01 able 0 001 ! 00001235 a 0202 | some gloss\n"
	if err := os.WriteFile(filepath.Join(dir, "data.test"), []byte(line), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(dir); err == nil {
		t.Error("expected error for out-of-bounds source word index")
	}
}

func TestReadLineNoTrailingNewline(t *testing.T) {
	var lines []string
	err := inPlaceReadLine(strings.NewReader("alpha\nbeta"), func(data []byte, _, _ int64) error {
		lines = append(lines, string(data))
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 || lines[1] != "beta" {
		t.Errorf("expected [alpha beta], got %v", lines)
	}
}

func TestReadLineCallbackError(t *testing.T) {
	sentinel := fmt.Errorf("stop")
	err := inPlaceReadLine(strings.NewReader("line1\nline2\n"), func(_ []byte, _ int64, _ int64) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestReadFromNonExistentFile(t *testing.T) {
	err := inPlaceReadLineFromPath("/nonexistent/file.dat", func(_ []byte, _ int64, _ int64) error {
		return nil
	})
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestLexNextEmpty(t *testing.T) {
	l := lexable("")
	if _, ok := l.next(); ok {
		t.Error("next() on empty lexable should return ok=false")
	}
}

func TestLexDecimalNumberErrors(t *testing.T) {
	l := lexable("")
	if _, err := l.lexDecimalNumber(); err == nil {
		t.Error("expected error calling lexDecimalNumber on empty input")
	}

	// value too large for int64 → ParseInt overflow
	l = lexable("99999999999999999999")
	if _, err := l.lexDecimalNumber(); err == nil {
		t.Error("expected ParseInt overflow error for large decimal number")
	}
}

func TestLexHexNumberErrors(t *testing.T) {
	l := lexable("")
	if _, err := l.lexHexNumber(); err == nil {
		t.Error("expected error calling lexHexNumber on empty input")
	}

	// 16 hex F's = 0xFFFFFFFFFFFFFFFF overflows int64
	l = lexable("FFFFFFFFFFFFFFFF")
	if _, err := l.lexHexNumber(); err == nil {
		t.Error("expected ParseInt overflow error for large hex number")
	}
}

func TestLexGlossErrors(t *testing.T) {
	l := lexable("")
	if _, err := l.lexGloss(); err == nil {
		t.Error("expected error for empty gloss input")
	}

	l = lexable("no-pipe-here")
	if _, err := l.lexGloss(); err == nil {
		t.Error("expected error when gloss doesn't start with '|'")
	}
}

func TestLexPOSErrors(t *testing.T) {
	l := lexable("")
	if _, err := l.lexPOS(); err == nil {
		t.Error("expected error for empty POS input")
	}

	l = lexable("X")
	if _, err := l.lexPOS(); err == nil {
		t.Error("expected error for invalid POS character 'X'")
	}
}

func TestLexRelationTypeUnknown(t *testing.T) {
	l := lexable("??")
	if _, err := l.lexRelationType(); err == nil {
		t.Error("expected error for unrecognized relation type symbol")
	}
}

func TestParseLineEmpty(t *testing.T) {
	p, err := parseLine([]byte{}, 1)
	if err != nil || p != nil {
		t.Errorf("empty line: want (nil, nil), got (%v, %v)", p, err)
	}
}

func TestParseLineComment(t *testing.T) {
	p, err := parseLine([]byte("  1 This is a comment"), 1)
	if err != nil || p != nil {
		t.Errorf("comment: want (nil, nil), got (%v, %v)", p, err)
	}
}

func TestParseLineErrors(t *testing.T) {
	tests := []struct {
		name string
		data string
		line int64
	}{
		{"comment line number mismatch", "  5 some text", 1},
		{"bad filenumber", "00001234 XX", 1},
		{"bad POS", "00001234 00 X", 1},
		{"bad wordcount", "00001234 00 a XX", 1},
		{"bad sense id", "00001234 00 a 01 word", 1},
		{"bad pointer count", "00001234 00 a 01 word 0 XX", 1},
		{"bad relation type", "00001234 00 a 01 word 0 01 ??", 1},
		{"bad ptr offset", "00001234 00 a 01 word 0 01 ! BADOFFSET a 0000", 1},
		{"bad ptr POS", "00001234 00 a 01 word 0 01 ! 00001235 X 0000", 1},
		{"bad ptr nature", "00001234 00 a 01 word 0 01 ! 00001235 a ZZZZ", 1},
		{"bad frame marker", "00001234 00 v 01 word 0 00 01 bad 01 00 | gloss", 1},
		{"bad frame number", "00001234 00 v 01 word 0 00 01 + bad 00 | gloss", 1},
		{"bad frame word number", "00001234 00 v 01 word 0 00 01 + 01 ZZ | gloss", 1},
		{"missing gloss", "00001234 00 a 01 word 0 00", 1},
		{"wrong gloss separator", "00001234 00 a 01 word 0 00 no-gloss", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseLine([]byte(tt.data), tt.line); err == nil {
				t.Errorf("parseLine(%q, %d) expected error, got nil", tt.data, tt.line)
			}
		})
	}
}
