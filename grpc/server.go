package grpc

import (
	"context"
)

type Server struct {
	ctrl *Controller
}

func NewServer(ctrl *Controller) *Server {
	return &Server{
		ctrl: ctrl,
	}
}

func (s *Server) GetRegister(ctx context.Context, req *GetRegisterRequest) (*Register, error) {
	return s.ctrl.GetRegister(ctx, req)
}

func (s *Server) GetValues(ctx context.Context, req *GetValuesRequest) (*Values, error) {
	return s.ctrl.GetValues(ctx, req)
}