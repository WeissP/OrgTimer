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
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

type Msg struct {
	title, content, img, from string
	timer                     *time.Timer
	id                        int
}

const (
	orgFilePath = "/home/weiss/Dropbox/Org-roam/daily/"
)

var (
	maxId   = 0
	msgMap  = make(map[int]Msg)
	idMutex sync.Mutex
	todoReg = regexp.MustCompile(``)
	timeReg = regexp.MustCompile(``)
)

func incId() int {
	idMutex.Lock()
	maxId++
	idMutex.Unlock()
	return maxId
}

func New(title string) (m Msg) {
	m.title = title
	return m
}

func NewDefault(t string) (Msg, error) {
	var (
		m Msg
	)
	m.from = "default"
	m.title = "timeup"
	duration, err := time.ParseDuration(t)
	if err != nil {
		return m, errors.New(fmt.Sprintf("can't parse %v to duration", t))
	}
	m.timer = time.NewTimer(duration)
	m.id = incId()
	msgMap[m.id] = m
	return m, nil
}

func (m Msg) toString() string {
	return fmt.Sprintf("============\n%d: %s\n%s\n", m.id, m.title, m.content)
}

func (m Msg) notify() {
	// notify.Alert(m.title, "notice", m.content,"")
	err := beeep.Notify(m.title, m.content, "")
	if err != nil {
		panic(err)
	}
	exec.Command("mplayer", "-endpos", "5", "/home/weiss/Music/soft_alarm_2.mp3").Output()
}

func (m Msg) Start() {
	<-m.timer.C
	m.notify()
	delete(msgMap, m.id)
}

func getOrgFileName() string {
	return fmt.Sprintf("Ʀd-%v.org", time.Now().Format("2006-01-02"))
}

// func getNew() (newMsg map[int]Msg) {
// 	f, err := os.Open(filepath.Join(orgFilePath, getOrgFileName()))
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer func() {
// 		if err = f.Close(); err != nil {
// 			log.Fatal(err)
// 		}
// 	}()
// 	s := bufio.NewScanner(f)
// 	for s.Scan() {
// 		if todoReg.MatchString(s.Text()) {
// 			todo := s.Text()
// 		} else {
// 			timeReg.MatchString(s.Text()){
// 				new(todo)
// 			}
// 		}
// 		fmt.Println(s.Text())
// 	}
// 	err = s.Err()
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

func PrintAll() {
	fmt.Println("all items:")
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
