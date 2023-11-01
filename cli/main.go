package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"spell"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/hvlck/txt"
)

type Dictionary struct {
	trie *txt.Node
}

func main() {
	s := time.Now()

	b, err := os.ReadFile("../data/final.txt")
	if err != nil {
		panic(err)
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
	d := Dictionary{trie: t}

	fmt.Printf("loaded dictionary in %vms\n", time.Since(s).Milliseconds())
	scn := bufio.NewScanner(os.Stdin)

	for {
		io.WriteString(os.Stdout, "spell>> ")
		scanned := scn.Scan()
		if !scanned {
			return
		}

		ln := scn.Text()

		start := time.Now()
		results := spell.PartialMatch(d.trie, ln, 10, 10)
		end := time.Since(start).Milliseconds()

		table := tabby.New()
		table.AddHeader("Rank", "Correction", "Weight", "Levenshtein Distance", "Insertions/Deletions", "Substitutions", "Transpositions", "Frequency", "Matching Characters", "Prefix", "Suffix", "Keyboard Distance")
		for idx, res := range results {
			metrics := res.Metrics()
			table.AddLine(fmt.Sprintf("%v", idx+1), res.Word, res.Weight, metrics["levenshtein"], metrics["ins/del"], metrics["subs"], metrics["transpositions"], metrics["frequency"], "", metrics["prefix-length"], metrics["suffix-length"], "", metrics["keyboard-length"])
		}

		table.Print()
		fmt.Printf("\nresults generated in %vms\n", end)
	}
}
