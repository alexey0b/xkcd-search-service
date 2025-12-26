package words

import (
	"strings"
	"unicode"

	"github.com/kljensen/snowball/english"
)

func Norm(phrase string) []string {
	filteredSeq := strings.FieldsFunc(phrase, func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	})
	keywords := make([]string, 0)
	dict := make(map[string]bool)
	for _, word := range filteredSeq {
		stemmed := english.Stem(word, true)
		if !english.IsStopWord(stemmed) && !dict[stemmed] {
			keywords = append(keywords, stemmed)
			dict[stemmed] = true
		}
	}
	return keywords
}
