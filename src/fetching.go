package scheduler

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"net/http"
	"strings"
	"time"
	"bytes"

	"github.com/jlaffaye/ftp"
)

type Times struct {
	Hours []string `json:"hour"`
	WorkMins []string `json:"work"`
	SaturdayMins []string `json:"saturday"`
	HolidayMins []string `json:"holiday"`
}

type Stop struct {
	Id int `json:"id"`
	LineNr int `json:"line"`
	Direction string `json:"direction"`
	Name string `json:"stop_name"`
	Times Times `json:"times"`
}

type Database struct {
	Stops []Stop
	Complete bool
}

func NewDatabase() Database {
	return Database {
		Complete: false,
	}
}

func CreateDatabasePath() string {
	path, exists := os.LookupEnv("XDG_DATA_HOME")
	if exists {
		return path + "/scheduler/schedule.json"
	} else {
		home := os.Getenv("HOME")

		return home + "/.schedule.json"
	}
}

func DatabaseFromJSON() (result *Database) {
	result = new(Database)
	dbPath := CreateDatabasePath()
	
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("Missing database file, fetching it from the web")
		err := DatabaseFromWeb(dbPath)
		if err != nil {
			panic(err)
		}
	}

	b, err := ioutil.ReadFile(dbPath)
	if err != nil {
		panic(err)
	}

	go ConcurJSONDec(result, bytes.NewReader(b))

	for len(result.Stops) < 100 {
	 	time.Sleep(time.Millisecond)
	}
	
	return 
}

func RefreshJson(db *Database) {
	dbPath := CreateDatabasePath()
	
	*db = NewDatabase()
	err := DatabaseFromWeb(dbPath)
	if err != nil {
		panic(err)
	}

	b, err := ioutil.ReadFile(dbPath)
	if err != nil {
		panic(err)
	}

	go ConcurJSONDec(db, bytes.NewReader(b))
	
	for len(db.Stops) < 100 {
		time.Sleep(time.Millisecond)
	}
}	


func ConcurJSONDec(db *Database, reader io.Reader) {
	dec := json.NewDecoder(reader)
	_, err := dec.Token()
	if err != nil {
		panic(err)
	}
		
	for dec.More() {
		var s Stop
		err := dec.Decode(&s)
		if err != nil {
			panic(err)
		}

		db.Stops = append(db.Stops, s)
	}

	db.Complete = true
	
	_, err = dec.Token()
	if err != nil {
		panic(err)
	}

	db.Stops = TimesToOneDay(db.Stops)
}

func DatabaseFromWeb(dbPath string) error {
	r, err := http.Get("https://mradomski.top/scheduler/latest.json.gz")
	if err != nil {
		return err
	}
	defer r.Body.Close()

	gzReader, err := gzip.NewReader(r.Body)
	content, err := ioutil.ReadAll(gzReader)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(dbPath, os.O_RDWR | os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(content)
	return nil
}

func FetchFTP(host, username, password string) (b *ftp.Response, err error) {
	c, err := ftp.Dial(host + ":21", ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}

	err = c.Login(username, password)
	if err != nil {
		return nil, err
	}

	r, err := c.Retr("schedule.json.gz")
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if err := c.Quit(); err != nil {
		panic(err)
		return r, err
	}

	return r, nil
}

func DatabaseFromFTP(dbPath, host, username, password string) (e error) {
	r, err := FetchFTP(host, username, password)
	if err != nil {
		return err
	}

	// The response is gzip compressed
	reader, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	
	p, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(dbPath, os.O_RDWR | os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write(p)
	return
}

func ReadFTPCred(path string) (host, user, pass string, e error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		e = err
		return 
	}

	parts := strings.Split(string(b), ";")
	if len(parts) != 3 {
		e = errors.New("Garbage in ftp credentials")
		return
	}

	host = parts[0]
	user = parts[1]
	pass = parts[2]

	return
}
