package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"syscall"

	"github.com/jessevdk/go-flags"
)

type session struct{}

func (session) Execute(args []string) error {
	resp, err := http.Get("http://localhost:8080/session")
	if err != nil {
		return err
	}
	io.Copy(os.Stdout, resp.Body)
	return nil
}

type track struct{}

func (track) Execute(args []string) error {
	historyRaw, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	history := strings.TrimSpace(strings.SplitN(strings.TrimSpace(string(historyRaw)), " ", 2)[1])
	sessionFileName := os.Getenv("HISTORY_SESSION")
	if sessionFileName == "" {
		return nil
	}
	sessionFile, err := os.OpenFile(sessionFileName, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	err = syscall.Flock(int(sessionFile.Fd()), syscall.LOCK_EX)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(sessionFile)
	if err != nil {
		return err
	}
	parts := strings.Split(string(data), "\000")
	session := ""
	upload := append(parts[1:], history)

	if len(parts) == 0 || parts[0] == "" {
		resp, err := http.Get("http://localhost:8080/session")
		if err != nil {
			sessionFile.Seek(0, io.SeekStart)
			sessionFile.Truncate(0)
			sessionFile.WriteString(strings.Join(append(parts, history), "\000"))
			return nil
		}
		sessionRaw, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		session = string(sessionRaw)
		if err != nil {
			sessionFile.Seek(0, io.SeekStart)
			sessionFile.Truncate(0)
			sessionFile.WriteString(strings.Join(append(parts, history), "\000"))
			return nil
		}
	} else {
		session = parts[0]
	}

	for i, p := range upload {
		_, err = http.PostForm("http://localhost:8080/log", url.Values{"command": {p}, "session": {session}})
		if err != nil {
			sessionFile.Seek(0, io.SeekStart)
			sessionFile.Truncate(0)
			sessionFile.WriteString(strings.Join(append([]string{session}, upload[i:]...), "\000"))
			return err
		}
	}
	if len(upload) > 1 {
		sessionFile.Seek(0, io.SeekStart)
		sessionFile.Truncate(0)
		sessionFile.WriteString(session)
	}
	return nil
}

type last struct {
	Count int `short:"c" long:"count" description:"Number of commands to return" default:"10000"`
}

func (l last) Execute(args []string) error {
	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/last?count=%d", l.Count))
	if err != nil {
		return err
	}
	io.Copy(os.Stdout, resp.Body)
	return nil
}

func main() {
	var options struct {
		Session session `command:"session" description:"Start a new session and return the session number"`
		Track   track   `command:"track" description:"Record a command to the session's log"`
		Last    last    `command:"last" description:"Return the last N commands"`
	}
	p := flags.NewParser(&options, flags.Default)
	_, err := p.Parse()
	if err != nil {
		if err == flags.ErrHelp {
			os.Exit(0)
		}
		log.Fatal(err)
	}
}
