package protocols

import (
	"github.com/universe-10th/chasqui"
	"net"
)

type Protocols map[Protocol]bool

// Implementors of this interface will have the
// following contract:
// - Dependencies(): They will know all of their
//   dependencies at instantiation time and they
//   must guarantee those dependencies will be
//   strictly immutable. Dependencies will provide
//   features to dependent protocols (e.g. features
//   to require authentication) and, since they
//   are protocols on their own, they will also
//   handle their own messages.
// - Handlers(): They will know all of their handlers
//   not just at instantiation time but most likely
//   at design time (in the worst case, they will
//   know them at instantiation time). As with the
//   dependencies, they must guarantee that these
//   handlers will be strictly immutable.
//
// Knowing what services does it provide and what does
// it depend on, the following methods involve all the
// lifecycle:
// - Started(...): Processes when a server has just
//   started at certain address. It can veto such start
//   by panicking an error. The server will then attempt
//   to stop.
// - Stopped(...): Processes when a server has just
//   stopped. The server has already stopped and there
//   is no place (it does not make sense) to veto. This
//   callback will be invoked only if the protocol did
//   run Started() on the server and did not veto it.
// - AttendantStarted(...): Processes when a socket
//   has just connected. It can veto such start by
//   panicking an error. The socket will then stop
//   normally.
// - AttendantStopped(...): Processes when a socket
//   has just stopped. The socket has already stopped
//   and there is no place (it does not make sense) to
//   veto. This callback will be invoked only if the
//   protocol did run AttendantStarted() on the socket
//   and did not veto it.
type Protocol interface {
	Dependencies() Protocols
	Handlers() MessageHandlers
	Started(server *chasqui.Server, addr *net.TCPAddr)
	AttendantStarted(server *chasqui.Server, attendant *chasqui.Attendant)
	AttendantStopped(server *chasqui.Server, attendant *chasqui.Attendant, stopType chasqui.AttendantStopType, err error)
	Stopped(server *chasqui.Server)
}
