// Package peers provides support for manually configured peer addresses.
package peers

import (
	"net"
	"strconv"
	"strings"

	"github.com/rhemvi/omaclip/foundation/mdns"
)

type ManualPeers struct {
	p []mdns.Peer
}

func New(p []string) *ManualPeers {
	parsedPeers := []mdns.Peer{}
	for _, addr := range p {
		peer, ok := parsePeer(addr)
		if !ok {
			continue
		}
		parsedPeers = append(parsedPeers, peer)
	}
	return &ManualPeers{p: parsedPeers}
}

func (mp *ManualPeers) Peers() []mdns.Peer {
	return mp.p
}

func parsePeer(addr string) (mdns.Peer, bool) {
	parts := strings.SplitN(addr, "@", 2)
	var name, networkPart string
	if len(parts) == 2 {
		name = parts[0]
		networkPart = parts[1]
	} else {
		networkPart = parts[0]
	}
	networkEntries := strings.Split(networkPart, ":")
	if len(networkEntries) != 2 {
		return mdns.Peer{}, false
	}
	if net.ParseIP(networkEntries[0]) == nil {
		return mdns.Peer{}, false
	}
	port, err := strconv.ParseInt(networkEntries[1], 10, 32)
	if err != nil || port < 1 || port > 65535 {
		return mdns.Peer{}, false
	}
	if name == "" {
		name = networkEntries[0] + ":" + networkEntries[1]
	}
	return mdns.Peer{Addr: networkEntries[0], Port: int(port), Name: name}, true
}
