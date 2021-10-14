package connector

import (
	"OrgTimer/msg"
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbPath = "/home/weiss/.emacs.d/org-roam.db"
	// 2021-05-14T14:40:00+0200
	timeFormat = `"2006-01-02T15:04:05-0700"`
)

func getConnector() *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func parseTime(s string) time.Time {
	t, err := time.Parse(timeFormat, s)
	if err != nil {
		panic(err)
	}
	return t
}

func GetSchedule() (res msg.MsgList) {
	db := getConnector()
	query := "select nodes.title, nodes.scheduled from nodes where nodes.scheduled is not null"
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var title string
		var scheduled string
		err = rows.Scan(&title, &scheduled)
		if err != nil {
			log.Fatal(err)
		}
		msg := msg.NewOrgMsg(title, "", parseTime(scheduled))
		res = append(res, &msg)
	}
	return res
}
