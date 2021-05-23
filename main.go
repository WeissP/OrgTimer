package main

import (
	"OrgTimer/timerMsg"
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gookit/color"
)

var (
	red   = color.FgRed.Render
	green = color.FgGreen.Render
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
		fmt.Printf("new Timer with %v\n", green(arrInput[1]))
	case "c":
		if l == 1 {
			return errors.New("please enter the timer id that you want to cancel!")
		}
		id, err := strconv.Atoi(arrInput[1])
		if err != nil {
			return errors.New(fmt.Sprintf("except time but was: %s", arrInput[1]))
		}
		timerMsg.CancelTimer(id)
	case "l":
		timerMsg.PrintAll()
	case "a":
		timerMsg.AllChangedFiles()
	case "r":
		timerMsg.RefreshAllFiles()
		timerMsg.PrintAll()
	case "q":
		os.Exit(0)
	}
	return nil
}

func main() {
	// signal.Ignore(syscall.SIGINT)
	reader := bufio.NewReader(os.Stdin)
	timerMsg.PrintAll()
	for {
		fmt.Print("OrgTimer>>>> ")
		// NewHeader <- true
		// Read the keyboad input.
		input, err := reader.ReadString('\n')
		if err != nil {
			// fmt.Printf("%v", red(err))
			fmt.Fprintln(os.Stderr, err)
		}

		// Handle the execution of the input.
		if err = execInput(input); err != nil {
			fmt.Fprintln(os.Stderr, red(err))
		}
	}
}
