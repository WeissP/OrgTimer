package msg

import (
	"OrgTimer/common"
	"fmt"
	"os/exec"
	"time"

	"github.com/gen2brain/beeep"
	"github.com/gookit/color"
	"github.com/hako/durafmt"
)

var (
	red     = color.FgRed.Render
	green   = color.FgGreen.Render
	blue    = color.FgLightBlue.Render
	magenta = color.FgLightMagenta.Render
)

type State int

const (
	Wait = iota
	Activ
	Disabled
)

type MsgType int

const (
	Terminal = iota
	Org
)

type Msg struct {
	title, content string
	msgType        MsgType
	endTime        time.Time
	timer          *time.Timer
	state          State
}

func newMsg(msgType MsgType, title, content string, endTime time.Time) (m Msg) {
	m.title = title
	m.content = content
	m.endTime = endTime
	m.timer = nil
	m.state = Wait
	m.msgType = msgType
	return m
}

func NewOrgMsg(title, content string, endTime time.Time) (m Msg) {
	return newMsg(Org, title, content, endTime)
}

func NewTerminalMsg(title, content string, endTime time.Time) (m Msg) {
	return newMsg(Terminal, title, content, endTime)
}

func (msg *Msg) equals(other Msg) bool {
	return msg.title == other.title &&
		msg.content == other.content &&
		msg.msgType == other.msgType &&
		msg.endTime.Equal(other.endTime)
}

func (msg *Msg) equalsAll(other Msg) bool {
	return msg.equals(other) && msg.state == other.state
}

func (msg *Msg) earlier(other Msg) bool {
	return msg.endTime.Before(other.endTime)
}

func (msg *Msg) within(d time.Duration) bool {
	return time.Until(msg.endTime) < d
}

func (msg *Msg) WithinMin(min float64) bool {
	d, err := time.ParseDuration(fmt.Sprintf("%vm", min))
	if err != nil {
		panic(err)
	}
	return msg.within(d)
}

func (msg *Msg) notify() {
	err := beeep.Notify(msg.title, msg.content, "")
	if err != nil {
		panic(err)
	}
	exec.Command("mplayer", "-endpos", "5", "/home/weiss/Music/soft_alarm_2.mp3").Output()
}

func (msg Msg) ToOrgSchedule() string {
	return fmt.Sprintf("** TODO %v\n SCHEDULED: <%v>", msg.title, msg.endTime.Format("2006-01-02 15:04"))
}

func (msg *Msg) ToString() string {
	var (
		dura    = time.Until(msg.endTime)
		duraStr string
	)
	switch {
	case dura.Hours() > 24:
		duraStr = durafmt.Parse(time.Until(msg.endTime).Round(time.Hour)).String()
	case dura.Hours() > 1:
		duraStr = durafmt.Parse(time.Until(msg.endTime).Round(time.Minute)).String()
	default:
		duraStr = durafmt.Parse(time.Until(msg.endTime).Round(time.Second)).String()
	}
	return fmt.Sprintf("===========>\n%s:\nstate:%v \nstart at: %s\nremain: %s\n%s\n", msg.title, msg.state, green(msg.endTime.Format("15:04 02.01.2006")), green(duraStr), msg.content)
}

func (msg *Msg) disable() {
	msg.state = Disabled
}

func (msg *Msg) Activate() {
	f := func() {
		msg.notify()
		msg.disable()
	}
	timer := time.AfterFunc(time.Until(msg.endTime), f)
	msg.timer = timer
	msg.state = Activ
}

func (msg *Msg) Cancel() {
	if msg.state == Activ {
		msg.timer.Stop()
	}
	msg.disable()
	fmt.Printf(red("cancelled!\n"))
	common.PrintHeader()
}

func (msg *Msg) IsActiv() bool {
	return msg.state == Activ
}
