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

func (b *BellCompleter) Do(line []rune, pos int) ([][]rune, int) {
	current := string(line[:pos])

	if current == b.lastLine {
    b.tabCount++
} else {
    b.tabCount = 1
    b.lastLine = current
}

	candidates, length := b.inner.Do(line, pos)

	if len(candidates) == 0 {
		tty, _ := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
		if tty != nil {
			defer tty.Close()
			tty.Write([]byte("\a"))
		}
		return candidates, length

	} else if len(candidates) == 1 {
		full := current + string(candidates[0])
		full = strings.TrimRight(full, " ")
		suffix := full[len(current):]
		withSpace := []rune(suffix + " ")
		b.tabCount = 0
		return [][]rune{withSpace}, length

	} else {
		  var names []string
    for _, c := range candidates {
        name := current + strings.TrimRight(string(c), " ")
        names = append(names, name)
    }
    sort.Strings(names)

    os.Stdout.Write([]byte("\r\n"))
    os.Stdout.Write([]byte(strings.Join(names, "  ")))
    os.Stdout.Write([]byte("\r\n"))
    os.Stdout.Write([]byte("$ " + current))

    b.tabCount = 0
    return [][]rune{[]rune("")}, 0
	}
}
func main() {
	builtins := []readline.PrefixCompleterInterface{
		readline.PcItem("echo"),
		readline.PcItem("cd"),
		readline.PcItem("pwd"),
		readline.PcItem("exit"),
		readline.PcItem("type"),
	}

	// combine builtins + PATH commands
	allItems := append(builtins, getPathCommands()...)
	completer := &BellCompleter{
		inner: readline.NewPrefixCompleter(
			allItems...,
		),
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
		info, err := os.Stat(path)
		if path == "~" {
			os.Chdir(os.Getenv("HOME"))
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stdout, "cd: %s: No such file or directory\n", path)
		} else if os.IsNotExist(err) {
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
		content := makeStringForEcho(input[:redirectIdx])

		err := os.MkdirAll(filepath.Dir(filePath), 0755)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			return
		}

		var file *os.File
		if redirectType == ">>" || redirectType == "1>>" {
			// append mode — O_CREATE handles file not existing
			file, err = os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			// truncate mode
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
			_, err = file.WriteString(content + "\n")
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			}
		}
	} else {
		fmt.Fprintf(os.Stdout, "%s\n", makeStringForEcho(input))
	}
}
func makeStringForEcho(input []string) string {
	var out bytes.Buffer
	for i, str := range input {
		if i == len(input)-1 {
			out.WriteString(str)
		} else {
			out.WriteString(str)
			out.WriteString(" ")
		}
	}
	rawInput := out.String()
	args := parseArgs(rawInput)
	var result bytes.Buffer
	for i, str := range args {
		if i == len(input)-1 {
			result.WriteString(str)
		} else {
			result.WriteString(str)
			result.WriteString(" ")
		}
	}
	return out.String()

}

func parseArgs(input string) []string {
	var args []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false

	for i := 0; i < len(input); i++ {
		ch := input[i]
		switch {

		// --- single quote ---
		case ch == '\'' && !inSingleQuote && !inDoubleQuote:
			inSingleQuote = true

		case ch == '\'' && inSingleQuote:
			inSingleQuote = false

		// --- double quote ---
		case ch == '"' && !inDoubleQuote && !inSingleQuote:
			inDoubleQuote = true

		case ch == '"' && inDoubleQuote:
			inDoubleQuote = false

		// --- backslash outside quotes ---
		case ch == '\\' && !inSingleQuote && !inDoubleQuote:
			i++
			if i < len(input) {
				current.WriteByte(input[i])
			}

		// --- backslash inside double quotes ---
		// only escapes special characters
		case ch == '\\' && inDoubleQuote:
			if i+1 < len(input) {
				next := input[i+1]
				if next == '"' || next == '\\' || next == '$' || next == '`' || next == '\n' {
					i++
					current.WriteByte(input[i])
				} else {
					current.WriteByte(ch) // literal backslash
				}
			}

		// --- backslash inside single quotes ---
		// always literal, handled by default (no case needed)

		// --- space ---
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
	commandFound := false
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
					commandFound = true
					return true, dir, entry.Name()
				}
			} else {
				continue
			}
		}
	}
	return commandFound, "", ""
}

func handleCommand(command string, rest []string) {
	commandFound, _, _ := commandExists(command)
	if !commandFound {
		fmt.Fprintf(os.Stderr, "%s: command not found\n", command)
		return
	}

	// find redirect
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
			// O_CREATE handles file not existing — no need for os.Stat
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

func getPathCommands() []readline.PrefixCompleterInterface {
	var items []readline.PrefixCompleterInterface

	seen := make(map[string]bool) // avoid duplicates
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
