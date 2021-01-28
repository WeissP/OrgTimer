package timerMsg

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/gen2brain/beeep"
)

type Msg struct {
	title, content, img, from string
	endTime                   time.Time
	timer                     *time.Timer
	id                        int
}

const (
	orgFilePath = "/home/weiss/Dropbox/Org-roam/daily/"
)

var (
	id      = make(chan int)
	msgMap  = make(map[int]Msg)
	idMutex sync.Mutex
	todoReg = regexp.MustCompile(``)
	timeReg = regexp.MustCompile(``)
)

func IncId() {
	maxId := 0
	for {
		id <- maxId
		maxId++
	}
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
	m.endTime = time.Now().Add(duration)
	// m.timer = time.NewTimer(duration)
	m.id = <-id
	msgMap[m.id] = m
	return m, nil
}

func (m Msg) toString() string {
	return fmt.Sprintf("============\n%d: %s\nend at: %s\n%s\n", m.id, m.title, m.endTime.Format("15:04 02.01.2006"), m.content)
}

func (m Msg) notify() {
	// notify.Alert(m.title, "notice", m.content,"")
	err := beeep.Notify(m.title, m.content, "")
	if err != nil {
		panic(err)
	}
	// exec.Command("mplayer", "-endpos", "5", "/home/weiss/Music/soft_alarm_2.mp3").Output()
}

func (m Msg) Start() {
	d := m.endTime.Sub(time.Now())
	m.timer = time.NewTimer(d)
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
