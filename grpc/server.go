package grpc

import "context"

// Server is a simple implementation of the generated APIServer interface.
// It simply forwards requests to its controller directly without any extra logic.
// It could be used later on to specify GRPC options specifically for certain routes.
type Server struct {
	ctrl *Controller
}

// NewServer creates a Server given a Controller pointer.
func NewServer(ctrl *Controller) *Server {
	return &Server{
		ctrl: ctrl,
	}
}

// GetRegister calls the server's controller with the GetRegister method.
func (s *Server) GetRegister(ctx context.Context, req *GetRegisterRequest) (*GetRegisterResponse, error) {
	return s.ctrl.GetRegister(ctx, req)
}

// GetValues calls the server's controller with the GetValues method.
func (s *Server) GetValues(ctx context.Context, req *GetValuesRequest) (*GetValuesResponse, error) {
	return s.ctrl.GetValues(ctx, req)
}
