package handler

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// newChromeTransport returns an http.RoundTripper that mimics Chrome's TLS fingerprint
// and properly supports HTTP/2.
func newChromeTransport() http.RoundTripper {
	return &chromeTransport{}
}

type chromeTransport struct {
	mu       sync.Mutex
	h2Conns  map[string]*http2.ClientConn
}

func (t *chromeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" {
		return http.DefaultTransport.RoundTrip(req)
	}

	addr := req.URL.Host
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = addr + ":443"
	}
	host := req.URL.Hostname()

	t.mu.Lock()
	if t.h2Conns == nil {
		t.h2Conns = make(map[string]*http2.ClientConn)
	}
	cc, ok := t.h2Conns[addr]
	if ok {
		// Check if connection is still usable
		if cc.CanTakeNewRequest() {
			t.mu.Unlock()
			return cc.RoundTrip(req)
		}
		delete(t.h2Conns, addr)
	}
	t.mu.Unlock()

	// Establish new connection
	conn, err := t.dialTLS(req.Context(), addr, host)
	if err != nil {
		return nil, err
	}

	// Create HTTP/2 client connection
	tr := &http2.Transport{}
	newCC, err := tr.NewClientConn(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("http2 client conn: %w", err)
	}

	t.mu.Lock()
	t.h2Conns[addr] = newCC
	t.mu.Unlock()

	return newCC.RoundTrip(req)
}

func (t *chromeTransport) dialTLS(ctx context.Context, addr, host string) (net.Conn, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	tlsConn := utls.UClient(conn, &utls.Config{
		ServerName: host,
		NextProtos: []string{"h2", "http/1.1"},
	}, utls.HelloChrome_Auto)

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	return tlsConn, nil
}
