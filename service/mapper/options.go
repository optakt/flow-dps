package mapper

// WithTransition specifies which TransitionFunc should be used when the state machine
// has the given status.
func WithTransition(status Status, transition TransitionFunc) func(*FSM) {
	return func(f *FSM) {
		f.transitions[status] = transition
	}
}
