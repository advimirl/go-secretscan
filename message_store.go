package main

import "github.com/sirupsen/logrus"

type MessageStore struct {
	store map[string][]*Message
}

func (m *MessageStore) getMessages(projectName string) []*Message {
	logrus.Debugf("get messages for project %s", projectName)
	if _, ok := m.store[projectName]; !ok {
		m.store[projectName] = make([]*Message, 0)
	}
	return m.store[projectName]
}

func newMessageStore() *MessageStore {
	return &MessageStore{
		make(map[string][]*Message),
	}
}
