package spell

import "testing"

type LdResult struct {
	one, two string
	dist     uint8
}

func TestLd(t *testing.T) {
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
		l := Ld(v.one, v.two)
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
func BenchmarkLd(b *testing.B) {
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
		Ld(one, two)
		b.StopTimer()
	})
}
