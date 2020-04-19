package protocols

import (
	"github.com/universe-10th/chasqui"
	"github.com/universe-10th/chasqui/types"
	"net"
	"time"
)

type ProtocolsFunnel struct {
	flattened               []Protocol
	handlers                MessageHandlers
	serverLoadProgress      map[*chasqui.Server]int
	attendantLoadProgress   map[*chasqui.Attendant]int
	onStartedPanic          func(*chasqui.Server, *net.TCPAddr, Protocol, interface{})
	onAttendantStartedPanic func(*chasqui.Server, *chasqui.Attendant, Protocol, interface{})
	onAcceptFailed          func(*chasqui.Server, error)
	onMessageUnknown        MessageHandler
	onMessagePanic          MessagePanicHandler
	onThrottled             func(*chasqui.Server, *chasqui.Attendant, types.Message, time.Time, time.Duration)
	onAttendantStoppedPanic func(*chasqui.Server, *chasqui.Attendant, chasqui.AttendantStopType, error, Protocol, interface{})
	onStoppedPanic          func(*chasqui.Server, Protocol, interface{})
}

// Attempts to start all the protocols with respect to a server.
// Each protocol must attempt to start, or panic an error. "Starting
// with respect to a server" must have nothing to do with "starting
// with respect to another server", so the interactions must be thought
// as completely isolated among servers.
func (funnel *ProtocolsFunnel) Started(server *chasqui.Server, addr *net.TCPAddr) {
	var protocol Protocol
	defer func() {
		if recovered := recover(); recovered != nil {
			if funnel.onStartedPanic != nil {
				funnel.onStartedPanic(server, addr, protocol, recovered)
			}
			// noinspection GoUnhandledErrorResult
			server.Stop()
		}
	}()
	for _, protocol = range funnel.flattened {
		protocol.Started(server, addr)
		funnel.serverLoadProgress[server] = funnel.serverLoadProgress[server] + 1
	}
	// If no panic occurred, we don't need to keep the server load progress
	// anymore.
	delete(funnel.serverLoadProgress, server)
}

// Executes tha stopped callback safely.
func (funnel *ProtocolsFunnel) safeStoppedCallback(server *chasqui.Server, protocol Protocol) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if funnel.onStoppedPanic != nil {
				funnel.onStoppedPanic(server, protocol, recovered)
			}
		}
	}()
	protocol.Stopped(server)
}

// Attempts to "stop" each protocol's relationship with a server
// (which is already stopped). It iterates, in reverse order, over
// all the protocols that started with it. Stop callbacks may panic,
// and that will be reported, but they shouldn't. Each stop callback
// will be recovered from panics independently.
func (funnel *ProtocolsFunnel) Stopped(server *chasqui.Server) {
	var count int
	var ok bool
	if count, ok = funnel.serverLoadProgress[server]; !ok {
		count = len(funnel.flattened)
	}
	for count > 0 {
		index := count - 1
		funnel.safeStoppedCallback(server, funnel.flattened[index])
		count = index
	}
	delete(funnel.serverLoadProgress, server)
}

// Processes errors related to connections not being accepted.
func (funnel *ProtocolsFunnel) AcceptFailed(server *chasqui.Server, err error) {
	if funnel.onAcceptFailed != nil {
		funnel.onAcceptFailed(server, err)
	}
}

func (funnel *ProtocolsFunnel) AttendantStarted(server *chasqui.Server, attendant *chasqui.Attendant) {
	var protocol Protocol
	defer func() {
		if recovered := recover(); recovered != nil {
			if funnel.onAttendantStartedPanic != nil {
				funnel.onAttendantStartedPanic(server, attendant, protocol, recovered)
			}
			// noinspection GoUnhandledErrorResult
			attendant.Stop()
		}
	}()
	for _, protocol = range funnel.flattened {
		protocol.AttendantStarted(server, attendant)
		funnel.attendantLoadProgress[attendant] = funnel.attendantLoadProgress[attendant] + 1
	}
	// If no panic occurred, we don't need to keep the attendant load progress
	// anymore.
	delete(funnel.attendantLoadProgress, attendant)
}

// This event is strictly bypassed to a callback.
func (funnel *ProtocolsFunnel) MessageThrottled(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message,
	instant time.Time, lapse time.Duration) {
	if funnel.onThrottled != nil {
		funnel.onThrottled(server, attendant, message, instant, lapse)
	}
}

// Delegates the processing to the handlers.
func (funnel *ProtocolsFunnel) MessageArrived(server *chasqui.Server, attendant *chasqui.Attendant, message types.Message) {
	funnel.handlers.Handle(server, attendant, message, funnel.onMessageUnknown, funnel.onMessagePanic)
}

// Executes tha stopped callback safely.
func (funnel *ProtocolsFunnel) safeAttendantStoppedCallback(server *chasqui.Server, attendant *chasqui.Attendant,
	stopType chasqui.AttendantStopType, err error,
	protocol Protocol) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if funnel.onAttendantStoppedPanic != nil {
				funnel.onAttendantStoppedPanic(server, attendant, stopType, err, protocol, recovered)
			}
		}
	}()
	protocol.AttendantStopped(server, attendant, stopType, err)
}

// Attempts to "stop" each protocol's relationship with an attendant
// (which is already stopped). It iterates, in reverse order, over
// all the protocols that started with it. Stop callbacks may panic,
// and that will be reported, but they shouldn't. Each stop callback
// will be recovered from panics independently.
func (funnel *ProtocolsFunnel) AttendantStopped(server *chasqui.Server, attendant *chasqui.Attendant,
	stopType chasqui.AttendantStopType, err error) {
	var count int
	var ok bool
	if count, ok = funnel.attendantLoadProgress[attendant]; !ok {
		count = len(funnel.flattened)
	}
	for count > 0 {
		index := count - 1
		funnel.safeAttendantStoppedCallback(server, attendant, stopType, err, funnel.flattened[index])
		count = index
	}
	delete(funnel.attendantLoadProgress, attendant)
}
