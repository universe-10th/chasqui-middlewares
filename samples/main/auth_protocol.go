package main

import (
	"fmt"
	"github.com/universe-10th/chasqui"
	"github.com/universe-10th/chasqui-protocols"
	"github.com/universe-10th/chasqui/types"
	"net"
)

type User struct {
	nick     string
	role     string
	password string
}

var Users = map[string]User{
	"pepe": {
		"pepe", "user", "pepe$123",
	},
	"toto": {
		"toto", "user", "toto$123",
	},
	"carlos": {
		"carlos", "admin", "carlos$123",
	},
}

type AuthProtocol struct {
	serverConns  map[*chasqui.Server]map[*chasqui.Attendant]bool
	serverLogins map[*chasqui.Server]map[string]*chasqui.Attendant
}

func (protocol *AuthProtocol) Dependencies() protocols.Protocols {
	return nil
}

func (protocol *AuthProtocol) Handlers() protocols.MessageHandlers {
	return protocols.MessageHandlers{
		"LOGOUT": func(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message) {
			args := message.Args()
			kwArgs := message.KWArgs()
			if len(args) != 0 || len(kwArgs) != 0 {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_FORMAT", types.Args{"LOGOUT", "No arguments expected"}, nil)
			} else if _, ok := attendant.Context("User"); !ok {
				// noinspection GoUnhandledErrorResult
				attendant.Send("ALREADY_LOGGED_OUT", nil, nil)
			} else {
				// noinspection GoUnhandledErrorResult
				attendant.Send("LOGGED_OUT", nil, nil)
				ctx, _ := attendant.Context("User")
				attendant.RemoveContext("User")
				delete(protocol.serverLogins[server], ctx.(User).nick)
			}
		},
		"LOGIN": func(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message) {
			args := message.Args()
			kwArgs := message.KWArgs()
			if len(args) != 2 || len(kwArgs) != 0 {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_FORMAT", types.Args{"LOGIN", "Expected 2 positional arguments: username, password. No keyword arguments expected"}, nil)
			} else if userName, ok := args[0].(string); !ok {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_FORMAT", types.Args{"LOGIN", "First argument (username) must be a string"}, nil)
			} else if password, ok := args[1].(string); !ok {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_FORMAT", types.Args{"LOGIN", "Second argument (password) must be a string"}, nil)
			} else if user, ok := Users[userName]; !ok {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_USER", types.Args{userName}, nil)
			} else if password != user.password {
				// noinspection GoUnhandledErrorResult
				attendant.Send("INVALID_PASSWORD", nil, nil)
			} else {
				attendant.SetContext("User", user)
				if attendant2, ok := protocol.serverLogins[server][user.nick]; ok {
					// noinspection GoUnhandledErrorResult
					attendant2.Send("GHOSTED", nil, nil)
					attendant2.RemoveContext("User")
				}
				protocol.serverLogins[server][user.nick] = attendant
				// noinspection GoUnhandledErrorResult
				attendant.Send("OK", nil, nil)
			}
		},
	}
}

func (protocol *AuthProtocol) Started(server *chasqui.Server, addr *net.TCPAddr) {
	protocol.serverConns[server] = make(map[*chasqui.Attendant]bool)
	protocol.serverLogins[server] = make(map[string]*chasqui.Attendant)
	fmt.Println("Auth started for server:", server, addr)
}

func (protocol *AuthProtocol) AttendantStarted(server *chasqui.Server, attendant *chasqui.Attendant) {
	protocol.serverConns[server][attendant] = true
	fmt.Println("Auth started for server and socket:", server, attendant)
}

func (protocol *AuthProtocol) AttendantStopped(server *chasqui.Server, attendant *chasqui.Attendant, stopType chasqui.AttendantStopType, err error) {
	if protocol.serverLogins != nil {
		if serverLogins, ok := protocol.serverLogins[server]; ok {
			if user, ok := attendant.Context("User"); ok {
				delete(serverLogins, user.(User).nick)
			}
		}
	}
	if protocol.serverConns != nil {
		delete(protocol.serverConns[server], attendant)
	}
	fmt.Println("Auth stopped for server and socket:", server, attendant, stopType, err)
}

func (protocol *AuthProtocol) Stopped(server *chasqui.Server) {
	delete(protocol.serverConns, server)
	delete(protocol.serverLogins, server)
	fmt.Println("Chat stopped for server:", server)
}

func (protocol *AuthProtocol) AuthRequired(handler protocols.MessageHandler) protocols.MessageHandler {
	return func(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message) {
		if _, ok := attendant.Context("User"); ok {
			handler(server, attendant, message)
		} else {
			// noinspection GoUnhandledErrorResult
			attendant.Send("LOGIN_REQUIRED", nil, nil)
		}
	}
}
