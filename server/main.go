package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.etcd.io/bbolt"
)

func main() {
	s, err := New("history.db")
	if err != nil {
		panic(err)
	}
	http.HandleFunc("/log", s.Log)
	http.HandleFunc("/session", s.Session)
	http.HandleFunc("/last", s.Last)
	http.ListenAndServe(":8080", nil)
}

type Server struct {
	db *bbolt.DB
}

func New(path string) (*Server, error) {
	db, err := bbolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &Server{db}, nil
}

func itob(v int64) []byte {
	return []byte{
		byte(v >> 56),
		byte(v >> 48),
		byte(v >> 40),
		byte(v >> 32),
		byte(v >> 24),
		byte(v >> 16),
		byte(v >> 8),
		byte(v),
	}
}

// Log records a command to the server's database
func (s *Server) Log(rw http.ResponseWriter, req *http.Request) {
	command := req.FormValue("command")
	session, _ := strconv.Atoi(req.FormValue("session"))
	log.Printf("%d: %s", session, command)
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("commands"))
		if err != nil {
			return err
		}
		ts := time.Now().UnixNano()
		data := itob(int64(session))
		data = append(data, []byte(command)...)
		return b.Put(itob(ts), data)
	})
	if err != nil {
		http.Error(rw, "error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) Session(rw http.ResponseWriter, req *http.Request) {
	var seq int64
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("sessions"))
		if err != nil {
			return err
		}
		s, err := b.NextSequence()
		if err != nil {
			return err
		}
		seq = int64(s)
		ts := time.Now().UnixNano()
		data := itob(ts)
		source := strings.Split(req.RemoteAddr, ":")[0]
		data = append(data, []byte(source)...)
		return b.Put(itob(seq), data)
	})
	if err != nil {
		http.Error(rw, "error", http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(rw, "%d", seq)
}

func (s *Server) Last(rw http.ResponseWriter, req *http.Request) {
	count, _ := strconv.Atoi(req.FormValue("count"))
	if count == 0 {
		count = 10000
	}
	session, _ := strconv.Atoi(req.FormValue("session"))

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("commands"))
		c := b.Cursor()
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			if session != 0 && uint64(session) != binary.BigEndian.Uint64(v[:8]) {
				continue
			}
			fmt.Fprintf(rw, "%s\000", v[8:])
			count--
			if count == 0 {
				break
			}
		}
		return nil
	})
	if err != nil {
		http.Error(rw, "error", http.StatusInternalServerError)
		return
	}
}
