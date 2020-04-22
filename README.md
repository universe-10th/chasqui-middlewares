# chasqui-protocols
Provides a similar feature to what websites tend to have as "middleware" but this
time in a socket context. In this case, protocols capture their messages directly.

Please note, this is a feature over [universe-10th/chasqui](http://github.com/universe-10th/chasqui)
and implies previous knowledge of how `chasqui` works (concept, architecture and types).

Usage
-----

This is a server-only feature (not means for client sockets) which allows a more
comfortable way of creating the handling of the whole server lifecycle. This
handling is composed by different *protocols* that can work together by having
interdependence.

A protocol satisfies the `Protocol` interface, which is defined as:

    type Protocol interface {
        Dependencies() Protocols
        Handlers() MessageHandlers
        Started(server *chasqui.Server, addr *net.TCPAddr)
        AttendantStarted(server *chasqui.Server, attendant *chasqui.Attendant)
        AttendantStopped(server *chasqui.Server, attendant *chasqui.Attendant, stopType chasqui.AttendantStopType, err error)
        Stopped(server *chasqui.Server)
    }
    
This implies a syntactic and semantic contract defined as follows:

  - `Dependencies()` must return a set of protocols this protocol depends on. If
    no protocols are to be specified, just return `nil` with no issue. One thing
    that must be taken into consideration is that dependencies **must not be
    instantiated for the current protocol / inside this method but be external**
    references (i.e. able to be used by others and traversed only once). This is
    important, for dependencies will be *flattened* on protocols startup, so they
    know which is the startup order from *independent* to *most-dependent* protocols.
    In the opposite direction, the teardown order (most-dependent -> independent)
    will also be considered.
  - `Started()` must initialize the protocol for a given server. This method will
    be run in *startup order* for all the protocols. If something should abort the
    load for a server, and should call the server to stop, panicking any value is
    safe for that (later, it will be explained how to handle the panicked value).
  - `Stopped()` must finalize the protocol for a given server. This method will be
    run in *teardown order* for all the protocols that have been initialized with
    success for the server being stopped (this means: these methods will be run
    whether the server was told to stop by a panic, or a regular server stop).
  - `AttendantStarted()` and `AttendantStopped()` work in an analogous way but
    within a server lifecycle (between the `Started` and `Stopped` calls) and for
    particular connections. This means: each protocol will attempt to initialize
    them in *startup order* and then attempt to stop them in *teardown order*, and
    only for the protocols that successfully initialized them in first place.

Just to remember: the Stop callbacks exist just for cleanup purpose: they will imply
the underlying connection / socket respectively is already closed.

There is another method, which is the central core of protocols:

  - `Handlers()` returns a dictionary mapping from a command name (string) and a
    handler function. Each protocol will tell which commands do they understand
    and how do they handle them. Clashes (i.e. command names being handled by
    two or more protocols being used together) will result in error, so it is
    recommended that a prefix feature to exist in the protocol being developed
    to let users prefix the commands in the protocols to avoid eventual clashes.
    Each handler processes the server-side logic of incoming messages, but does
    not give any restrictions to what messages can be sent to the client sockets.

Once the desired protocols are implemented and instantiated, they must be put in
an *array* of protocols and funneled together, with some code like this:

Considering the `chasqui` and `chasqui-protocols` being imported:

    import (
        "fmt"
        "github.com/universe-10th/chasqui"
        "github.com/universe-10th/chasqui-protocols"
    )

A chunk of code would be:

	server := ... build a standard *chasqui.Server instance ...
	if err := server.Run("0.0.0.0:3000"); err != nil {
		fmt.Printf("An error was raised while trying to start the server at address 0.0.0.0:3000: %s\n", err)
		return
	} else if funnel, err := protocols.NewProtocolsFunnel([]protocols.Protocol{... your protocol instances ...}); err != nil {
		fmt.Printf("An error was raised while trying to create the server funnel: %s\n", err)
		return
    } else {
        chasqui.FunnelServerWith(server, funnel)
    }

The only remaining thing to know regarding the funnels is the set of allowed optional
callbacks to the `protocols.NewProtocolsFunnel` constructor. These options will be described
now:

  * `protocols.WithStartedPanic(callback func(*chasqui.Server, *net.TCPAddr, protocols.Protocol, interface{}))`
    sets a function that will handle a panic occurring in a server's *startup* cycle. For each server
    startup cycle, this callback will be invoked only once (just for the panicking protocol) and then
    the server will stop.
  * `protocols.WithAcceptFailed(callback func(*chasqui.Server, error))` sets a function that will
    handle an error occurring while a server tries to accept a connection. This is a direct optional
    implementation of the `AcceptFailed` method in the `chasqui.ServerFunnel` contract.
  * `protocols.WithAttendantStartedPanic(callback func(*chasqui.Server, *chasqui.Attendant, protocols.Protocol, interface{}))`
    sets a function that will handle a panic occurring in a socket's *startup* cycle. For each server
    and each socket's startup cycle, this callback will be invoked only once (just for the panicking
    protocol) and then the socket will stop.
  * `protocols.WithMessageUnknown(callback protocols.MessageHandler)` sets a function that will handle
    messages that no protocol can handle. Any logic can be executed here: a standard reply message, or
    perhaps telling the socket to stop,... users are totally free here.
  * `protocols.WithMessagePanic(callback MessagePanicHandler)` sets a function that will handle when a
    panic occurs inside the handling of a known message or [the handling of] an "unknown message".
  * `protocols.WithMessageThrottled(callback func(*chasqui.Server, *chasqui.Attendant, types.Message, time.Time, time.Duration)) func(target *ProtocolsFunnel)WithMessageThrottled(callback func(*chasqui.Server, *chasqui.Attendant, types.Message, time.Time, time.Duration))`
    sets a function that will handle when a message is being throttled. This is a direct optional
    implementation of the `MessageThrottled` method in the `chasqui.ServerFunnel` contract.
  * `protocols.WithAttendantStoppedPanic(callback func(*chasqui.Server, *chasqui.Attendant, chasqui.AttendantStopType, error, protocols.Protocol, interface{}))`
    sets a function that will handle a panic occurring in a socket's *teardown* cycle. For each server
    and each socket's teardown cycle, this callback will be invoked once for *each* panicking protocol,
    in *teardown order*.
  * `protocols.WithStoppedPanic(callback func(*chasqui.Server, Protocol, interface{}))` will set a
    function that will handle a panic occurring in a server's *teardown* cycle. For each server teardown
    cycle, this callback will be invoked once for *each* panicking protocol, in *teardown order*.

This said, **these functions must guarantee to not panic**. Otherwise, the entire server funnel will
crash, and perhaps not even be correctly cleanup, for the panicking server.