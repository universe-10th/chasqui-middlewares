package protocols

import (
	"github.com/universe-10th/chasqui"
	"github.com/universe-10th/chasqui/types"
)

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
			onPanic(server, attendant, message, recovered)
		}
	}()
	if onMessage, exists := handlers[message.Command()]; exists {
		onMessage(server, attendant, message)
	} else {
		onUnknown(server, attendant, message)
	}
}
