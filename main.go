package main

import (
	"OrgTimer/timerMsg"
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
)

func execInput(input string) error {
	// Remove the newline character.
	input = strings.TrimSuffix(input, "\n")
	arrInput := strings.Fields(input)
	l := len(arrInput)
	if l == 0 {
		return nil
	}
	switch arrInput[0] {
	case "n":
		if l == 1 {
			return errors.New("please enter time!")
		}
		m, err := timerMsg.NewDefault(arrInput[1])
		if err != nil {
			return err
		}
		go m.Start()
		fmt.Printf("new Timer with %v\n", arrInput[1])
	case "l":
		timerMsg.PrintAll()
	case "q":
		os.Exit(0)
	}

	return nil
}

func main() {
	// signal.Ignore(syscall.SIGINT)
	reader := bufio.NewReader(os.Stdin)
	go timerMsg.IncId()
	for {
		fmt.Print("OrgTimer>>>> ")
		// Read the keyboad input.
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		// Handle the execution of the input.
		if err = execInput(input); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}
