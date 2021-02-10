package timerMsg

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

type Msg struct {
	title, content, img, from string
	endTime                   time.Time
	timer                     *time.Timer
	id                        int
	cancel                    chan bool
}

type changedItems struct {
	added, removed []string
}

const (
	orgFilePath = "/home/weiss/Dropbox/Org-roam/daily/"
)

var (
	id           = make(chan int)
	msgMap       = make(map[int]Msg)
	ChangedItems changedItems
	refreshMutex sync.Mutex
	todoReg      = regexp.MustCompile(`^\*+ TODO (?P<title>.+)`)
	timeReg      = regexp.MustCompile(`.*SCHEDULED: <(?P<date>\d{4}-\d{2}-\d{2}) [a-zA-Z]{2} (?P<time>\d{2}:\d{2})>`)
)
var test = make(chan bool)

func CancelTimer(id int) {
	if x, ok := msgMap[id]; ok {
		x.cancel <- true
		delete(msgMap, id)
	} else {
		fmt.Printf("the id %v doesn't exist!\n", id)
	}
}

func NotifyChanged() {
	sl := ""
	if len(ChangedItems.added) > 0 {
		sl += "Added:\n"
		sl += strings.Join(ChangedItems.added, "\n")
	}
	if len(ChangedItems.removed) > 0 {
		if sl != "" {
			sl += "\n=============\n"
		}
		sl += "Removed:\n"
		sl += strings.Join(ChangedItems.removed, "\n")
		sl += "\n"
	}
	err := beeep.Notify("items Changed", sl, "")
	if err != nil {
		panic(err)
	}
	ChangedItems.added = nil
	ChangedItems.removed = nil
}

func IncId() {
	maxId := 1
	for {
		id <- maxId
		maxId++
	}
}

func newTemp(title, content string, endTime time.Time) (m Msg) {
	m.title = title
	m.content = content
	m.endTime = endTime
	m.from = "Org"
	m.cancel = make(chan bool)
	return m
}

func NewDefault(t string) (Msg, error) {
	var (
		m Msg
	)
	m.cancel = make(chan bool)
	m.from = "default"
	m.title = "timeup"
	duration, err := time.ParseDuration(t)
	if err != nil {
		return m, errors.New(fmt.Sprintf("can't parse %v to duration", t))
	}
	m.endTime = time.Now().Add(duration)
	// m.timer = time.NewTimer(duration)
	m.id = <-id
	msgMap[m.id] = m
	return m, nil
}

func (m Msg) toString() string {
	return fmt.Sprintf("============\n%d: %s\nend at: <%s>\n%s\n", m.id, m.title, m.endTime.Format("15:04 02.01.2006"), m.content)
}

func (m Msg) notify() {
	err := beeep.Notify(m.title, m.content, "")
	if err != nil {
		panic(err)
	}
	exec.Command("mplayer", "-endpos", "5", "/home/weiss/Music/soft_alarm_2.mp3").Output()
}

func (m Msg) Start() {
	if m.id == 0 {
		fmt.Println("cannot start Msg with id=0")
	} else {
		if m.endTime.After(time.Now()) {
			d := time.Until(m.endTime)
			m.timer = time.NewTimer(d)
			for {
				time.Sleep(1000 * time.Millisecond)
				select {
				case <-m.timer.C:
					m.notify()
					break
				case <-m.cancel:
					fmt.Printf("cancelled!\n")
					PrintAll()
					fmt.Print("OrgTimer>>>> ")
					break
				}
			}
		}
		delete(msgMap, m.id)
	}
}

func getOrgFileName() string {
	return fmt.Sprintf("Ʀd-%v.org", time.Now().Format("2006-01-02"))
}

func parseOrgTime(t string) time.Time {
	loc, _ := time.LoadLocation("Europe/Berlin")
	dateStr := timeReg.ReplaceAllString(t, `${date}`)
	timeStr := timeReg.ReplaceAllString(t, `${time}`)
	res, err := time.ParseInLocation("2006-01-02,15:04", dateStr+","+timeStr, loc)
	if err != nil {
		log.Fatal(err)
	}
	return res
}

func getNew() (map[int]Msg, bool) {
	f, err := os.Open(filepath.Join(orgFilePath, getOrgFileName()))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false
		}
		log.Fatal(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	s := bufio.NewScanner(f)
	var (
		newMsgMap = make(map[int]Msg)
		temp      string
		i         = 0
	)
	for s.Scan() {
		// temp is the previous line
		if timeReg.MatchString(s.Text()) && todoReg.MatchString(temp) {
			title := todoReg.ReplaceAllString(temp, `${title}`)
			m := newTemp(title, "", parseOrgTime(s.Text()))
			newMsgMap[i] = m
			i++
		}
		temp = s.Text()
	}
	err = s.Err()
	if err != nil {
		log.Fatal(err)
	}
	return newMsgMap, true
}

func merge(o map[int]Msg) {
	exist := false
	changed := false
	for i, x := range msgMap {
		if x.from == "Org" {
			for j, y := range o {
				if x.title == y.title && x.endTime.Equal(y.endTime) {
					delete(o, j)
					exist = true
					break
				}
			}
			if !exist {
				ChangedItems.removed = append(ChangedItems.removed, msgMap[i].title)
				delete(msgMap, i)
				exist = false
				changed = true
			}
		}
	}
	for _, x := range o {
		if x.endTime.After(time.Now()) {
			d := time.Until(x.endTime)
			changed = true
			ChangedItems.added = append(ChangedItems.added, x.title)
			x.timer = time.NewTimer(d)
			x.id = <-id
			msgMap[x.id] = x
			go x.Start()
		}
	}
	if changed {
		NotifyChanged()
	}
}

func Refresh() {
	m, exist := getNew()
	if exist {
		refreshMutex.Lock()
		merge(m)
		refreshMutex.Unlock()
	}
}

func AutoRefresh() {
	// c := time.Tick(5 * time.Minute)
	c := time.Tick(5 * time.Second)
	for range c {
		Refresh()
	}
}

func PrintAll() {
	if len(msgMap) == 0{
		fmt.Println("there is current no timer")
		return 
	}
	fmt.Println("\nall items:")
	// print map with the order of Msg.id
	l, i := len(msgMap), 0
	for l > 0 {
		if x, ok := msgMap[i]; ok {
			fmt.Printf(x.toString())
			l--
		}
		i++
	}
}
