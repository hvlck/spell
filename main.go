package spell

import (
	"bytes"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode"

	txt "github.com/hvlck/txt"
)

// go:embed ./data/words.txt
var dict_file []byte

// Loads the dictionary words list
func loadDict() [][]byte {
	return bytes.Split(dict_file, []byte("\n"))
}

var dict = loadDict()

// Generates a list of spelling corrections for the provided `word`.
// `lim` is the maximum levenshtein distance away for a correction to be returned (inclusive)
// e.g. a correction with a LD of 3 would be returned with a limit of `3`, but a word with a LD of 4 would not
// in the return values, the `uint8` in the map corresponds to levenshtein distance of the corrected word
// todo: better, more focused results; current implementation returns many options, especially for smaller words
// could possibly be fixed by weighting spelling errors that are closer on keyboards
// e.g. with the input `vad`
// `tad` and `bad` are both options, but the "b" in `bad` is closer physically on the keyboard than the "t" in
// `tab`, and so would be the better choice
func Correct(word string, lim uint8) (map[string]uint8, error) {
	if dictErr != nil {
		return nil, dictErr
	}

	// all found matches
	matches := map[string]uint8{}

	for i := 0; i < len(dict); i++ {
		// levenshtein distance of correction
		l := ld(word, string(dict[i]))
		if l <= lim {
			matches[string(dict[i])] = l
			lim = l
		}
	}

	return matches, nil
}

type Dict struct {
	*txt.Node
}

// A word correction. A copy of the original word is not stored.
type Correction struct {
	// Corrected word
	Word string
	// Levenshtein distance from original. Lower is closer.
	ld uint8
	// Number of characters that both words share at the beginning.
	// For example, grace and grant have a prefix_len of 3 as they both share `gra` at the beginning.
	// Higher is better.
	prefix_len uint8
	// Frequency of use of the word in an English text corpus
	frequency uint
	// Sum of the distance between each character in the original and corrected word. Lower is better.
	key_len uint8
	// Weight of word correction. Higher values mean the correction is closer to the original word.
	Weight float32
}

// Searches for all words in the trie within a fixed `limit` edit distance away from the original string `s`.
func search_lev(n *txt.Node, s, b string, limit uint8, prev ...Correction) []Correction {
	if n == nil {
		return make([]Correction, 0)
	}

	if n.Id == 0 {
		for rn, v := range n.Kids {
			prev = append(prev, search_lev(v, s, string(rn), limit)...)
		}
		return prev
	} else {
		for rn, v := range n.Kids {
			lev := ld(b, s)
			// if lev > limit {
			// 	continue
			// }

			if v.Done && len(v.Kids) == 0 {
				if lev <= limit {
					freq, err := strconv.Atoi(string(v.Data))
					if err != nil {
						freq = 0
					}
					prev = append(prev, Correction{ld: lev, Word: b, Weight: 0, frequency: uint(freq)})
				}

				continue
			} else {
				prev = append(prev, search_lev(v, s, b+string(rn), limit)...)
			}
		}
	}

	return prev
}

// PrefixLength calculates the number of same characters at the beginning of both strings.
func PrefixLength(o, t string) uint8 {
	var n uint8 = 0
	for i, v := range t {
		if len(o)-1 < i {
			return n
		}

		if v == rune(o[i]) {
			n++
		} else {
			break
		}
	}

	return n
}

var keys = [][]rune{
	{'`', '1', '2', '3', '4', '5', '6', '7', '8', '9', '0', '-', '='},
	{'q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', '[', ']', '\\'},
	{'a', 's', 'd', 'f', 'g', 'h', 'j', 'k', 'l', ';', '\'', ' ', ' '},
	{'z', 'x', 'c', 'v', 'b', 'n', 'm', ',', '.', '/', ' ', ' ', ' '},
}

// one-dimensional array of all keys
var all_keys = make([]rune, 0, 13*4)

// Returns the absolute value.
func abs(x int) int {
	y := 0
	if x < y {
		return y - x
	}
	return x - y
}

// Returns the max of the two numbers
func max(x, y int) uint8 {
	if x > y {
		return uint8(x)
	} else {
		return uint8(y)
	}
}

// Returns the number of keys away `t` is from `o`.
// This is used as a measure of accidental typos, e.g. `jat` when the intention was `hat`.
func KeyProximity(o, t rune) uint8 {
	if o == t {
		return 0
	}

	if len(all_keys) == 0 {
		for _, v := range keys {
			all_keys = append(all_keys, v...)
		}
	}

	// row
	rO := 0
	// column
	cO := 0

	rT := 0
	cT := 0
	for idx, v := range all_keys {
		idx += 1
		if v == o {
			cO = idx / 13
			rO = idx - cO*13
		}

		if v == t {
			cT = idx / 13
			rT = idx - cT*13
		}
	}
	if o == 'b' {
		fmt.Println(rO, cO, rT, cT)
	}

	rowDiff := abs(rT - rO)
	colDiff := abs(cT - cO)

	// largest value, no trig
	return max(colDiff, rowDiff)
}

const (
	KEYDIST_WEIGHT   = 200
	PREFIX_WEIGHT    = 5
	LEV_WEIGHT       = 0.0001
	FREQUENCY_WEIGHT = 100
)

// Weighs a given correction for the provided original string.
// todo: improvements to waiting algorithm, documentation
func (c *Correction) weigh(original string) {
	// todo: sometimes this returns true for multiple values, and occassionally doesn't work at all
	if c.Word == original {
		c.Weight = 1
		return
	}

	// sum of key lengths
	var key_len uint8 = 0
	for i, v := range c.Word {
		if len(original)-1 < i {
			break
		}

		key_len += KeyProximity(v, rune(original[i]))
	}

	if len(original) > len(c.Word) {
		key_len += uint8(len(original) - len(c.Word))
	} else if len(c.Word) > len(original) {
		key_len += uint8(len(c.Word) - len(original))
	}

	c.key_len = key_len
	c.prefix_len = PrefixLength(c.Word, original)

	var wld float32 = float32(c.ld) * LEV_WEIGHT
	var wkey_len float32 = float32(c.key_len) * KEYDIST_WEIGHT
	var wprefix_len float32 = PREFIX_WEIGHT / float32(c.prefix_len)
	if c.prefix_len == 0 {
		wprefix_len = 0
	}
	var wfrequency float32 = FREQUENCY_WEIGHT / float32(c.frequency)

	c.Weight = float32(10 / (wld + wkey_len + wprefix_len + wfrequency))
	if c.Weight > 1 {
		c.Weight = 0.999
	}
}

// Returns all matches in the given trie within `target` edit distances of `s`. Max is the maximum number of corrections
// to return.
// todo: -1 value for `max` to include all matches
func PartialMatch(n *txt.Node, s string, target uint8, max int) []Correction {
	f := search_lev(n, strings.ToLower(s), "", target)

	var lim float32 = 0
	res := make([]Correction, max)

	last := 0
	for _, v := range f {
		v.weigh(s)
		// first element
		if lim == 0 {
			lim = v.Weight
		}

		if v.Weight >= lim {
			// res is filled
			if n := res[last]; len(n.Word) != 0 && n.ld != 0 {
				// search for element with lowest weight, replace it
				for i, k := range res {
					// levenshtein and weight of word being examined is less than word currently in final results
					if v.Weight > k.Weight && v.ld <= target {
						// t := time.Now()
						res[i] = v
						sort.Slice(res, func(i, j int) bool {
							return res[i].Weight < res[j].Weight
						})
						// fmt.Println(time.Since(t))
						break
					}
				}
			} else if last < max {
				res[last] = v
				if last+1 < max {
					last++
				}
				sort.Slice(res, func(i, j int) bool {
					return res[i].Weight < res[j].Weight
				})
			}
		}
		// ignore words with weights smaller than limit
	}

	return res
}

// returns the minimum of a function
func min(v ...uint8) uint8 {
	m := v[0]

	for _, k := range v {
		if k < m {
			m = k
		}
	}

	return m
}

// levenshtein distance
// based in part on https://rosettacode.org/wiki/Levenshtein_distance#Go, some modifications made to use one-dimensional array
// this version usually takes about half the time as the second version, and usually less than half the time of the first version on RosettaCode
// todo: add swap variant (e.g. `liek` -> `like`)
func ld(a, b string) uint8 {
	if a == "" {
		return uint8(len(b))
	}
	if b == "" {
		return uint8(len(a))
	}
	if a == b {
		return 0
	}

	// row is the previous row in the LD table (contains top right at current index and top left at current index - 1)
	row := make([]uint8, len(a)+1)
	for i := range row {
		row[i] = uint8(i)
	}

	// first characters aren't the same
	var current uint8

	// bottom left, starts at 1
	var bl uint8
	// go through columns first
	for i := 1; i <= len(b); i++ {
		// previous top left - used for if letters are the same
		ptl := uint8(i - 1)
		// set first value of previous row equal to ptl
		row[0] = ptl
		current = 0

		// top left
		var tl uint8
		// top right
		var tr uint8
		bl = uint8(i)

		// go through each character in the row
		for j := 1; j <= len(a); j++ {
			// set top right equal to the value at
			tr = row[j]
			tl = ptl

			// in first row of array, so top values should be equal to index of item (e.g. [0 1 2 3 4 5])
			// value of top right should then be the value of the array at the index in the current loop
			if i == 1 {
				tr = uint8(j)
			}

			// characters are the same - use previous top left value
			if a[j-1] == b[i-1] {
				current = tl
			} else {
				// characters are different - take minimum of the three, add one operation
				current = min(tl, tr, bl) + 1
			}

			// set the previous top left value equal to
			ptl = row[j]
			row[j] = current
			bl = current
		}
	}

	return uint8(current)
}
