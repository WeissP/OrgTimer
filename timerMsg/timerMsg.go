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

	"github.com/fsnotify/fsnotify"
	"github.com/gen2brain/beeep"
	"github.com/gookit/color"
)

func init() {
	today = time.Now().Day()
	go IncId()

	RefreshAllFiles()
	go watchFiles()
	go AutoRefresh()
}

type Msg struct {
	title, content, img, from string
	endTime                   time.Time
	timer                     *time.Timer
	id                        int
	cancel                    chan bool
	disabled                  bool
}

type changedItems struct {
	added, removed []string
}

const (
	orgFilePath = "/home/weiss/Dropbox/Org-roam/daily/"
	watchDays   = 1 // 0 is only today
)

var (
	id           = make(chan int)
	msgMap       = make(map[int]*Msg)
	changedFiles []string
	ChangedItems changedItems
	refreshMutex sync.Mutex
	today        int
	todoReg      = regexp.MustCompile(`^\*+ TODO (?P<title>.+)`)
	timeReg      = regexp.MustCompile(`.*SCHEDULED: <(?P<date>\d{4}-\d{2}-\d{2}) [a-zA-Z]{2} (?P<time>\d{2}:\d{2})>`)

	red   = color.FgRed.Render
	green = color.FgGreen.Render
	blue  = color.FgLightBlue.Render
)

func CancelTimer(id int) {
	if x, ok := msgMap[id]; ok {
		x.disableTimer()
		x.cancel <- true
	} else {
		fmt.Printf("the id %v doesn't exist or is aleady disabled!\n", id)
	}
}

func (x *Msg) disableTimer() {
	x.disabled = true
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
	msgMap[m.id] = &m
	return m, nil
}

func (m *Msg) toString() string {
	return fmt.Sprintf("============\n%s: %s\nend at: %s\nremain: %s\n%s\n", blue(m.id), m.title, green(m.endTime.Format("15:04 02.01.2006")), green(time.Until(m.endTime).Round(time.Second)), m.content)
}

func (m *Msg) Start() {
	if m.id == 0 {
		fmt.Println("cannot start Msg with id=0")
	} else {
		if m.endTime.After(time.Now()) {
			d := time.Until(m.endTime)
			m.timer = time.NewTimer(d)
		loop:
			for {
				time.Sleep(1000 * time.Millisecond)
				select {
				case <-m.timer.C:
					if !m.disabled {
						m.notify()
					}
					break loop
				case <-m.cancel:
					fmt.Printf(red("cancelled!\n"))
					PrintAll()
					fmt.Print("OrgTimer>>>> ")
					break loop
				}
			}
		}
		m.disableTimer()
	}
}

func getOrgFileName(day int) string {
	return fmt.Sprintf("Ʀd-%v.org", time.Now().AddDate(0, 0, day).Format("2006-01-02"))
}

func parseOrgTime(t string) time.Time {
	loc := time.Now().Location()
	dateStr := timeReg.ReplaceAllString(t, `${date}`)
	timeStr := timeReg.ReplaceAllString(t, `${time}`)
	res, err := time.ParseInLocation("2006-01-02,15:04", dateStr+","+timeStr, loc)
	if err != nil {
		log.Fatal(err)
	}
	return res
}

func scanFiles(l []string) (map[int]Msg, bool) {
	var (
		newMsgMap = make(map[int]Msg)
		temp      string
		index     = 0
		found     = false
	)
	for _, x := range l {
		f, err := os.Open(x)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			log.Fatal(err)
		}
		defer func() {
			if err = f.Close(); err != nil {
				log.Fatal(err)
			}
		}()
		s := bufio.NewScanner(f)
		for s.Scan() {
			// temp is the previous line
			if timeReg.MatchString(s.Text()) && todoReg.MatchString(temp) {
				found = true
				title := todoReg.ReplaceAllString(temp, `${title}`)
				m := newTemp(title, "", parseOrgTime(s.Text()))
				newMsgMap[index] = m
				index++
			}
			temp = s.Text()
		}
		err = s.Err()
		if err != nil {
			log.Fatal(err)
		}
	}
	return newMsgMap, found
}

func getNew(day int) (map[int]Msg, bool) {
	var (
		newMsgMap = make(map[int]Msg)
		temp      string
		index     = 0
	)
	for i := 0; i <= day; i++ {
		f, err := os.Open(filepath.Join(orgFilePath, getOrgFileName(i)))
		if err != nil {
			if os.IsNotExist(err) {
				if i == 0 {
					return nil, false
				} else {
					continue
				}
			}
			log.Fatal(err)
		}
		defer func() {
			if err = f.Close(); err != nil {
				log.Fatal(err)
			}
		}()
		s := bufio.NewScanner(f)
		for s.Scan() {
			// temp is the previous line
			if timeReg.MatchString(s.Text()) && todoReg.MatchString(temp) {
				title := todoReg.ReplaceAllString(temp, `${title}`)
				m := newTemp(title, "", parseOrgTime(s.Text()))
				newMsgMap[i] = m
				index++
			}
			temp = s.Text()
		}
		err = s.Err()
		if err != nil {
			log.Fatal(err)
		}
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
				m := msgMap[i]
				m.disableTimer()
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
			msgMap[x.id] = &x
			go x.Start()
		}
	}
	if changed {
		NotifyChanged()
	}
}

func refreshMap(m map[int]Msg) {
	refreshMutex.Lock()
	merge(m)
	refreshMutex.Unlock()
}

func RefreshAllFiles() {
	var allFiles [watchDays + 1]string
	for i := 0; i <= watchDays; i++ {
		fmt.Printf("%v", getOrgFileName(i))
		allFiles[i] = filepath.Join(orgFilePath, getOrgFileName(i))
	}
	m, found := scanFiles(allFiles[:])
	if found {
		refreshMap(m)
	}
}

func refreshFiles() {
	m, found := scanFiles(changedFiles)
	if found {
		refreshMap(m)
	}
}

func AutoRefresh() {
	// c := time.Tick(5 * time.Minute)
	c := time.Tick(5 * time.Second)
	for range c {
		if changedFiles != nil {
			refreshFiles()
			changedFiles = nil
		}
	}
}

func watchFiles() {
	watcher := refreshWatcher()
	for {
		if today != time.Now().Day() {
			watcher.Close()
			today = time.Now().Day()
			watcher = refreshWatcher()
		}
		select {
		case event, eventOk := <-watcher.Events:
			if !eventOk {
				log.Println("error")
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				contains := false
				for _, x := range changedFiles {
					if x == event.Name {
						contains = true
					}
				}
				if !contains {
					changedFiles = append(changedFiles, event.Name)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}

	}
}

func refreshWatcher() *fsnotify.Watcher {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("ERROR", err)
	}

	for i := 0; i <= watchDays; i++ {
		if err := watcher.Add(filepath.Join(orgFilePath, getOrgFileName(i))); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Println("ERROR", err)
		}
	}
	return watcher
}

func PrintAll() {
	if len(msgMap) == 0 {
		fmt.Println(red("there is current no timer"))
		return
	}
	fmt.Println("\nall items:")
	for _, x := range msgMap {
		if !x.disabled {
			fmt.Printf(x.toString())
		}
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

func (m Msg) notify() {
	err := beeep.Notify(m.title, m.content, "")
	if err != nil {
		panic(err)
	}
	exec.Command("mplayer", "-endpos", "5", "/home/weiss/Music/soft_alarm_2.mp3").Output()
}
