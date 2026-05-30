package main

import (
	"bufio"
	"fmt"
	"os"
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


func handleInput(input string){
	switch input{
	case "exit":
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stdout, "%s: command not found", input)
		fmt.Fprintf(os.Stdout,"\n")
		
	}
}
