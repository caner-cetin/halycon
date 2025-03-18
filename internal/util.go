package internal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"unicode"

	"github.com/rs/zerolog/log"
	"github.com/valyala/fastjson"
)

// OpenFile opens a file at the specified path and returns a file handle.
// It logs and exits the program with a fatal error if the file does not exist
// or if any other error occurs while opening the file.
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

// ReadFile reads files contents into memory, and returns the data as a byte slice.
func ReadFile(input string) ([]byte, error) {
	file := OpenFile(input)
	defer file.Close()
	contents, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading contents of %s: %w", input, err)
	}
	return contents, nil
}

// RemoveAllNonDigit removes all non-digit characters from the input string and returns
// the resulting string containing only digits.
func RemoveAllNonDigit(input string) string {
	var result strings.Builder
	for _, char := range input {
		if unicode.IsDigit(char) {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// OpenURL opens the specified URL in the system's default web browser.
// It supports Windows, macOS (darwin), Linux, and other Unix-like operating systems.
// For Windows Subsystem for Linux (WSL), it uses the Windows browser through cmd.exe.
// For other Linux/Unix systems, it uses xdg-open.
//
// https://stackoverflow.com/questions/39320371/how-start-web-server-to-open-page-in-browser-in-golang
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
	err := exec.Command(cmd, args...).Start()
	if err != nil {
		return fmt.Errorf("error executing %s %s: %w", cmd, strings.Join(args, ","), err)
	}
	return nil
}

// isWSL checks if the Go program is running inside Windows Subsystem for Linux
func isWSL() bool {
	releaseData, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(releaseData)), "microsoft")
}

// PromptFor displays a message to the user and returns the user's input as a string.
// It reads the input from standard input until a newline character is encountered.
func PromptFor(message string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(message)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read user input: %w", err)
	}
	text = strings.TrimSpace(text)
	return text, nil
}

// PromptForPickFromSlice prompts the user to select an item from a slice by displaying a message
// and returning the selected item. It accepts a generic type parameter T.
//
// Parameters:
//   - message: The prompt message to display to the user
//   - slice: A slice of type T from which the user will select an item
//
// Returns:
//   - T: The selected item from the slice
//   - int: index of selected item
//   - error: An error if:
//   - The user input could not be obtained
//   - The input is not a valid number
//   - The selected index is out of bounds
func PromptForPickFromSlice[T any](message string, slice []T) (T, int, error) {
	choice_str, err := PromptFor(message)
	if err != nil {
		var zero T
		return zero, 0, fmt.Errorf("failed to prompt user: %s", choice_str)
	}
	choice, err := strconv.Atoi(choice_str)
	if err != nil {
		var zero T
		return zero, 0, fmt.Errorf("invalid number input")
	}
	if choice >= len(slice) {
		var zero T
		return zero, 0, fmt.Errorf("choice is not in list")
	}
	return slice[choice], choice, nil
}

// DisplayInterface logs the contents of an interface{} after marshalling it to JSON.
// String values are unquoted before logging while other types are logged as raw bytes.
//
// Example:
//
//	type Person struct {
//		Name string
//		Age  int
//	}
//
//	people := []Person{
//		{Name: "Alice", Age: 30},
//		{Name: "Bob", Age: 25},
//	}
//	DisplayInterface(people)
//	// This will log each person's details as separate log messages
func DisplayInterface(i interface{}) {
	attrs_bytes, err := json.Marshal(i)
	if err != nil {
		log.Error().Err(err).Send()
	}
	attrs := fastjson.MustParseBytes(attrs_bytes)
	attrs.GetObject().Visit(func(key []byte, v *fastjson.Value) {
		ev := log.Info()
		for _, obj := range v.GetArray() {
			obj.GetObject().Visit(func(key []byte, v *fastjson.Value) {
				if v.Type() == fastjson.TypeString {
					marshalled, err := strconv.Unquote(string(v.MarshalTo(nil)))
					if err != nil {
						log.Error().Err(err).Send()
					}
					ev.Str(string(key), marshalled)
				} else {
					marshalled := v.MarshalTo(nil)
					ev.Bytes(string(key), marshalled)
				}
			})
		}
		ev.Msg(string(key))
	})
}

// CloseResponseBody safely closes the response body and logs any errors that occur during closing.
func CloseResponseBody(resp *http.Response) {
	if cerr := resp.Body.Close(); cerr != nil {
		log.Error().Err(fmt.Errorf("error closing response body: %w", cerr)).Send()
	}
}

// Ptr returns a pointer to the provided value of type T, useful for one liners
//
// Example:
//
//	str := "hello"
//	strPtr := Ptr("hello") // returns *string
func Ptr[T any](v T) *T {
	return &v
}
