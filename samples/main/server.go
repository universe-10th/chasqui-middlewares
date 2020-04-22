package main

import (
	"github.com/universe-10th/chasqui"
	"github.com/universe-10th/chasqui-protocols"
	"github.com/universe-10th/chasqui/marshalers/json"
)

var auth = &AuthProtocol{
	serverConns:  make(map[*chasqui.Server]map[*chasqui.Attendant]bool),
	serverLogins: make(map[*chasqui.Server]map[string]*chasqui.Attendant),
}
var chat = &ChatProtocol{auth}
var funnel, _ = protocols.NewProtocolsFunnel([]protocols.Protocol{chat, auth})

func MakeServer() *chasqui.Server {
	return chasqui.NewServer(
		&json.JSONMessageMarshaler{}, 1024, 1, 0,
	)
}

func funnelServer(server *chasqui.Server) {
	chasqui.FunnelServerWith(server, funnel)
}
