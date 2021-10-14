package msg_test

import (
	"OrgTimer/msg"
	"fmt"
	"testing"
	"time"
)

func genMsgs(baseTime time.Time, number int) (res msg.MsgList) {
	for i := 0; i < number; i++ {
		msg := msg.NewOrgMsg(fmt.Sprintf("title[%v]", i), "", baseTime.Add(time.Hour))
		res = append(res, &msg)
	}
	return res
}

func TestMergeTerminal(t *testing.T) {
	baseTime := time.Now()
	tMsgs := msg.NewTerminalMsg("t", "", baseTime)
	oldMsgs := genMsgs(baseTime, 4)
	oriMsgs := append(oldMsgs, &tMsgs)
	oldMsgs = append(oldMsgs, &tMsgs)

	newMsgs := genMsgs(baseTime, 4)

	oldMsgs.Merge(newMsgs)
	if !oldMsgs.EqualsAll(oriMsgs) {
		t.Errorf("wrong, ori:%v, merged:%v", oriMsgs.ToString(), oldMsgs.ToString())
	}
}

func TestMergeRemoved(t *testing.T) {
	baseTime := time.Now()
	oldMsgs := genMsgs(baseTime, 4)

	newMsgs := genMsgs(baseTime, 3)

	oldMsgs.Merge(newMsgs)
	if !oldMsgs.EqualsAll(newMsgs) {
		t.Errorf("wrong, ori:%v, merged:%v", genMsgs(baseTime, 4).ToString(), oldMsgs.ToString())
	}
}

func TestMergeActiv(t *testing.T) {
	baseTime := time.Now()
	oldMsgs := genMsgs(baseTime, 4)
	oldMsgs[0].Activate()
	oriMsgs := genMsgs(baseTime, 4)
	oriMsgs[0].Activate()

	if !oldMsgs[0].IsActiv() {
		t.Errorf("msg:%v should be activ", oldMsgs[0].ToString())
	}
	newMsgs := genMsgs(baseTime, 3)

	resMsgs := genMsgs(baseTime, 3)
	resMsgs[0].Activate()

	oldMsgs.Merge(newMsgs)
	if !oldMsgs.EqualsAll(resMsgs) {
		t.Errorf("wrong, oldMsgs:%v, merged:%v, resMsgs:%v", oriMsgs.ToString(), oldMsgs.ToString(), resMsgs.ToString())
	}
}

func TestMergeInit(t *testing.T) {
	baseTime := time.Now()
	var oldMsgs msg.MsgList

	newMsgs := genMsgs(baseTime, 3)

	resMsgs := genMsgs(baseTime, 3)
	// resMsgs[0].Activate()

	oldMsgs.Merge(newMsgs)
	if !oldMsgs.EqualsAll(resMsgs) {
		t.Errorf("wrong, oldMsgs:%v, merged:%v, resMsgs:%v", genMsgs(baseTime, 0).ToString(), oldMsgs.ToString(), resMsgs.ToString())
	}
}

func TestClean(t *testing.T) {
	baseTime := time.Now()
	oldMsgs := genMsgs(baseTime, 4)
	newMsgs := genMsgs(baseTime, 4)
	newMsgs[3].Cancel()
	newMsgs.Clean()
	if !newMsgs.EqualsAll(genMsgs(baseTime, 3)) {
		t.Errorf("wrong, old:%v, merged:%v", oldMsgs.ToString(), newMsgs.ToString())
	}
}
