package dnsserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/coredns/coredns/plugin/pkg/dnsutil"
	"github.com/coredns/coredns/plugin/pkg/doh"
	"github.com/coredns/coredns/plugin/pkg/response"
	"github.com/coredns/coredns/plugin/pkg/transport"
)

// ServerHTTP represents an instance of a DNS-over-HTTP server.
type ServerHTTP struct {
	*Server
	httpServer *http.Server
	listenAddr  net.Addr
}

// NewServerHTTP returns a new CoreDNS HTTP server and compiles all plugins in to it.
func NewServerHTTP(addr string, group []*Config) (*ServerHTTP, error) {
	s, err := NewServer(addr, group)
	if err != nil {
		return nil, err
	}

	sh := &ServerHTTP{Server: s, httpServer: new(http.Server)}
	sh.httpServer.Handler = sh

	return sh, nil
}

// Serve implements caddy.TCPServer interface.
func (s *ServerHTTP) Serve(l net.Listener) error {
	s.m.Lock()
	s.listenAddr = l.Addr()
	s.m.Unlock()

	return s.httpServer.Serve(l)
}

// ServePacket implements caddy.UDPServer interface.
func (s *ServerHTTP) ServePacket(p net.PacketConn) error { return nil }

// Listen implements caddy.TCPServer interface.
func (s *ServerHTTP) Listen() (net.Listener, error) {
	l, err := net.Listen("tcp", s.Addr[len(transport.HTTP+"://"):])
	if err != nil {
		return nil, err
	}
	return l, nil
}

// ListenPacket implements caddy.UDPServer interface.
func (s *ServerHTTP) ListenPacket() (net.PacketConn, error) { return nil, nil }

// OnStartupComplete lists the sites served by this server
// and any relevant information, assuming Quiet is false.
func (s *ServerHTTP) OnStartupComplete() {
	if Quiet {
		return
	}

	out := startUpZones(transport.HTTP+"://", s.Addr, s.zones)
	if out != "" {
		fmt.Print(out)
	}
	return
}

// Stop stops the server. It blocks until the server is totally stopped.
func (s *ServerHTTP) Stop() error {
	s.m.Lock()
	defer s.m.Unlock()
	if s.httpServer != nil {
		s.httpServer.Shutdown(context.Background())
	}
	return nil
}

// ServeHTTP is the handler that gets the HTTP request and converts to the dns format, calls the plugin
// chain, converts it back and write it to the client.
func (s *ServerHTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.URL.Path != doh.Path {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	msg, err := doh.RequestToMsg(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create a DoHWriter with the correct addresses in it.
	h, p, _ := net.SplitHostPort(r.RemoteAddr)
	port, _ := strconv.Atoi(p)
	dw := &DoHWriter{laddr: s.listenAddr, raddr: &net.TCPAddr{IP: net.ParseIP(h), Port: port}}

	// We just call the normal chain handler - all error handling is done there.
	// We should expect a packet to be returned that we can send to the client.
	ctx := context.WithValue(context.Background(), Key{}, s.Server)
	s.ServeDNS(ctx, dw, msg)

	// See section 4.2.1 of RFC 8484.
	// We are using code 500 to indicate an unexpected situation when the chain
	// handler has not provided any response message.
	if dw.Msg == nil {
		http.Error(w, "No response", http.StatusInternalServerError)
		return
	}

	buf, _ := dw.Msg.Pack()

	mt, _ := response.Typify(dw.Msg, time.Now().UTC())
	age := dnsutil.MinimalTTL(dw.Msg, mt)

	w.Header().Set("Content-Type", doh.MimeType)
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%f", age.Seconds()))
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))
	w.WriteHeader(http.StatusOK)

	w.Write(buf)
}

// Shutdown stops the server (non gracefully).
func (s *ServerHTTP) Shutdown() error {
	if s.httpServer != nil {
		s.httpServer.Shutdown(context.Background())
	}
	return nil
}
