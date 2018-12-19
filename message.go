package sse

import (
	"bytes"
	"fmt"
	"strings"
)

// Message represents a event source message.
type Message struct {
	id,
	data,
	event string
	retry   int
	version int
}

func SimpleMessage(data string) *Message {
	return NewMessage("", data, "")
}

func SimpleMessageVer(data string, version int) *Message {
	return NewMessageVer("", data, "", version)
}

func NewMessage(id, data, event string) *Message {
	return &Message{
		id,
		data,
		event,
		0,
		0,
	}
}

func NewMessageVer(id, data, event string, version int) *Message {
	return &Message{
		id,
		data,
		event,
		0,
		version,
	}
}

func (m *Message) String() string {
	var buffer bytes.Buffer

	if len(m.id) > 0 {
		buffer.WriteString(fmt.Sprintf("id: %s\n", m.id))
	}

	if m.retry > 0 {
		buffer.WriteString(fmt.Sprintf("retry: %d\n", m.retry))
	}

	if len(m.event) > 0 {
		buffer.WriteString(fmt.Sprintf("event: %s\n", m.event))
	}

	if len(m.data) > 0 {
		buffer.WriteString(fmt.Sprintf("data: %s\n", strings.Replace(m.data, "\n", "\ndata: ", -1)))
	}

	buffer.WriteString("\n")

	return buffer.String()
}
