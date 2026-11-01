package gothic

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// Addr returns the address used by Run and by ListenAndServe when its argument
// is empty.
func (s *Server) Addr() string {
	if s.root != nil {
		return s.root.Addr()
	}
	return s.addr
}

// ListenAndServe starts the server on addr with the configured timeouts.
func (s *Server) ListenAndServe(addr string) error {
	if s.root != nil {
		return s.root.ListenAndServe(addr)
	}
	if addr == "" {
		addr = s.addr
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(ln)
}

// Serve starts the server on ln. It returns http.ErrServerClosed after a
// graceful shutdown.
func (s *Server) Serve(ln net.Listener) error {
	if s.root != nil {
		return s.root.Serve(ln)
	}
	srv := s.newHTTPServer()
	s.setServer(srv)
	defer s.clearServer(srv)
	log.Printf("gothic: listening on %s", displayURL(ln.Addr().String()))
	return srv.Serve(ln)
}

// ServeTLS starts the server on ln using the certificate and key files.
func (s *Server) ServeTLS(ln net.Listener, certFile, keyFile string) error {
	if s.root != nil {
		return s.root.ServeTLS(ln, certFile, keyFile)
	}
	srv := s.newHTTPServer()
	s.setServer(srv)
	defer s.clearServer(srv)
	log.Printf("gothic: listening on %s", displayURL(ln.Addr().String()))
	return srv.ServeTLS(ln, certFile, keyFile)
}

// Shutdown gracefully stops the running server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.root != nil {
		return s.root.Shutdown(ctx)
	}
	s.srvMu.Lock()
	srv := s.srv
	s.srvMu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

// Close immediately closes the running server.
func (s *Server) Close() error {
	if s.root != nil {
		return s.root.Close()
	}
	s.srvMu.Lock()
	srv := s.srv
	s.srvMu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Close()
}

// Run listens on the configured address and serves until ctx is canceled or the
// process receives SIGINT or SIGTERM. It then drains connections for the
// configured shutdown timeout.
func (s *Server) Run(ctx context.Context) error {
	if s.root != nil {
		return s.root.Run(ctx)
	}
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Serve(ln)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()
		if err := s.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func (s *Server) newHTTPServer() *http.Server {
	return &http.Server{
		Handler:           s,
		ReadHeaderTimeout: s.readHeaderTimeout,
		ReadTimeout:       s.readTimeout,
		WriteTimeout:      s.writeTimeout,
		IdleTimeout:       s.idleTimeout,
	}
}

func (s *Server) setServer(srv *http.Server) {
	s.srvMu.Lock()
	s.srv = srv
	s.srvMu.Unlock()
}

func (s *Server) clearServer(srv *http.Server) {
	s.srvMu.Lock()
	if s.srv == srv {
		s.srv = nil
	}
	s.srvMu.Unlock()
}

func displayURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	return "http://" + net.JoinHostPort(host, port)
}
