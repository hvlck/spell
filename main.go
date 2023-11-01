package spell

import (
	"bytes"
	"math"
	"sort"
	"strconv"
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
func Correct(word string, lim float64) map[string]float64 {
	// all found matches
	matches := map[string]float64{}

	for i := 0; i < len(dict); i++ {
		// levenshtein distance of correction
		l := levenshtein(word, string(dict[i]))
		if l <= lim {
			matches[string(dict[i])] = l
			lim = l
		}
	}

	return matches
}

type Dict struct {
	*txt.Node
}

// A word correction. A copy of the original word is not stored.
type Correction struct {
	// Corrected word
	Word string
	// Levenshtein distance from original. Lower is closer.
	ld [4]float64
	// Number of characters that both words share at the beginning.
	// For example, grace and grant have a prefix_len of 3 as they both share `gra` at the beginning.
	// Higher is better.
	prefix_len uint8
	suffix_len uint8
	// Frequency of use of the word in an English text corpus
	frequency float64
	// Sum of the distance between each character in the original and corrected word. Lower is better.
	key_len uint8
	// Weight of word correction. Higher values mean the correction is closer to the original word.
	Weight float64
}

func (c *Correction) Metrics() map[string]float64 {
	return map[string]float64{
		"levenshtein":     c.ld[0],
		"ins/del":         c.ld[1],
		"subs":            c.ld[2],
		"transpositions":  c.ld[3],
		"frequency":       c.frequency,
		"prefix-length":   float64(c.prefix_len),
		"suffix-length":   float64(c.suffix_len),
		"keyboard-length": float64(c.key_len),
	}
}

// Searches for all words in the trie within a fixed `limit` edit distance away from the original string `s`.
func search_lev(n *txt.Node, s, b string, limit float64, prev ...Correction) []Correction {
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
			lev := levenshtein_with_operations(b, s)

			if v.Done && len(v.Kids) == 0 {
				if lev[0] <= limit {
					freq, err := strconv.ParseFloat(string(v.Data), 64)
					if err != nil {
						freq = 0
					}
					prev = append(prev, Correction{ld: lev, Word: b, Weight: 0, frequency: freq})
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
func abs[T int | int8 | uint8](x T) T {
	var y T = 0
	if x < y {
		return y - x
	}
	return x - y
}

// Returns the max of the two numbers
func max[T int8 | uint8 | int | float64](numbers ...T) T {
	var highest T = 0
	for _, num := range numbers {
		if num > highest {
			highest = num
		}
	}

	return highest
}

// Returns the number of keys away `t` is from `o`.
// This is used as a measure of accidental typos, e.g. `jat` when the intention was `hat`.
// Case is also handled; if the two cases differ, the final score is incremented by 1.
func KeyProximity(original, target rune) uint8 {
	if original == target {
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

	// target row/col
	rT := 0
	cT := 0

	for idx, v := range all_keys {
		idx += 1
		if v == unicode.ToLower(original) {
			cO = idx / 13
			rO = idx - cO*13
		}

		if v == unicode.ToLower(target) {
			cT = idx / 13
			rT = idx - cT*13
		}
	}

	rowDiff := abs(rT - rO)
	colDiff := abs(cT - cO)

	var key_case uint8 = 0
	original_is_lower := unicode.ToLower(original) == original
	target_is_lower := unicode.ToLower(target) == target
	if original_is_lower != target_is_lower {
		key_case = 1
	}

	// largest value, no trig
	return uint8(max(colDiff, rowDiff)) + key_case
}

const (
	LEV_WEIGHT       = 1e-5
	LEV_INDEL_WEIGHT = 1.0
	LEV_SUB_WEIGHT   = 10.0
	LEV_SWAP_WEIGHT  = .1

	KEYDIST_WEIGHT   = 20
	PREFIX_WEIGHT    = 1
	SUFFIX_WEIGHT    = PREFIX_WEIGHT
	FREQUENCY_WEIGHT = 10
	MATCHES_WEIGHT   = 1
)

var lev_weights = map[int]float64{
	0: LEV_WEIGHT,
	1: LEV_SUB_WEIGHT,
	2: LEV_INDEL_WEIGHT,
	3: LEV_SWAP_WEIGHT,
}

// calculates the number of characters two strings share
// characters match if the character and index of the character are the same in both strings
// matching characters do not have to be continuous; e.g. the words
// test        tertiary
// have 3 shared characters (t, e, and t again)
func SharedCharacters(original, target string) float64 {
	matches := 0.0

	length := min(len(original), len(target))

	for i := 0; i < length; i++ {
		if original[i] == target[i] {
			matches += 1
		}
	}

	return matches
}

// reverses a string
func reverse(s string) string {
	res := ""
	for _, v := range s {
		res = string(v) + res
	}
	return res
}

// Weighs a given correction for the provided original string.
// todo: improvements to waiting algorithm, documentation
func (c *Correction) weigh(original string) {
	// todo: sometimes this returns true for multiple values, and occassionally doesn't work at all
	if c.Word == original {
		c.Weight = math.Inf(1)
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

	magic_weight := 0.0
	c.key_len = key_len
	c.prefix_len = PrefixLength(c.Word, original)

	c.suffix_len = PrefixLength(reverse(c.Word), reverse(original))

	var wld_div float64 = 1
	for i := 0; i < len(c.ld); i++ {
		w := c.ld[i] * lev_weights[i]
		if w != 0 {
			wld_div *= w
		}
	}
	var wld float64 = 1 / wld_div

	if c.ld[0] == 0 {
		magic_weight += math.Inf(1)
	}

	var wkey_len float64 = KEYDIST_WEIGHT / (float64(c.key_len))
	var wprefix_len float64 = PREFIX_WEIGHT * float64(c.prefix_len)
	var wsuffix_len float64 = SUFFIX_WEIGHT * float64(c.suffix_len)

	if wprefix_len == wsuffix_len {
		magic_weight += 25
	}

	var wfrequency float64 = FREQUENCY_WEIGHT * c.frequency
	var wmatches float64 = MATCHES_WEIGHT * SharedCharacters(original, c.Word)

	c.Weight = wld + wkey_len + wprefix_len + wfrequency + wmatches + wsuffix_len + magic_weight
}

// Returns all matches in the given trie within `target` edit distances of `s`. Max is the maximum number of corrections
// to return. Exact matches will have a weight of +Inf.
// todo: -1 value for `max` to include all matches
func PartialMatch(n *txt.Node, s string, target float64, max int) []Correction {
	f := search_lev(n, s, "", target)

	var lim float64 = 0
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
			if n := res[last]; len(n.Word) != 0 && n.ld[0] != 0 {
				// search for element with lowest weight, replace it
				for i, k := range res {
					// levenshtein and weight of word being examined is less than word currently in final results
					if v.Weight > k.Weight && v.ld[0] <= target {
						res[i] = v
						sort.Slice(res, func(i, j int) bool {
							return res[i].Weight < res[j].Weight
						})
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

// returns the minimum of a set of numbers
func min[T int | uint8 | float64](v ...T) T {
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
func levenshtein(a, b string) float64 {
	if a == "" {
		return float64(len(b))
	}

	if b == "" {
		return float64(len(a))
	}

	if a == b {
		return 0
	}

	// row is the previous row in the LD table (contains top right at current index and top left at current index - 1)
	prev_row := make([]uint8, len(a)+1)
	for i := range prev_row {
		prev_row[i] = uint8(i)
	}

	result := 0.0
	// first characters aren't the same
	var current uint8

	// bottom left, starts at 1
	var bl uint8

	// go through columns first
	for i := 1; i <= len(b); i++ {
		// previous top left - used if letters are the same

		// set first value of previous row equal to ptl
		prev_row[0] = uint8(i)
		current = 0

		// top left
		var tl uint8
		// top right
		var tr uint8
		// bottom left
		bl = uint8(i)

		// go through each character in the row
		for j := 1; j <= len(a); j++ {
			// set top right equal to the value at
			tr = prev_row[j]
			tl = prev_row[j-1]

			// in first row of array, so top values should be equal to index of item (e.g. [0 1 2 3 4 5])
			// value of top right should then be the value of the array at the index in the current loop
			if i == 1 {
				tr = uint8(j)
			}

			// characters are the same - use previous top left value
			if a[j-1] == b[i-1] {
				current = tl
			} else {
				current = min(tl, tr, bl) + 1

				// todo: verify this works correctly - hard to reason about
				if (j < len(a) && i < len(b)) && (j+1 < len(a) && i+1 < len(b)) {
					// transpositions
					// bounds check for transposition indexing
					if a[j-1] == b[i] && a[j] == b[i-1] {
						current = tl
					}
				}
			}

			// set the previous top left value equal to
			prev_row[j] = current
			bl = current
		}

		result = float64(current)
	}

	return result
}

func levenshtein_with_operations(a, b string) [4]float64 {
	results := [4]float64{0, 0, 0, 0}

	// basic cases - empty string edit distance equal to length of other string b/c only n insertions needed
	if a == "" || b == "" {
		return [4]float64{(float64(max(len(a), len(b))))}
	}

	// same string, no edit distance
	if a == b {
		return results
	}

	lenA := len(a)
	lenB := len(b)

	// matrix of levenshtein distances between each substring
	matrix := make([][]int, lenA+1)
	// operations
	// 1 - substitution
	// 2 - insertion/deletion
	// 3 - transposition
	ops := make([][]int, lenA+1)

	// fill matrix w/ initial table values
	for i := range matrix {
		matrix[i] = make([]int, lenB+1)
		matrix[i][0] = i
		ops[i] = make([]int, lenB+1)
		ops[i][0] = 1
	}

	for i := range matrix[0] {
		matrix[0][i] = i
		ops[0][i] = 1
	}

	for i := 1; i < lenA+1; i++ {
		for j := 1; j < lenB+1; j++ {
			cost := 0
			// not the same character
			if a[i-1] != b[j-1] {
				cost = 1
			}

			ins := matrix[i][j-1] + 1
			del := matrix[i-1][j] + 1
			sub := matrix[i-1][j-1] + cost

			matrix[i][j] = min(ins, del, sub)

			// calculate if transposition can be used
			var trans int = -1
			if (i > 1 && j > 1) && a[i-2] == b[j-1] && a[i-1] == b[j-2] {
				trans = matrix[i-2][j-2] + cost
				matrix[i][j] = min(matrix[i][j], trans)
			}

			m := matrix[i][j]
			switch {
			// substitution
			case m == sub && cost == 1:
				ops[i][j] = 1
			// insertion
			case m == ins:
				ops[i][j] = 2
			// deletion
			// insertions are combined w/ deletions in final count, but are retained here so that backtracking can work properly
			case m == del:
				ops[i][j] = 3
			// transposition
			case m == trans:
				ops[i][j] = 4
			}
		}
	}
	results[0] = float64(matrix[lenA][lenB])

	for lenA > -1 || lenB > -1 {
		// for strings w/ different sizes, ensures algorithm will run down entire length of array rather than quitting once we iterate over
		// the length of one of the strings
		decA := 1
		decB := 1
		if lenA == 0 {
			decA = 0
		}

		if lenB == 0 {
			decB = 0
		}

		if (decB == 0) && decA == 0 {
			break
		}

		op := ops[lenA][lenB]
		switch op {
		case 1:
			results[1]++
			lenA -= decA
			lenB -= decB
		case 2:
			results[2]++
			lenB -= decB
			if lenB == 0 {
				lenA -= decA
			}
		case 3:
			results[2]++
			lenA -= decA
			if lenA == 0 {
				lenB -= decB
			}
		case 4:
			results[3]++
			lenA -= decA * 2
			lenB -= decB * 2
		default:
			lenA -= decA
			lenB -= decB
		}
	}

	return results
}
