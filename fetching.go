package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

var dbPath string = CreateDatabasePath()

func CreateDatabasePath() string {
	path, exists := os.LookupEnv("XDG_DATA_HOME")
	if exists {
		return path + "/scheduler/schedule.json"
	} else {
		home := os.Getenv("HOME")

		return home + "/.schedule.json"
	}
}

func ReadJson() []Stop {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("Missing database file, fetching it from set FTP server")
		FTPFetch(ReadFTPCred("my.cred"))
	}

	b, err := ioutil.ReadFile(dbPath)
	if err != nil {
		panic(err)
	}

	var stops []Stop
	json.Unmarshal(b, &stops)
	return stops
}

func FTPFetch(host, username, password string) {
	c, err := ftp.Dial(host + ":21", ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		panic(err)
	}

	err = c.Login(username, password)
	if err != nil {
		panic(err)
	}

	r, err := c.Retr("schedule.json.gz")
	if err != nil {
		panic(err)
	}
	defer r.Close()

	reader, err := gzip.NewReader(r)
	p, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(dbPath, os.O_RDWR | os.O_CREATE, 0755)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	f.Write(p)

	if err := c.Quit(); err != nil {
		panic(err)
	}
}

func ReadFTPCred(path string) (host, user, pass string) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	parts := strings.Split(string(b), ";")
	if len(parts) != 3 {
		panic("Garbage in ftp credentials")
	}

	host = parts[0]
	user = parts[1]
	pass = parts[2]

	return
}
