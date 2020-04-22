package main

import (
	"github.com/universe-10th/chasqui"
	"github.com/universe-10th/chasqui-protocols"
	"github.com/universe-10th/chasqui/types"
	"net"
)

type ChatProtocol struct {
	auth *AuthProtocol
}

func (protocol *ChatProtocol) Dependencies() protocols.Protocols {
	return protocols.Protocols{
		protocol.auth: true,
	}
}

func (protocol *ChatProtocol) Handlers() protocols.MessageHandlers {
	return protocols.MessageHandlers{
		"MSG": protocol.auth.AuthRequired(func(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message) {
			args := message.Args()
			kwArgs := message.KWArgs()
			if len(args) != 1 || len(kwArgs) != 0 {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_FORMAT", types.Args{"MSG", "Expected 1 positional (string) argument, and no keyword arguments"}, nil)
			} else if text, ok := args[0].(string); !ok {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_FORMAT", types.Args{"MSG", "The content must be a string"}, nil)
			} else {
				user, _ := attendant.Context("User")
				for _, attendant := range protocol.auth.serverLogins[server] {
					// noinspection GoUnhandledErrorResult
					attendant.Send("MSG_RECEIVED", types.Args{user.(User).nick, text}, nil)
				}
			}
		}),
		"PMSG": protocol.auth.AuthRequired(func(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message) {
			args := message.Args()
			kwArgs := message.KWArgs()
			if len(args) != 2 || len(kwArgs) != 0 {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_FORMAT", types.Args{"PMSG", "Expected 2 positional (string) arguments: user and content, and no keyword arguments"}, nil)
			} else if targetName, ok := args[0].(string); !ok {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_FORMAT", types.Args{"PMSG", "The target username must be a string"}, nil)
			} else if text, ok := args[1].(string); !ok {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_FORMAT", types.Args{"PMSG", "The content must be a string"}, nil)
			} else if attendant2, ok := protocol.auth.serverLogins[server][targetName]; !ok {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_TARGET", types.Args{"PMSG", "The target is not logged in"}, nil)
			} else {
				source, _ := attendant.Context("User")
				// noinspection GoUnhandledErrorResult
				attendant2.Send("MSG_RECEIVED", types.Args{source.(User).nick, text}, nil)
			}
		}),
	}
}

// Nothing needed in these, since the users management is in the auth protocol

func (protocol *ChatProtocol) Started(server *chasqui.Server, addr *net.TCPAddr) {}

func (protocol *ChatProtocol) AttendantStarted(server *chasqui.Server, attendant *chasqui.Attendant) {
}

func (protocol *ChatProtocol) AttendantStopped(server *chasqui.Server, attendant *chasqui.Attendant, stopType chasqui.AttendantStopType, err error) {
}

func (protocol *ChatProtocol) Stopped(server *chasqui.Server) {}
