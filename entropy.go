package main

import (
	"math"
	"strings"
)

type EntropySearch struct {
	Alphabet      string
	EntropyBorder float64
	RuleName      string
}

var entropySearches = []*EntropySearch{
	&EntropySearch{
		Alphabet:      "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
		EntropyBorder: 4.5,
		RuleName:      "Base64 entropy",
	},
	&EntropySearch{
		Alphabet:      "0123456789abcdefABCDEF",
		EntropyBorder: 3.0,
		RuleName:      "Hexadecimal entropy",
	},
}

//GetEntropy calculates Shannon entropy in bits
func getEntropy(data string, es *EntropySearch) (entropy float64) {
	entropy = 0
	if data == "" {
		return
	}
	for _, c := range es.Alphabet {
		px := float64(strings.Count(data, string(c))) / float64(len(data))
		if px > 0 {
			entropy += -px * math.Log2(px)
		}
	}
	return
}
