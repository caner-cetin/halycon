package internal

import (
	"os"
	"strings"
	"unicode"

	"github.com/rs/zerolog/log"
)

func OpenFile(input string) *os.File {
	f, err := os.Open(input)
	if err != nil {
		ev := log.With().Str("path", input).Err(err).Logger()
		if os.IsNotExist(err) {
			ev.Fatal().Msg("path does not exist")
		}
		ev.Fatal().Msg("unknown error while opening file")
	}
	return f
}

func CleanUPC(input string) string {
	var result strings.Builder
	for _, char := range input {
		if unicode.IsDigit(char) {
			result.WriteRune(char)
		}
	}
	return result.String()
}
