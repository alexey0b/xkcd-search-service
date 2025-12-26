package words_test

import (
	"search-service/words/words"
	"testing"

	"github.com/stretchr/testify/require"
)

var testCases = []struct {
	desc     string
	given    string
	expected []string
}{
	{
		desc:     "empty",
		given:    "",
		expected: []string{},
	},
	{
		desc:     "simple",
		given:    "simple",
		expected: []string{"simpl"},
	},
	{
		desc:     "followers",
		given:    "I follow followers",
		expected: []string{"follow"},
	},
	{
		desc:     "punctuation",
		given:    "I shouted: 'give me your car!!!",
		expected: []string{"shout", "give", "car"},
	},
	{
		desc:     "stop words only",
		given:    "I and you or me or them, who will?",
		expected: []string{},
	},
	{
		desc:     "weird",
		given:    "Moscow!123'check-it'or   123, man,that,difficult:heck",
		expected: []string{"moscow", "check", "123", "man", "difficult", "heck"},
	},
	{
		desc:     "numbers only",
		given:    "123 456 789",
		expected: []string{"123", "456", "789"},
	},
	{
		desc:     "mixed case",
		given:    "GoLang GOLANG golang",
		expected: []string{"golang"},
	},
	{
		desc:     "multiple spaces",
		given:    "hello    world     test",
		expected: []string{"hello", "world", "test"},
	},
	{
		desc:     "special characters",
		given:    "test@email.com #hashtag $100",
		expected: []string{"test", "email", "com", "hashtag", "100"},
	},
	{
		desc:     "single character",
		given:    "a b c",
		expected: []string{"b", "c"},
	},
	{
		desc:     "long word",
		given:    "supercalifragilisticexpialidocious",
		expected: []string{"supercalifragilisticexpialidoci"},
	},
}

func TestWords(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			keywords := words.Norm(tc.given)
			require.ElementsMatch(t, tc.expected, keywords)
		})
	}
}
