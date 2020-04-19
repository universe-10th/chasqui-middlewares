package protocols

import (
	"errors"
	"github.com/universe-10th/chasqui"
	"github.com/universe-10th/chasqui/types"
	"net"
	"time"
)

var ErrNoProtocols = errors.New("no protocols specified")

// Funnels several protocols simultaneously for the
// managed server & attendants. It MAY handle several
// servers at once, if the protocols are carefully
// designed.
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
	onMessageThrottled      func(*chasqui.Server, *chasqui.Attendant, types.Message, time.Time, time.Duration)
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
	if funnel.onMessageThrottled != nil {
		funnel.onMessageThrottled(server, attendant, message, instant, lapse)
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

// Option to set the "server started panic" callback to handle the panics
// when a protocol tries to initialize for a server.
func WithStartedPanic(callback func(*chasqui.Server, *net.TCPAddr, Protocol, interface{})) func(target *ProtocolsFunnel) {
	return func(target *ProtocolsFunnel) {
		target.onStartedPanic = callback
	}
}

// Option to set the "accept failed" callback to handle when the server cannot
// accept a connection.
func WithAcceptFailed(callback func(*chasqui.Server, error)) func(target *ProtocolsFunnel) {
	return func(target *ProtocolsFunnel) {
		target.onAcceptFailed = callback
	}
}

// Option to set the "attendee started panic" callback to handle the panics
// when a protocol tries to initialize for an attendee.
func WithAttendantStartedPanic(callback func(*chasqui.Server, *chasqui.Attendant, Protocol, interface{})) func(target *ProtocolsFunnel) {
	return func(target *ProtocolsFunnel) {
		target.onAttendantStartedPanic = callback
	}
}

// Option to set the "unknown message" callback to handle when the protocols
// cannot understand a message.
func WithMessageUnknown(callback MessageHandler) func(target *ProtocolsFunnel) {
	return func(target *ProtocolsFunnel) {
		target.onMessageUnknown = callback
	}
}

// Option to set the "message panic" callback to handle when the underlying
// message handling incurs in a panic.
func WithMessagePanic(callback MessagePanicHandler) func(target *ProtocolsFunnel) {
	return func(target *ProtocolsFunnel) {
		target.onMessagePanic = callback
	}
}

// Option to set the "message throttled" callback to handle when the protocols
// cannot understand a message.
func WithMessageThrottled(callback func(*chasqui.Server, *chasqui.Attendant, types.Message, time.Time, time.Duration)) func(target *ProtocolsFunnel) {
	return func(target *ProtocolsFunnel) {
		target.onMessageThrottled = callback
	}
}

// Option to set the "attendant stopped panic" callback to handle the panics
// when a protocol tries to cleanup for an attendant.
func WithAttendantStoppedPanic(callback func(*chasqui.Server, *chasqui.Attendant, chasqui.AttendantStopType, error, Protocol, interface{})) func(target *ProtocolsFunnel) {
	return func(target *ProtocolsFunnel) {
		target.onAttendantStoppedPanic = callback
	}
}

// Option to set the "stopped panic" callback to handle the panics
// when a protocol tries to cleanup for a server.
func WithStoppedPanic(callback func(*chasqui.Server, Protocol, interface{})) func(target *ProtocolsFunnel) {
	return func(target *ProtocolsFunnel) {
		target.onStoppedPanic = callback
	}
}

// Creates a new protocols funnel. It takes some of the protocols
// involved, and also takes the options to configure the callbacks
// for reporting.
func NewProtocolsFunnel(protocols []Protocol, options ...func(target *ProtocolsFunnel)) (*ProtocolsFunnel, error) {
	funnel := &ProtocolsFunnel{}
	if len(protocols) == 0 {
		return nil, ErrNoProtocols
	}
	flattened, err := flatten(protocols)
	if err != nil {
		return nil, err
	} else {
		funnel.flattened = flattened
	}

	handlers := make(MessageHandlers)
	for _, protocol := range flattened {
		if err := handlers.Merge(protocol.Handlers()); err != nil {
			return nil, err
		}
	}
	funnel.handlers = handlers

	funnel.serverLoadProgress = make(map[*chasqui.Server]int)
	funnel.attendantLoadProgress = make(map[*chasqui.Attendant]int)

	for _, option := range options {
		option(funnel)
	}
	return funnel, nil
}
