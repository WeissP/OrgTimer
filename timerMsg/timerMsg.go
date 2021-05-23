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
	go dailyNotify()
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
	orgDailyFilePath = "/home/weiss/Dropbox/Org-roam/daily/"
	watchDays        = 1 // 0 is only today
	dailyNotifyHour  = 12
	dailyNotifyDays  = 2
)

var (
	extraOrgFiles = []string{"/home/weiss/Dropbox/Org-roam/orgzly/todo.org"}

	id           = make(chan int)
	msgMap       = make(map[int]*Msg)
	watchedFiles []string
	changedFiles []string
	ChangedItems changedItems
	refreshMutex sync.Mutex
	today        int
	todoReg      = regexp.MustCompile(`^\*+ TODO|INPROGRESS (?P<title>.+)`)
	timeReg      = regexp.MustCompile(`.*SCHEDULED: <(?P<date>\d{4}-\d{2}-\d{2}) [a-zA-Z]{2} (?P<time>\d{2}:\d{2})(\-\d{2}:\d{2})?( \+1w)?>`)

	red   = color.FgRed.Render
	green = color.FgGreen.Render
	blue  = color.FgLightBlue.Render
)

func CancelTimer(id int) {
	if x, ok := msgMap[id]; ok {
		x.disableTimer(true)
		x.cancel <- true
	} else {
		fmt.Printf("the id %v doesn't exist or is aleady disabled!\n", id)
	}
}

func (x *Msg) disableTimer(notify bool) {
	if notify {
		fmt.Printf("%v", x.toString())
		fmt.Println("disabled")
	}
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

func NewDefault(t string) (*Msg, error) {
	var (
		m Msg
	)
	m.cancel = make(chan bool)
	m.from = "default"
	m.title = "timeup"
	duration, err := time.ParseDuration(t)
	if err != nil {
		return &m, errors.New(fmt.Sprintf("can't parse %v to duration", t))
	}
	m.endTime = time.Now().Add(duration)
	m.id = <-id
	msgMap[m.id] = &m
	return &m, nil
}

func (m *Msg) toString() string {
	return fmt.Sprintf("============\n%s: %s\nend at: %s\nremain: %s\n%s\n", blue(m.id), m.title, green(m.endTime.Format("15:04 02.01.2006")), green(time.Until(m.endTime).Round(time.Second)), m.content)
}

func (m *Msg) test() {
	fmt.Printf("\nTested msg id: %v title: %v\n", m.id, m.title)
}

func (m *Msg) Start() {
	if m.id == 0 {
		fmt.Println("\ncannot start Msg with id=0{")
		fmt.Printf("%v", m.toString())
		fmt.Println("}")
	} else {
		if m.endTime.After(time.Now()) {
			d := time.Until(m.endTime)
			m.timer = time.NewTimer(d)
			for {
				time.Sleep(1000 * time.Millisecond)
				if m.endTime.Before(time.Now()) {
					m.disabled = true
					return
				}
				select {
				case <-m.timer.C:
					if !m.disabled {
						m.notify()
						m.disableTimer(false)
						fmt.Print("OrgTimer>>>> ")
					}
					fmt.Printf("%v", m.toString())
					return
				case <-m.cancel:
					m.timer.Stop()
					fmt.Printf(red("cancelled!\n"))
					PrintAll()
					m.disableTimer(true)
					fmt.Print("OrgTimer>>>> ")
					return
				}
			}
		}
	}
}

func getOrgDailyFileName(day int) string {
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

func scanFiles(l []string) (map[int]*Msg, bool) {
	var (
		newMsgMap = make(map[int]*Msg)
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
				title := todoReg.ReplaceAllString(temp, `${title}`)
				orgEndTime := parseOrgTime(s.Text())
				if orgEndTime.After(time.Now()) {
					found = true
					m := newTemp(title, "", orgEndTime)
					newMsgMap[index] = &m
					index++
				}
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

func merge(o map[int]*Msg) {
	exist := false
	changed := false
	for i, x := range msgMap {
		if x.from == "Org" {
			exist = false
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
				m.disableTimer(false)
				// fmt.Printf("deleted msgMap: %v", msgMap[i].toString())
				delete(msgMap, i)
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

func refreshMap(m map[int]*Msg) {
	refreshMutex.Lock()
	merge(m)
	refreshMutex.Unlock()
}

func getAllFilesByWatchDays() []string {
	var dailyFiles [watchDays + 1]string
	for i := 0; i <= watchDays; i++ {
		dailyFiles[i] = filepath.Join(orgDailyFilePath, getOrgDailyFileName(i))
	}
	allFiles := append(dailyFiles[:], extraOrgFiles...)
	return allFiles
}

func RefreshAllFiles() {
	m, found := scanFiles(getAllFilesByWatchDays())
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
	c := time.Tick(5 * time.Minute)
	// c := time.Tick(5 * time.Second)
	for range c {
		if changedFiles != nil {
			refreshFiles()
			changedFiles = nil
		}
	}
}

func watchNewFiles(ch chan bool) {
	c := time.Tick(5 * time.Second)
	for range c {
		for _, x := range getAllFilesByWatchDays() {
			contains := false
			for _, y := range watchedFiles {
				if x == y {
					contains = true
				}
			}
			if !contains {
				if _, err := os.Stat(x); err == nil {
					ch <- true
				}
			}
		}
	}
}

func AllChangedFiles() {
	fmt.Printf("%v", watchedFiles)
}

func watchFiles() {
	newFiles := make(chan bool)
	go watchNewFiles(newFiles)
	watcher := refreshWatcher()
	for {
		if today != time.Now().Day() {
			watcher.Close()
			today = time.Now().Day()
			watcher = refreshWatcher()
		}
		select {
		case <-newFiles:
			watcher = refreshWatcher()
			RefreshAllFiles()
		case event, eventOk := <-watcher.Events:
			if !eventOk {
				log.Println("error")
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				// fmt.Printf("Name: %v", event.Name)
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
	watchedFiles = nil
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("ERROR", err)
	}

	for _, x := range extraOrgFiles {
		if err := watcher.Add(x); err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("%v not exists!", x)
				continue
			}
			fmt.Println("ERROR", err)
		} else {
			watchedFiles = append(watchedFiles, extraOrgFiles...)
		}
	}

	for i := 0; i <= watchDays; i++ {
		if err := watcher.Add(filepath.Join(orgDailyFilePath, getOrgDailyFileName(i))); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Println("ERROR", err)
		}
		watchedFiles = append(watchedFiles, filepath.Join(orgDailyFilePath, getOrgDailyFileName(i)))
	}
	return watcher
}

func getActivateMsgMap() map[int]*Msg {
	activateMsgMap := make(map[int]*Msg)
	for _, x := range msgMap {
		if !x.disabled {
			activateMsgMap[x.id] = x
		}
	}
	return activateMsgMap
}

func PrintAll() {
	activateMsgMap := getActivateMsgMap()
	if len(activateMsgMap) == 0 {
		fmt.Println(red("there is current no timer"))
		return
	}
	fmt.Println("\nall items:")
	var s []int
	for _, x := range activateMsgMap {
		// fmt.Printf("\nmap in activateMsgMap: %v", x.toString())
		s = sortedInsert(s, x.id)
		// fmt.Printf("////s: %v", s)
	}
	for i := len(s) - 1; i >= 0; i-- {
		fmt.Printf(msgMap[s[i]].toString())
	}
}

func isEarlier(idA, idB int) bool {
	return msgMap[idA].endTime.Before(msgMap[idB].endTime)
}

func sortedInsert(s []int, n int) []int {
	if len(s) == 0 {
		s = append(s, n)
	} else {
		for i, x := range s {
			if isEarlier(n, x) {
				// fmt.Printf("\n%v is earlier than %v", n, x)
				s = append(s[:i+1], s[i:]...)
				s[i] = n
				break
			}
			if i == len(s)-1 {
				// fmt.Println("////len")
				s = append(s, n)
				break
			}
		}
	}
	return s
}

func dailyNotify() {
	for {
		var (
			tNow    = time.Now()
			y, m, d = tNow.Date()
			tNotify = time.Date(y, m, d, dailyNotifyHour, 0, 0, 0, tNow.Location())
		)
		// fmt.Printf("timedTimer Before add Date:%v", tNotify.Format("15:04 02.01.2006"))
		if tNotify.Before(tNow) {
			tNotify = tNotify.AddDate(0, 0, 1)
		}
		fmt.Printf("timedTimer:%v", tNotify.Format("15:04 02.01.2006"))
		timedTimer := time.NewTimer(time.Until(tNotify))
		<-timedTimer.C
		NotifyNextFewDays(dailyNotifyDays)
	}
}

func NotifyAll() {
	for _, x := range msgMap {
		if !x.disabled {
			beeep.Notify(x.title, x.content, "")
		}
	}
}

func NotifyNextFewDays(days int) {
	var (
		tNow      = time.Now()
		y, m, d   = tNow.Date()
		timePoint = time.Date(y, m, d, 23, 59, 59, 0, tNow.Location()).AddDate(0, 0, days)
	)
	for _, x := range msgMap {
		if !x.disabled && x.endTime.Before(timePoint) {
			beeep.Notify(x.title, x.content, "")
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
