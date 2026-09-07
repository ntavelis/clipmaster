// Package mdns wraps github.com/hashicorp/mdns to advertise and discover
// Omaclip instances on the local network.
package mdns

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/rhemvi/omaclip/business/passphrase"

	"github.com/hashicorp/mdns"
)

const (
	serviceType     = "_omaclip._tcp"
	domain          = "local."
	protocolVersion = "version=2"
)

var (
	ErrNoDiscoverableIPs            = fmt.Errorf("mdns: no discoverable IPs, skipping registering to the network")
	ErrInterfaceNotFound            = fmt.Errorf("mdns: requested network interface not found")
	ErrServiceRegistration          = fmt.Errorf("mdns: failed to register service")
	ErrInterfacesCouldNotBeResolved = fmt.Errorf("mdns: could not resolve any network interfaces")
)

// Peer describes a discovered remote Omaclip instance.
type Peer struct {
	Name string `json:"name"`
	Addr string `json:"addr"`
	Port int    `json:"port"`
}

const peerTTLCycles = 3

type interfaceBinding struct {
	iface *net.Interface
	ips   []net.IP
}

// Discoverer registers this instance via mDNS and continuously browses for peers.
type Discoverer struct {
	log             *slog.Logger
	servers         []*mdns.Server
	bindings        []interfaceBinding
	myName          string
	browsePeriod    time.Duration
	passphraseStore *passphrase.Store
	iface           *net.Interface

	mu       sync.RWMutex
	peers    map[string]Peer
	lastSeen map[string]int
	hostname string
}

// New creates a Discoverer. Call Register then Start to begin advertising and browsing.
// If ifaceName is non-empty, mDNS will bind to that network interface only.
func New(
	log *slog.Logger,
	browsePeriod time.Duration,
	hostname string,
	ps *passphrase.Store,
	ifaceName string,
) (*Discoverer, error) {
	d := &Discoverer{
		log:             log,
		browsePeriod:    browsePeriod,
		peers:           make(map[string]Peer),
		lastSeen:        make(map[string]int),
		hostname:        hostname,
		passphraseStore: ps,
	}

	if ifaceName != "" {
		iface, err := net.InterfaceByName(ifaceName)
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %v", ErrInterfaceNotFound, ifaceName, err)
		}
		d.iface = iface
	}

	return d, nil
}

// Register advertises this Omaclip instance at the given port via mDNS.
func (d *Discoverer) Register(port int) error {
	instanceName := fmt.Sprintf("%s-%d", d.hostname, port)
	d.myName = instanceName

	bindings, err := getInterfaceBindings(d.iface)
	if err != nil {
		return err
	}

	txt := []string{protocolVersion, "ph=" + d.passphraseStore.ShortHash()}
	servers := make([]*mdns.Server, 0, len(bindings))
	for _, binding := range bindings {
		hostName := instanceName + "." + domain
		service, err := mdns.NewMDNSService(
			instanceName,
			serviceType,
			domain,
			hostName,
			port,
			binding.ips,
			txt,
		)
		if err != nil {
			shutdownServers(servers)
			return fmt.Errorf("%w: creating service: %v", ErrServiceRegistration, err)
		}

		server, err := mdns.NewServer(&mdns.Config{
			Zone:   service,
			Iface:  binding.iface,
			Logger: log.New(io.Discard, "", 0),
		})
		if err != nil {
			shutdownServers(servers)
			return fmt.Errorf("%w: starting server on %s: %v", ErrServiceRegistration, binding.iface.Name, err)
		}
		servers = append(servers, server)
	}

	d.bindings = bindings
	d.servers = servers
	d.log.Info(
		"mdns registered",
		"instance", instanceName,
		"port", port,
		"interfaces", formatBindings(bindings),
	)
	return nil
}

// Start begins the periodic browse loop until ctx is cancelled.
func (d *Discoverer) Start(ctx context.Context) {
	go d.browseLoop(ctx)
}

// Peers returns a snapshot of currently known peers.
func (d *Discoverer) Peers() []Peer {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]Peer, 0, len(d.peers))
	for _, p := range d.peers {
		out = append(out, p)
	}
	return out
}

// Shutdown tears down the mDNS server.
func (d *Discoverer) Shutdown() {
	for _, server := range d.servers {
		if err := server.Shutdown(); err != nil {
			d.log.Warn("mdns shutdown failed", "error", err)
		}
	}
}

func (d *Discoverer) browseLoop(ctx context.Context) {
	d.browse(ctx)
	ticker := time.NewTicker(d.browsePeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.browse(ctx)
		}
	}
}

func (d *Discoverer) browse(ctx context.Context) {
	if len(d.bindings) == 0 {
		return
	}

	browseCtx, cancel := context.WithTimeout(ctx, d.browsePeriod)
	defer cancel()

	type queryResult struct {
		index int
		peers map[string]Peer
		err   error
	}

	results := make(chan queryResult, len(d.bindings))
	for i, binding := range d.bindings {
		go func(index int, binding interfaceBinding) {
			peers, err := d.queryBinding(browseCtx, binding)
			results <- queryResult{index: index, peers: peers, err: err}
		}(i, binding)
	}

	byBinding := make([]map[string]Peer, len(d.bindings))
	for range d.bindings {
		result := <-results
		byBinding[result.index] = result.peers
		if result.err != nil && browseCtx.Err() == nil {
			d.log.Warn("mdns browse failed", "interface", d.bindings[result.index].iface.Name, "error", result.err)
		}
	}

	seen := make(map[string]Peer)
	for _, peers := range byBinding {
		for name, peer := range peers {
			if _, exists := seen[name]; !exists {
				seen[name] = peer
			}
		}
	}
	d.reconcilePeers(seen)

	for _, p := range seen {
		d.log.Debug("mdns peer discovered", "name", p.Name, "addr", p.Addr, "port", p.Port)
	}
}

func (d *Discoverer) queryBinding(ctx context.Context, binding interfaceBinding) (map[string]Peer, error) {
	entries := make(chan *mdns.ServiceEntry, 256)
	params := mdns.DefaultParams(serviceType)
	params.Domain = domain
	params.Timeout = d.browsePeriod
	params.Interface = binding.iface
	params.Entries = entries
	params.DisableIPv6 = true
	params.Logger = log.New(io.Discard, "", 0)

	err := mdns.QueryContext(ctx, params)
	close(entries)

	seen := make(map[string]Peer)
	for entry := range entries {
		peer, ok := d.peerFromServiceEntry(entry)
		if !ok {
			continue
		}
		seen[peer.Name] = peer
	}

	return seen, err
}

func (d *Discoverer) peerFromServiceEntry(entry *mdns.ServiceEntry) (Peer, bool) {
	if entry == nil || entry.AddrV4 == nil || entry.Port <= 0 {
		return Peer{}, false
	}

	name := normalizeInstanceName(entry.Name)
	if name == "" || d.myName != "" && name == d.myName {
		return Peer{}, false
	}

	versionMatches := false
	for _, field := range entry.InfoFields {
		if strings.HasPrefix(field, "version=") {
			if field != protocolVersion {
				return Peer{}, false
			}
			versionMatches = true
		}
	}
	if !versionMatches {
		d.log.Debug("mdns peer skipped: incompatible protocol", "name", name)
		return Peer{}, false
	}

	if !d.peerMatchesPassphrase(entry.InfoFields) {
		d.log.Debug("mdns peer skipped: passphrase mismatch", "name", name)
		return Peer{}, false
	}

	return Peer{Name: name, Addr: entry.AddrV4.String(), Port: entry.Port}, true
}

func normalizeInstanceName(name string) string {
	suffix := "." + strings.TrimSuffix(serviceType, ".") + "." + strings.TrimSuffix(domain, ".") + "."
	if instance, ok := strings.CutSuffix(name, suffix); ok {
		return instance
	}
	return strings.TrimSuffix(name, ".")
}

func (d *Discoverer) reconcilePeers(seen map[string]Peer) {
	d.mu.Lock()
	defer d.mu.Unlock()

	for name, peer := range seen {
		d.peers[name] = peer
		d.lastSeen[name] = 0
	}
	for name := range d.peers {
		if _, ok := seen[name]; !ok {
			d.lastSeen[name]++
			if d.lastSeen[name] >= peerTTLCycles {
				delete(d.peers, name)
				delete(d.lastSeen, name)
			}
		}
	}
}

func lanIPs(hostname string) []net.IP {
	resolved, err := net.LookupIP(hostname)
	if err != nil {
		return nil
	}

	return filterIPs(resolved)
}

func getInterfaceBindings(userSelectedNetworkInterface *net.Interface) ([]interfaceBinding, error) {
	if userSelectedNetworkInterface != nil {
		ips := ifaceIPs(userSelectedNetworkInterface)
		if len(ips) == 0 {
			return nil, ErrNoDiscoverableIPs
		}
		return []interfaceBinding{{iface: userSelectedNetworkInterface, ips: ips}}, nil
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, ErrInterfacesCouldNotBeResolved
	}

	bindings := make([]interfaceBinding, 0, len(ifaces))
	for i := range ifaces {
		iface := &ifaces[i]
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		ips := filterIPs(ifaceIPs(iface))
		if len(ips) == 0 {
			continue
		}
		bindings = append(bindings, interfaceBinding{iface: iface, ips: ips})
	}

	if len(bindings) == 0 {
		return nil, ErrNoDiscoverableIPs
	}
	return bindings, nil
}

// ifaceIPs returns the IPv4 addresses assigned to a specific network interface.
func ifaceIPs(iface *net.Interface) []net.IP {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}

	var ips []net.IP
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			ips = append(ips, ip4)
		}
	}
	return ips
}

func filterIPs(candidates []net.IP) []net.IP {
	// Docker commonly uses this private range for bridge networks that are not routable across the LAN.
	_, blockedNet, _ := net.ParseCIDR("172.16.0.0/12")

	var ips []net.IP
	for _, ip := range candidates {
		ip4 := ip.To4()
		if ip4 == nil ||
			ip4.IsLoopback() ||
			ip4.IsUnspecified() ||
			ip4.IsMulticast() ||
			ip4.IsLinkLocalUnicast() ||
			!ip4.IsGlobalUnicast() ||
			blockedNet.Contains(ip4) {
			continue
		}
		ips = append(ips, ip4)
	}
	return ips
}

func shutdownServers(servers []*mdns.Server) {
	for _, server := range servers {
		_ = server.Shutdown()
	}
}

func formatBindings(bindings []interfaceBinding) string {
	values := make([]string, len(bindings))
	for i, binding := range bindings {
		ips := make([]string, len(binding.ips))
		for j, ip := range binding.ips {
			ips[j] = ip.String()
		}
		values[i] = fmt.Sprintf("%s[%s]", binding.iface.Name, strings.Join(ips, ","))
	}
	return strings.Join(values, ",")
}

func (d *Discoverer) peerMatchesPassphrase(infoFields []string) bool {
	hash := d.passphraseStore.ShortHash()
	for _, field := range infoFields {
		if after, ok := strings.CutPrefix(field, "ph="); ok {
			return after == hash
		}
	}
	return false
}
