package utils

import (
	"sort"
	"sync"

	"github.com/fatih/color"
)

type OutputMessage struct {
	Symbol  string
	Message string
	Color   color.Attribute
	Priority int
}

var (
	messages []OutputMessage
	mutex    sync.Mutex
)

// AddMessage adds a message to the output queue
func AddMessage(symbol string, message string, messageColor color.Attribute, priority int) {
	mutex.Lock()
	defer mutex.Unlock()
	messages = append(messages, OutputMessage{
		Symbol:  symbol,
		Message: message,
		Color:   messageColor,
		Priority: priority,
	})
}

// PrintMessages prints all collected messages sorted by priority
func PrintMessages() {
	mutex.Lock()
	defer mutex.Unlock()

	// Sort messages by priority (lower priority prints later)
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Priority > messages[j].Priority
	})

	for _, msg := range messages {
		PrintColouredMessage(msg.Symbol, msg.Message, msg.Color)
	}

	// Clear messages after printing
	messages = nil
}
