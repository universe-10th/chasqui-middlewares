package protocols

import "errors"

var ErrCircularDependencies = errors.New("a circular dependency was detected among protocols")

type Protocols map[Protocol]bool

func makeDependencySpec(protocols []Protocol) (depMatrix map[Protocol]Protocols, allProtocols Protocols) {
	depMatrix = make(map[Protocol]Protocols)
	allProtocols = make(map[Protocol]bool)
	for _, protocol := range protocols {
		allProtocols[protocol] = true
		protocolDeps := protocol.Dependencies()
		depMatrix[protocol] = make(map[Protocol]bool, len(protocolDeps))
		for _, dependency := range protocolDeps {
			allProtocols[dependency] = true
			depMatrix[protocol][dependency] = true
		}
	}
	return
}

// Sorts the protocols from less to most dependents. Returns an error if a circular
// dependency is found. Otherwise, returns two arrays: one sorted from less to most
// dependent, and the other being its inverse.
func flatten(protocols []Protocol) (direct []Protocol, reverse []Protocol, err error) {
	length := len(protocols)
	direct = make([]Protocol, length)
	reverse = make([]Protocol, length)

	// Start of the flattening algorithm.
	// First, copying all the dependencies from all the protocols.
	dependencies, allProtocols := makeDependencySpec(protocols)
	allProtocolsLength := len(allProtocols)
	currentProtocolsCount := 0
	// Then, loop infinitely until one of these occurs:
	// 1. The amount of found elements matches the length of the protocols.
	// 2. No new elements could be found.
	for {
		found := false
		// For each protocol, look into their dependencies.
		// If it has no dependencies, it should be added to
		// the "direct" array and removed from the dependency
		// matrix, both its entry and its reference in other
		// entries. Also, the "found" flag must be set.
		for protocol, protocolDeps := range dependencies {
			if len(protocolDeps) == 0 {
				found = true
				direct[currentProtocolsCount] = protocol
				currentProtocolsCount++
				delete(dependencies, protocol)
				for _, protocolDeps2 := range dependencies {
					delete(protocolDeps2, protocol)
				}
			}
		}
		// If all were found, then break and continue with the
		// finisher. If, instead, none was found this iteration,
		// it is because of a circular dependency error. Other
		// cases involve something being found, but not being
		// the last.
		if currentProtocolsCount == allProtocolsLength {
			break
		} else if !found {
			return nil, nil, ErrCircularDependencies
		}
	}
	// End of the flattening algorithm.

	for index, value := range direct {
		reverse[length-index-1] = value
	}
	return direct, reverse, nil
}
