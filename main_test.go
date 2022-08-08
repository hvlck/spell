package spell

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	txt "github.com/hvlck/txt"
)

type LdResult struct {
	one, two string
	dist     uint8
}

func Testld(t *testing.T) {
	results := []LdResult{
		{
			one:  "burn",
			two:  "bayou",
			dist: 4,
		},
		{
			one:  "avid",
			two:  "antidisestablishmentarianism",
			dist: 25,
		},
	}

	for _, v := range results {
		l := ld(v.one, v.two)
		if l != v.dist {
			t.Fatalf(v.one, v.two, v.dist)
		}
	}
}

// rt and ro() https://rosettacode.org/wiki/Levenshtein_distance#Go
func rt(s, t string) int {
	d := make([][]int, len(s)+1)
	for i := range d {
		d[i] = make([]int, len(t)+1)
	}
	for i := range d {
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for j := 1; j <= len(t); j++ {
		for i := 1; i <= len(s); i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				min := d[i-1][j]
				if d[i][j-1] < min {
					min = d[i][j-1]
				}
				if d[i-1][j-1] < min {
					min = d[i-1][j-1]
				}
				d[i][j] = min + 1
			}
		}

	}
	return d[len(s)][len(t)]
}

func ro(s, t string) int {
	if s == "" {
		return len(t)
	}
	if t == "" {
		return len(s)
	}
	if s[0] == t[0] {
		return ro(s[1:], t[1:])
	}
	a := ro(s[1:], t[1:])
	b := ro(s, t[1:])
	c := ro(s[1:], t)
	if a > b {
		a = b
	}
	if a > c {
		a = c
	}
	return a + 1
}

const one = "avid"
const two = "antidisestablishmentarianism"

// system specs:
// M1 8-core Macbook Air, 16GB RAM
// benchmark ran on July 9 2022
// latest commit was https://github.com/hvlck/txt/commit/66a24f9329b2d6bd8c9ff496314cc2bcca9aed49

// go test -bench=. -benchmem -benchtime=1x
// BenchmarkLd/rosetta_code_loop-8         	       1	      3333 ns/op	    1328 B/op	       6 allocs/op
// BenchmarkLd/rosetta_code_slice-8        	       1	     31292 ns/op	       0 B/op	       0 allocs/op
// BenchmarkLd/txt-8                       	       1	      1000 ns/op	      16 B/op	       1 allocs/op
// BenchmarkSpellcheck-8                   	       1	  10183583 ns/op	  339536 B/op	   84157 allocs/op

// go test -bench=. -benchmem -benchtime=100x
// BenchmarkLd/rosetta_code_loop-8         	     100	        89.16 ns/op	      13 B/op	       0 allocs/op
// BenchmarkLd/rosetta_code_slice-8        	     100	       317.9 ns/op	       0 B/op	       0 allocs/op
// BenchmarkLd/txt-8                       	     100	         8.330 ns/op	       0 B/op	       0 allocs/op
// BenchmarkSpellcheck-8                   	     100	    100943 ns/op	    3393 B/op	     841 allocs/op

// go test -bench=. -benchmem -benchtime=1000000x
// BenchmarkLd/rosetta_code_loop-8         	 1000000	         0.008833 ns/op	       0 B/op	       0 allocs/op
// BenchmarkLd/rosetta_code_slice-8        	 1000000	         0.03225 ns/op	       0 B/op	       0 allocs/op
// BenchmarkLd/txt-8                       	 1000000	         0.0007500 ns/op	       0 B/op	       0 allocs/op
// BenchmarkSpellcheck-8                   	 1000000	        10.12 ns/op	       0 B/op	       0 allocs/op
func Benchmarkld(b *testing.B) {
	b.SetParallelism(1)
	b.Run("rosetta code loop", func(b *testing.B) {
		rt(one, two)
		b.StopTimer()
	})

	b.Run("rosetta code slice", func(b *testing.B) {
		ro(one, two)
		b.StopTimer()
	})

	b.Run("txt", func(b *testing.B) {
		ld(one, two)
		b.StopTimer()
	})
}

func TestWeigh(t *testing.T) {
	c := Correction{
		Word: "typo",
		ld:   ld("typo", "testing"),
	}

	c.weigh("testing")
}
func TestPrefixLength(t *testing.T) {
	vals := []uint8{
		PrefixLength("tree", "trees"),
		PrefixLength("grant", "grace"),
		PrefixLength("hammer", "hankering"),
	}
	answers := []uint8{4, 3, 2}

	for i, v := range vals {
		if v != answers[i] {
			t.Fatal(v, answers[i], i)
		}
	}
}

func TestMax(t *testing.T) {

}

func TestAbs(t *testing.T) {
	nth := abs(10 - 23)
	th := abs(10 + 3)

	if nth != 13 || th != 13 {
		t.Fail()
	}
}

func TestKeyProximity(t *testing.T) {
	vals := []uint8{
		KeyProximity('r', 't'),
		KeyProximity('s', 'w'),
		KeyProximity('a', 'w'),
		KeyProximity('l', 'p'),
		KeyProximity('v', 'p'),
		KeyProximity('1', '.'),
		KeyProximity('b', 'w'),
	}
	answers := []uint8{1, 1, 1, 1, 6, 7, 5}

	for i, v := range vals {
		if v != answers[i] {
			t.Fatalf("expected %v, got %v", answers[i], v)
		}
	}
}

func BenchmarkKeyProximity(b *testing.B) {
	vals := []uint8{
		KeyProximity('r', 't'),
		KeyProximity('s', 'w'),
		KeyProximity('a', 'w'),
		KeyProximity('l', 'p'),
		KeyProximity('v', 'p'),
		KeyProximity('1', '.'),
	}
	answers := []uint8{1, 1, 1, 1, 6, 7}

	for i, v := range vals {
		if v != answers[i] {
			b.Fatalf("expected %v, got %v", answers[i], v)
		}
	}
}

func TestSearch_Lev(t *testing.T) {
}

func TestPartialMatch(t *testing.T) {
	if dErr != nil {
		t.Fatal(dErr)
	}

	matches := PartialMatch(d.trie, "tesk", 3, 15)
	if len(matches) != 15 {
		t.Fail()
	}

	// adopted from Peter Norvig's spelling correcter test suite
	// http://norvig.com/spell-correct.html
	results := map[string]string{
		"speling":    "spelling",
		"korrectud":  "corrected",
		"bycycle":    "bicycle",
		"inconvient": "inconvenient",
		"arrainged":  "arranged",
		"peotry":     "poetry",
		"peotryy":    "poetry",
		"word":       "word",
	}

	for i, v := range results {
		r := PartialMatch(d.trie, i, 2, 10)
		if r != nil && len(r) > 0 {
			if r[len(r)-1].Word != v {
				t.Fatalf("expected %v, got %v (ld: %v)", v, r[len(r)-1], ld(v, i))
				for _, tt := range r {
					if tt.Word == v {
						fmt.Println(tt)
					}
				}
			}
		}
	}
}

func BenchmarkPartialMatch(b *testing.B) {
	b.SetParallelism(1)
	b.StopTimer()
	if dErr != nil {
		b.Fatal(dErr)
	}
	b.StartTimer()

	matches := PartialMatch(d.trie, "tesk", 3, 15)
	if len(matches) != 15 {
		b.Fail()
	}
}

func TestSpellcheck(t *testing.T) {
	results, err := Correct("wat", 3)
	fmt.Println(results)
	if err != nil {
		t.Fail()
	}
}

var d, dErr = loadTrie()

type Dictionary struct {
	trie *txt.Node
}

func loadTrie() (Dictionary, error) {
	b, err := ioutil.ReadFile("./data/final.txt")
	if err != nil {
		return Dictionary{}, err
	}
	t := txt.NewTrie()

	lines := bytes.Split(b, []byte("\n"))
	for _, v := range lines {
		if len(v) > 0 {
			r := bytes.Split(v, []byte(","))
			w := string(r[0])
			t.Insert(w, r[1])
			if err != nil {
				panic(err)
			}
		}
	}
	return Dictionary{trie: t}, nil
}

func BenchmarkTrieSpellcheck(b *testing.B) {
	b.SetParallelism(1)
	if dErr != nil {
		b.Fatal(dErr)
	}

	f := PartialMatch(d.trie, "wat", 5, 15)

	if len(f) != 15 {
		b.Fail()
	}
}

func BenchmarkSpellcheck(b *testing.B) {
	b.SetParallelism(1)

	_, err := Correct("wat", 3)
	b.StopTimer()
	if err != nil {
		b.Fail()
	}
}
