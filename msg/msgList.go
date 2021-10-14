package msg

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"
)

const (
	orgFilePath = "/home/weiss/Documents/Org/orgzly/todo.org"
)

type MsgList []*Msg

func (msgs MsgList) Write (){
	f , err := os.Create("/home/weiss/Documents/Org/orgzly/todo.org")
	if err != nil {
		panic(err)
	}
	str := "* todo"
	for _, x := range msgs{
		str += "\n" + x.ToOrgSchedule() 
	}
	f.WriteString(str)
}

func (msgs MsgList) isIndexValid(index int) error {
	if len(msgs) < index+1 {
		return errors.New("please enter the timer id that you want to cancel!")
	}
	return nil
}

func (msgs MsgList) CancelMsg(index int) error {
	if err := msgs.isIndexValid(index); err != nil {
		return err
	}
	msgs[index].Cancel()
	return nil
}

func (msgs *MsgList) sort() {
	sort.SliceStable(*msgs, func(i, j int) bool {
		return (*msgs)[j].earlier(*(*msgs)[i])
	})
}

// filter MsgList, the first returned value is filted list, the second is the rest
func (msgs MsgList) filter(f func(*Msg) bool) (res MsgList, rest MsgList) {
	for _, x := range msgs {
		if f(x) {
			res = append(res, x)
		} else {
			rest = append(rest, x)
		}
	}
	return res, rest
}

func (msgs MsgList) GetOrgMsgs() (newMsgs MsgList) {
	f := func(msg *Msg) bool {
		return msg.msgType == Org
	}
	res, _ := msgs.filter(f)
	return res
}

func (msgs MsgList) GetActivMsgs() (newMsgs MsgList) {
	f := func(msg *Msg) bool {
		return msg.state == Activ
	}
	res, _ := msgs.filter(f)
	return res
}

func (msgs MsgList) GetUndisabledMsgs() (newMsgs MsgList) {
	f := func(msg *Msg) bool {
		return msg.state != Disabled
	}
	res, _ := msgs.filter(f)
	return res
}

func (msgs MsgList) GetWaitMsgs() (newMsgs MsgList) {
	f := func(msg *Msg) bool {
		return msg.state == Wait
	}
	res, _ := msgs.filter(f)
	return res
}

// the date must later than now
func (msgs MsgList) GetValidMsgs() (newMsgs MsgList) {
	f := func(msg *Msg) bool {
		return msg.endTime.After(time.Now())
	}
	res, _ := msgs.filter(f)
	return res
}

func (msgs MsgList) Contains(msg Msg) bool {
	for _, x := range msgs {
		if x.equals(msg) {
			return true
		}
	}
	return false
}

func (msgs *MsgList) Clean() {
	*msgs = msgs.GetUndisabledMsgs().GetValidMsgs()
	// .GetValidMsgs()
}

func (msgs MsgList) ContainsAll(msg Msg) bool {
	for _, x := range msgs {
		if x.equalsAll(msg) {
			return true
		}
	}
	return false
}

// (old msgs && from terminal) + (old msgs && activ && exists in new) + (new msgs && rest)
func (msgs *MsgList) Merge(newMsgs MsgList) {
	res, rest := msgs.filter(func(msg *Msg) bool {
		return msg.msgType == Terminal
	})
	for _, x := range rest {
		if x.state == Activ && newMsgs.Contains(*x) {
			res = append(res, x)
			newMsgs, _ = newMsgs.filter(func(msg *Msg) bool {
				return !msg.equals(*x)
			})
		}
	}
	*msgs = append(res, newMsgs...)
}

func (msgs MsgList) GetMsgsWithinMin(min float64) (res MsgList) {
	f := func(msg *Msg) bool {
		return msg.WithinMin(min)
	}
	res, _ = msgs.filter(f)
	return res
}

func (msgs MsgList) ActivateAll() {
	for _, x := range msgs {
		x.Activate()
	}
}

func (msgs MsgList) ToString() (res string) {
	for i, x := range msgs {
		res += fmt.Sprintf("\n[%v]%v", magenta(i), x.ToString())
	}
	return res
}

func (msgs MsgList) PrintAll() {
	msgs.sort()
	if len(msgs) == 0 {
		fmt.Println(red("there is current no timer"))
	} else {
		fmt.Printf("%v", msgs.ToString())
	}
}

func (msgs MsgList) EqualsAll(other MsgList) bool {
	for _, x := range msgs {
		if !other.ContainsAll(*x) {
			return false
		}
	}
	for _, x := range other {
		if !msgs.ContainsAll(*x) {
			return false
		}
	}
	return true
}
