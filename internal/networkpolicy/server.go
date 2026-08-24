package networkpolicy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"
)

var timeNow = time.Now

// PeerAuthorizer obtains an unforgeable local peer identity before any request
// body is decoded. Linux production uses SO_PEERCRED plus executable identity.
type PeerAuthorizer interface {
	AuthorizePeer(*net.UnixConn) (uint32, error)
}

// PeerService is the privileged operation boundary after peer authorization.
// Its input remains the fixed ResolverPolicy value, never a command fragment.
type PeerService interface {
	CreateForPeer(context.Context, uint32, ResolverPolicy) error
	VerifyForPeer(context.Context, uint32, ResolverPolicy) error
	RemoveForPeer(context.Context, uint32, ResolverPolicy) error
}

// Server serves exactly one bounded request per Unix stream connection.
type Server struct {
	listener   *net.UnixListener
	authorizer PeerAuthorizer
	service    PeerService
	closeOnce  sync.Once
}

func NewServer(listener *net.UnixListener, authorizer PeerAuthorizer, service PeerService) (*Server, error) {
	if listener == nil || authorizer == nil || service == nil {
		return nil, errors.New("network policy server dependencies are required")
	}
	return &Server{listener: listener, authorizer: authorizer, service: service}, nil
}

func (s *Server) Serve(ctx context.Context) error {
	if s == nil || s.listener == nil || s.authorizer == nil || s.service == nil || ctx == nil {
		return errors.New("network policy server is unavailable")
	}
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	for {
		connection, err := s.listener.AcceptUnix()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return errors.New("accept network policy client")
		}
		go s.handle(ctx, connection)
	}
}

func (s *Server) Close() error {
	if s == nil || s.listener == nil {
		return nil
	}
	var result error
	s.closeOnce.Do(func() { result = s.listener.Close() })
	return result
}

func (s *Server) handle(ctx context.Context, connection *net.UnixConn) {
	defer connection.Close()
	if err := connection.SetDeadline(timeNow().Add(requestTimeout)); err != nil {
		return
	}
	peer, err := s.authorizer.AuthorizePeer(connection)
	if err != nil {
		return
	}
	reader := bufio.NewReaderSize(connection, maxMessageBytes+1)
	body, readErr := reader.ReadBytes('\n')
	if readErr != nil || len(body) < 2 || len(body) > maxMessageBytes {
		return
	}
	op, policy, err := decodeRequest(body[:len(body)-1])
	if err != nil || s.call(ctx, peer, op, policy) != nil {
		return
	}
	_ = json.NewEncoder(connection).Encode(wireResponse{OK: true})
}

func (s *Server) call(ctx context.Context, peer uint32, op operation, policy ResolverPolicy) error {
	switch op {
	case createOperation:
		return s.service.CreateForPeer(ctx, peer, policy)
	case verifyOperation:
		return s.service.VerifyForPeer(ctx, peer, policy)
	case removeOperation:
		return s.service.RemoveForPeer(ctx, peer, policy)
	default:
		return errors.New("network policy operation is invalid")
	}
}
