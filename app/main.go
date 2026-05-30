package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Print

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Fprint(os.Stdout, "$ ")
		if !scanner.Scan() {
			os.Exit(1)
		}
		readInput := scanner.Text()
		handleInput(readInput)

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
		commandFound, _, _ := commandExists(command)
		if commandFound {
			cmd := exec.Command(command, rest...)
			output, err := cmd.Output()
			if err != nil {
				fmt.Fprintf(os.Stdout, "%s\n", err.Error())
			} else {
				fmt.Fprintf(os.Stdout, "%s", string(output))
			}

		} else {
			fmt.Fprintf(os.Stdout, "%s: command not found", input)
			fmt.Fprintf(os.Stdout, "\n")
		}

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
    length := len(input)

    // check if second-to-last element is ">"
    if length >= 3 && input[length-2] == ">" {
        fileName := input[length-1]              // file is the last element
        content := makeStringForEcho(input[:length-2]) // everything before ">"

        file, err := os.Create(fileName)
        if err != nil {
            fmt.Fprintf(os.Stderr, "%s\n", err.Error())
            return
        }
        defer file.Close()

        _, err = file.WriteString(content + "\n")
        if err != nil {
            fmt.Fprintf(os.Stderr, "%s\n", err.Error())
            return
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
