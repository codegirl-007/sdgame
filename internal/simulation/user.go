package simulation

// UserLogic represents the behavior of user components in the simulation.
// User components serve as traffic sources and don't process requests themselves.
// Traffic generation is handled by the simulation engine at the entry point.
type UserLogic struct{}

// Tick implements the NodeLogic interface for User components.
// User components don't process requests - they just pass them through.
// The simulation engine handles traffic generation at entry points.
func (u UserLogic) Tick(props map[string]any, queue []*Request, tick int) ([]*Request, bool) {
	// User components just pass through any requests they receive
	// In practice, User components are typically entry points so they
	// receive requests from the simulation engine itself
	return queue, true
}


