package protocols

import (
	"errors"
	"fmt"
	"github.com/universe-10th/chasqui"
	"github.com/universe-10th/chasqui/types"
)

var ErrHandlerConflict = errors.New("a handler is already registered with that key")

// Handling a message involves the server, the
// involved attendant, and the received message.
type MessageHandler func(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message)

// Handling a panic is a guard for the eventual
// case of a panic occurring inside a message
// handler and we don't want the server to crash.
// Panic handlers are not meant to be left to the
// protocol maker, but to the server designer and
// administrator, and should be designed carefully
// to not panic themselves.
type MessagePanicHandler func(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message, recovered interface{})

// Handlers are a map of string -> handler with a
// convenience function to safely handle a command.
// Panics and unknown commands are captured and
// reported appropriately.
type MessageHandlers map[string]MessageHandler

// Handles a received message, the possibility of
// it being unknown, and capturing any panic it may
// occur inside.
func (handlers MessageHandlers) Handle(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message,
	onUnknown MessageHandler, onPanic MessagePanicHandler) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if onPanic != nil {
				onPanic(server, attendant, message, recovered)
			}
			fmt.Println("Panic on handle!", recovered)
		}
	}()
	if onMessage, exists := handlers[message.Command()]; exists {
		onMessage(server, attendant, message)
	} else if onUnknown != nil {
		onUnknown(server, attendant, message)
	}
}

// Attempts a merge. It only merges if all of the keys in the other
// handler are unused in the source handler.
func (handlers MessageHandlers) Merge(otherHandlers MessageHandlers) error {
	for key, other := range otherHandlers {
		if current, ok := handlers[key]; ok && current != nil && other != nil {
			return ErrHandlerConflict
		}
	}
	for key, other := range otherHandlers {
		if other != nil {
			handlers[key] = other
		}
	}
	return nil
}
