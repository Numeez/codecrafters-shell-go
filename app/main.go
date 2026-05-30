package main

import (
	"bufio"
	"fmt"
	"os"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Print

func main() {

	fmt.Print("$ ")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		readInput := scanner.Text()
		fmt.Fprintf(os.Stdout, "%s: command not found", readInput)
	}
}
