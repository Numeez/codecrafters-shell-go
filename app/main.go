package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

var _ = fmt.Print

type BellCompleter struct {
	inner    readline.AutoCompleter
	lastLine string
	tabCount int
}

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

func getFileCompletions(prefix string) []string {
	var dir, base string

	if strings.Contains(prefix, "/") {
		if strings.HasSuffix(prefix, "/") {
			dir = prefix
			base = ""
		} else {
			dir = filepath.Dir(prefix)
			base = filepath.Base(prefix)
		}
	} else {
		dir = "."
		base = prefix
	}

	if strings.HasPrefix(dir, "~") {
		dir = strings.Replace(dir, "~", os.Getenv("HOME"), 1)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var matches []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, base) {
			var full string
			if dir == "." {
				full = name
			} else {
				full = filepath.Join(dir, name)
			}
			if entry.IsDir() {
				full += "/"
			}
			matches = append(matches, full)
		}
	}
	return matches
}

func (b *BellCompleter) Do(line []rune, pos int) ([][]rune, int) {
	current := string(line[:pos])

	// Determine the last word being typed
	parts := strings.Fields(current)
	lastWord := ""
	if len(parts) > 0 {
		lastWord = parts[len(parts)-1]
	}
	if strings.HasSuffix(current, " ") {
		lastWord = ""
	}

	// Get PATH/builtin candidates from inner completer
	candidates, length := b.inner.Do(line, pos)

	seen := map[string]bool{}
	var unique []string

	for _, c := range candidates {
		s := strings.TrimRight(string(c), " ")
		if !seen[s] {
			seen[s] = true
			unique = append(unique, s)
		}
	}

	// Also get file completions from current dir (or path)
	if lastWord != "" {
		fileMatches := getFileCompletions(lastWord)
		for _, m := range fileMatches {
			// m is the full name e.g. "main.go"
			// convert to suffix by stripping already-typed prefix
			if len(m) >= len(lastWord) {
				suffix := m[len(lastWord):]
				if !seen[suffix] {
					seen[suffix] = true
					unique = append(unique, suffix)
				}
			}
		}
		// length should cover the lastWord so readline replaces it correctly
		length = len([]rune(lastWord))
	}

	switch len(unique) {
	case 0:
		// No match — ring bell
		b.lastLine = ""
		b.tabCount = 0
		tty, _ := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if tty != nil {
			defer tty.Close()
			tty.Write([]byte("\a"))
		}
		return [][]rune{}, 0

	case 1:
		// Single match — complete with trailing space (or / if directory)
		b.lastLine = ""
		b.tabCount = 0
		suffix := unique[0]
		fullName := lastWord + suffix
		if strings.HasSuffix(fullName, "/") {
			// directory — no trailing space
			return [][]rune{[]rune(suffix)}, length
		}
		return [][]rune{[]rune(suffix + " ")}, length

	default:
		// Multiple matches — compute LCP of suffixes
		lcp := longestCommonPrefix(unique)

		if lcp != "" {
			// LCP adds something — complete to it silently
			b.lastLine = ""
			b.tabCount = 0
			return [][]rune{[]rune(lcp)}, length
		}

		// LCP adds nothing — bell on first tab, show options on second
		if current != b.lastLine {
			b.lastLine = current
			b.tabCount = 1
			tty, _ := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
			if tty != nil {
				defer tty.Close()
				tty.Write([]byte("\a"))
			}
			return [][]rune{}, 0
		}

		b.tabCount++
		if b.tabCount == 2 {
			// Reconstruct full names for display
			var names []string
			for _, s := range unique {
				names = append(names, lastWord+s)
			}
			sort.Strings(names)
			os.Stdout.Write([]byte("\r\n"))
			os.Stdout.Write([]byte(strings.Join(names, "  ")))
			os.Stdout.Write([]byte("\r\n"))
			os.Stdout.Write([]byte("$ " + current))
			b.tabCount = 0
		}
		return [][]rune{[]rune("")}, 0
	}
}

func main() {
	builtinNames := map[string]bool{
		"echo": true, "cd": true, "pwd": true, "exit": true, "type": true,
	}

	builtins := []readline.PrefixCompleterInterface{
		readline.PcItem("echo"),
		readline.PcItem("cd"),
		readline.PcItem("pwd"),
		readline.PcItem("exit"),
		readline.PcItem("type"),
	}

	allItems := append(builtins, getPathCommands(builtinNames)...)
	completer := &BellCompleter{
		inner: readline.NewPrefixCompleter(allItems...),
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "$ ",
		AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		VimMode:         false,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		handleInput(line)
	}
}

func handleInput(input string) {
	args := parseArgs(input)
	command := args[0]
	rest := args[1:]
	switch command {
	case "exit":
		os.Exit(0)
	case "echo":
		handleEcho(rest)
	case "type":
		typeCommand(makeString(rest))
	case "pwd":
		pwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stdout, "%s\n", err.Error())
		} else {
			fmt.Fprintf(os.Stdout, "%s\n", pwd)
		}
	case "":
		return
	case "cd":
		path := rest[0]
		if path == "~" {
			os.Chdir(os.Getenv("HOME"))
			break
		}
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stdout, "cd: %s: No such file or directory\n", path)
		} else if !info.IsDir() {
			fmt.Fprintf(os.Stdout, "cd: %s: No such file or directory\n", path)
		} else {
			err := os.Chdir(path)
			if err != nil {
				fmt.Fprintf(os.Stdout, "%s\n", err.Error())
			}
		}
	default:
		handleCommand(command, rest)
	}
}

func makeString(input []string) string {
	var out bytes.Buffer
	for i, str := range input {
		if i == len(input)-1 {
			out.WriteString(str)
		} else {
			out.WriteString(str)
			out.WriteString(" ")
		}
	}
	return out.String()
}

func handleEcho(input []string) {
	redirectIdx := -1
	redirectType := ""
	for i, arg := range input {
		if arg == ">" || arg == "1>" || arg == "2>" ||
			arg == ">>" || arg == "1>>" || arg == "2>>" {
			redirectIdx = i
			redirectType = arg
			break
		}
	}

	if redirectIdx != -1 {
		filePath := input[redirectIdx+1]
		content := makeString(input[:redirectIdx])

		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			return
		}

		var file *os.File
		if redirectType == ">>" || redirectType == "1>>" {
			file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			file, err = os.Create(filePath)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			return
		}
		defer file.Close()

		if redirectType == "2>" || redirectType == "2>>" {
			fmt.Fprintf(os.Stdout, "%s\n", content)
		} else {
			file.WriteString(content + "\n")
		}
	} else {
		fmt.Fprintf(os.Stdout, "%s\n", makeString(input))
	}
}

func parseArgs(input string) []string {
	var args []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {
		case ch == '\'' && !inSingleQuote && !inDoubleQuote:
			inSingleQuote = true
		case ch == '\'' && inSingleQuote:
			inSingleQuote = false
		case ch == '"' && !inDoubleQuote && !inSingleQuote:
			inDoubleQuote = true
		case ch == '"' && inDoubleQuote:
			inDoubleQuote = false
		case ch == '\\' && !inSingleQuote && !inDoubleQuote:
			i++
			if i < len(input) {
				current.WriteByte(input[i])
			}
		case ch == '\\' && inDoubleQuote:
			if i+1 < len(input) {
				next := input[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' || next == '\n' {
					i++
					current.WriteByte(input[i])
				} else {
					current.WriteByte(ch)
				}
			}
		case ch == ' ' && !inSingleQuote && !inDoubleQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

func typeCommand(command string) {
	switch command {
	case "exit", "echo", "type", "pwd":
		fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", command)
	default:
		commandFound, dir, file := commandExists(command)
		if !commandFound {
			fmt.Fprintf(os.Stdout, "%s: not found\n", command)
		} else {
			fmt.Fprintf(os.Stdout, "%s is %s/%s\n", command, dir, file)
		}
	}
}

func commandExists(command string) (bool, string, string) {
	directories := strings.Split(os.Getenv("PATH"), ":")
	for _, dir := range directories {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.Name() == command {
				info, err := os.Stat(filepath.Join(dir, entry.Name()))
				if err != nil {
					continue
				}
				if info.Mode()&0111 != 0 {
					return true, dir, entry.Name()
				}
			}
		}
	}
	return false, "", ""
}

func handleCommand(command string, rest []string) {
	commandFound, _, _ := commandExists(command)
	if !commandFound {
		fmt.Fprintf(os.Stderr, "%s: command not found\n", command)
		return
	}

	redirectIdx := -1
	redirectType := ""
	for i, arg := range rest {
		if arg == ">" || arg == "1>" || arg == "2>" ||
			arg == ">>" || arg == "1>>" || arg == "2>>" {
			redirectIdx = i
			redirectType = arg
			break
		}
	}

	var cmd *exec.Cmd

	if redirectIdx != -1 {
		cmdArgs := rest[:redirectIdx]
		filePath := rest[redirectIdx+1]

		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			return
		}

		var file *os.File
		if redirectType == ">>" || redirectType == "1>>" || redirectType == "2>>" {
			file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			file, err = os.Create(filePath)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			return
		}
		defer file.Close()

		cmd = exec.Command(command, cmdArgs...)
		if redirectType == "2>" || redirectType == "2>>" {
			cmd.Stdout = os.Stdout
			cmd.Stderr = file
		} else {
			cmd.Stdout = file
			cmd.Stderr = os.Stderr
		}
	} else {
		cmd = exec.Command(command, rest...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil && redirectIdx == -1 {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
	}
}

func getPathCommands(builtinNames map[string]bool) []readline.PrefixCompleterInterface {
	var items []readline.PrefixCompleterInterface
	seen := make(map[string]bool)
	for name := range builtinNames {
		seen[name] = true
	}

	dirs := strings.Split(os.Getenv("PATH"), ":")
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name := entry.Name()
			if !seen[name] {
				seen[name] = true
				items = append(items, readline.PcItem(name))
			}
		}
	}
	return items
}