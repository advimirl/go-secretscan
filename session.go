package main

import (
	"encoding/hex"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Session struct {
	filePath string
	sessName string
	memStorage map[string]bool
	lock sync.Mutex
}



func createSession(folder string, token AccessToken) *Session {
	hash := sha256_encode(token.Token + token.URL + "_" + token.WorkerType)
	sessionName := hex.EncodeToString(hash)
	filePath := filepath.Join(folder, "." + sessionName[:6])
	return &Session{filePath: filePath,
		sessName: sessionName,
		memStorage: make(map[string]bool),
	}
}

func (s *Session) exists() bool {
	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func (s *Session) init(isForced bool) {
	if s.exists() && !isForced {
		rawData, err := ioutil.ReadFile(s.filePath)
		if err != nil {
			logrus.Error(err)
		}
		for _, line := range strings.Split(string(rawData), "\n") {
			if len(line) < 10 {
				continue
			}
			s.memStorage[strings.TrimSpace(line)] = true
		}
	} else if s.exists() && isForced {
		err := os.Remove(s.filePath)
		if err != nil {
			logrus.Panic(err)
		}
	}
}

func (s *Session) check(repoName string) bool {
	hash := sha256_encode(repoName)
	repoHash := hex.EncodeToString(hash)[:24]
	if _, ok := s.memStorage[repoHash]; !ok {
		return false
	} else {
		return true
	}
}

func (s *Session) _appendToFile(repoHash string) {
	s.lock.Lock()
	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
	if err != nil {
		logrus.Panic(err)
	}
	defer func (){
		err = f.Close()
		if err != nil {
			logrus.Panic(err)
		}
	}()

	_, err = f.WriteString(repoHash + "\n")
	if err != nil {
		logrus.Panic(err)
	}
	s.lock.Unlock()
}

func (s* Session) add(repoName string) {
	if s.check(repoName) {
		return
	}
	hash := sha256_encode(repoName)
	repoHash := hex.EncodeToString(hash)[:24]
	s.memStorage[repoHash] = true
	s._appendToFile(repoHash)
}