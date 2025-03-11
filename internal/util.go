package internal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
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

// https://stackoverflow.com/questions/39320371/how-start-web-server-to-open-page-in-browser-in-golang
// openURL opens the specified URL in the default browser of the user.
func OpenURL(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		// Check if running under WSL
		if isWSL() {
			// Use 'cmd.exe /c start' to open the URL in the default Windows browser
			cmd = "cmd.exe"
			args = []string{"/c", "start", url}
		} else {
			// Use xdg-open on native Linux environments
			cmd = "xdg-open"
			args = []string{url}
		}
	}
	if len(args) > 1 {
		// args[0] is used for 'start' command argument, to prevent issues with URLs starting with a quote
		args = append(args[:1], append([]string{""}, args[1:]...)...)
	}
	return exec.Command(cmd, args...).Start()
}

// isWSL checks if the Go program is running inside Windows Subsystem for Linux
func isWSL() bool {
	releaseData, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(releaseData)), "microsoft")
}

func PromptFor(message string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(message)
	text, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	return text
}
