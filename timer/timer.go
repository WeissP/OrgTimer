package timer

import (
	"OrgTimer/connector"
	"OrgTimer/msg"
	"sync"
	"time"

	"github.com/gookit/color"
)

const (
	activateThreshold = 30 // activate msg within activateThreshold(min)
	updateInterval    = 5 * time.Minute
)

var (
	refreshMutex sync.Mutex
	allMsgs      msg.MsgList
	today        int
	red          = color.FgRed.Render
	green        = color.FgGreen.Render
	blue         = color.FgLightBlue.Render
)

func init() {
	Refresh()
	go func() {
		c := time.Tick(updateInterval)
		for range c {
			Refresh()
		}
	}()
}

// create default timer from terminal
func NewDefault(dur string) error {
	d, err := time.ParseDuration(dur)
	if err != nil {
		return err
	}
	newTimer("timeup", "", time.Now().Add(d))
	return nil
}

// add new timer to allMsgs and enable it if needed
func newTimer(title, content string, endTime time.Time) {
	msg := msg.NewTerminalMsg(title, content, endTime)
	allMsgs = append(allMsgs, &msg)
	if msg.WithinMin(activateThreshold) {
		msg.Activate()
	}
}

// cancel the timer with the index of allMsgs, return error if index is not valid
func CancelTimer(index int) error {
	return allMsgs.CancelMsg(index)
}

func Write() {
	allMsgs.GetOrgMsgs().Write()
}

// update msgs from datebase
func updateMsgs() {
	oldMsgs := allMsgs
	newMsgs := connector.GetSchedule().GetValidMsgs()
	refreshMutex.Lock()
	allMsgs.Merge(newMsgs)
	if !oldMsgs.EqualsAll(allMsgs) {
		Write()
	}
	allMsgs.Clean()
	refreshMutex.Unlock()
}

// activate all msgs within threshold
func activateMsgs() {
	allMsgs.GetWaitMsgs().GetMsgsWithinMin(activateThreshold).ActivateAll()
}

// update msgs and then activate them if needed
func Refresh() {
	updateMsgs()
	activateMsgs()
}

func PrintAllActivMsgs() {
	allMsgs.GetActivMsgs().PrintAll()
}

func PrintAllMsgs() {
	allMsgs.Clean()
	allMsgs.PrintAll()
}
