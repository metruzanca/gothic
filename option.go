package gothic

import (
	"log/slog"
	"net"
	"strings"
	"time"
)

// Option configures a Server.
type Option func(*Server)

// WithAddr sets the default address for Run and the address ListenAndServe
// uses when it is called with an empty string.
func WithAddr(addr string) Option {
	return func(s *Server) {
		if addr != "" {
			s.addr = addr
		}
	}
}

// WithLogger sets the logger used for access logs and errors.
func WithLogger(logger *slog.Logger) Option {
	return func(s *Server) {
		if logger == nil {
			logger = slog.Default()
		}
		s.logger = logger
	}
}

// WithMaxBodyBytes sets the procedure body limit.
func WithMaxBodyBytes(n int64) Option {
	return func(s *Server) {
		if n > 0 {
			s.maxBodyBytes = n
		}
	}
}

// WithReadHeaderTimeout sets how long a client may take to send headers.
func WithReadHeaderTimeout(d time.Duration) Option {
	return func(s *Server) { s.readHeaderTimeout = d }
}

// WithReadTimeout sets the read timeout for the whole request.
func WithReadTimeout(d time.Duration) Option {
	return func(s *Server) { s.readTimeout = d }
}

// WithWriteTimeout sets the response write timeout.
func WithWriteTimeout(d time.Duration) Option {
	return func(s *Server) { s.writeTimeout = d }
}

// WithIdleTimeout sets how long an idle connection is kept open.
func WithIdleTimeout(d time.Duration) Option {
	return func(s *Server) { s.idleTimeout = d }
}

// WithShutdownTimeout bounds graceful shutdown in Run and Shutdown.
func WithShutdownTimeout(d time.Duration) Option {
	return func(s *Server) {
		if d > 0 {
			s.shutdownTimeout = d
		}
	}
}

// WithRequestTimeout bounds the request context. Handlers must honor the
// context to observe the deadline.
func WithRequestTimeout(d time.Duration) Option {
	return func(s *Server) {
		if d > 0 {
			s.requestTimeout = d
		}
	}
}

// WithoutDefaultMiddleware disables the request ID, recoverer, and access log.
// The request timeout still applies when it is configured.
func WithoutDefaultMiddleware() Option {
	return func(s *Server) { s.useDefaultMiddleware = false }
}

// WithTrustedProxies trusts forwarding headers from the given IPs or CIDRs. It
// panics on a malformed value, which is a programming error.
func WithTrustedProxies(cidrs ...string) Option {
	return func(s *Server) {
		for _, value := range cidrs {
			network := parseNetwork(value)
			if network == nil {
				panic("gothic: invalid trusted proxy " + value)
			}
			s.trustedProxies = append(s.trustedProxies, network)
		}
	}
}

func parseNetwork(value string) *net.IPNet {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if _, network, err := net.ParseCIDR(value); err == nil {
		return network
	}
	ip := net.ParseIP(value)
	if ip == nil {
		return nil
	}
	bits := 32
	if ip.To4() == nil {
		bits = 128
	}
	return &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)}
}
