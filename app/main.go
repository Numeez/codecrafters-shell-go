package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
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
	inputs := strings.Split(input, " ")
	var command string
	var rest []string
	if len(inputs) > 0 {
		command = inputs[0]
		if len(inputs[1:])!=0{
			rest = inputs[1:]
		}
	}
	switch command {
	case "exit":
		os.Exit(0)
	case "echo":
		fmt.Fprintf(os.Stdout, "%s\n",makeString(rest))
	case "type":
		typeCommand(makeString(rest))
	default:
		fmt.Fprintf(os.Stdout, "%s: command not found", input)
		fmt.Fprintf(os.Stdout, "\n")

	}
}


func makeString(input []string)string{
	var out bytes.Buffer
	for i,str:= range input{
		if i==len(input)-1{
		out.WriteString(str)
		}else{
			out.WriteString(str)
			out.WriteString(" ")
		}
	}
	return  out.String()
}

func typeCommand(command string){
	switch command{
	case "exit","echo":
		fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", command)
	default:
		fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
	}


}
