// Package agent defines the core contracts for the 12-Factor Agent Framework.
// It provides the fundamental interfaces and types that enable deterministic,
// event-sourced agent execution with reducer-based state management.
//
// The agent package implements the core principles from the 12-Factor Agents
// specification, including:
//   - Event-sourced state management (Factor 5)
//   - Pure reducer functions (Factor 12)
//   - Effect handlers for side effects
//   - Structured tool calls and intents
//
// Example usage:
//
//	type MyReducer struct{}
//	func (r MyReducer) Reduce(ctx context.Context, current State, event Event) (State, []Intent, error) {
//		// Pure state transformation logic
//		return newState, intents, nil
//	}
//
//	type MyEffectHandler struct{}
//	func (h MyEffectHandler) CanHandle(intent Intent) bool {
//		return intent.Name == "my_effect"
//	}
//	func (h MyEffectHandler) Handle(ctx context.Context, s State, intent Intent) ([]Event, error) {
//		// Side effect execution
//		return events, nil
//	}
package agent

import (
	"context"
	"time"
)

// Event represents a single event in the event-sourced execution model.
// Events are immutable records that capture state changes and external stimuli
// in the agent's execution timeline.
//
// Events must be:
//   - Immutable once created
//   - Serializable to JSON for persistence
//   - Ordered by timestamp for deterministic replay
//   - Unique by ID to prevent duplicates
//
// The Payload field can contain any JSON-serializable data structure
// that represents the event's specific data.
type Event struct {
	// ID is a unique identifier for this event, typically a UUID.
	// Used for deduplication and event ordering.
	ID string `json:"id"`

	// Type categorizes the event for routing and processing.
	// Common types include: "trigger", "tool_call", "state_change", "error"
	Type string `json:"type"`

	// Timestamp records when the event occurred.
	// Used for ordering events and debugging execution timelines.
	Timestamp time.Time `json:"timestamp"`

	// Payload contains the event-specific data as a JSON-serializable value.
	// The structure depends on the event Type.
	Payload any `json:"payload"`
}

// State represents the current state of an agent execution.
// State is immutable and can only be modified through reducer functions.
//
// All state implementations must:
//   - Be JSON-serializable for persistence and debugging
//   - Include a RunID for correlation across events
//   - Support deterministic replay from event history
//   - Be thread-safe for concurrent access
//
// State should contain only the essential data needed for decision-making,
// not transient execution details that can be reconstructed from events.
type State interface {
	// RunID returns the unique identifier for this execution run.
	// Used for correlating events, logs, and traces across the system.
	RunID() string

	// Clone creates a deep copy of the state for safe modification.
	// Implementations must ensure all nested structures are properly copied.
	Clone() State
}

// Intent represents a side effect that should be executed by an effect handler.
// Intents are emitted by reducers and processed by effect handlers to perform
// actions like tool calls, API requests, or state persistence.
//
// Intents are:
//   - Immutable once created
//   - Serializable for persistence and debugging
//   - Idempotent when possible to support retries
//   - Scoped to specific effect handlers by Name
//
// The Args field contains the parameters needed to execute the intent.
type Intent struct {
	// Name identifies the type of effect to execute.
	// Must match the capability of an EffectHandler.
	Name string `json:"name"`

	// Args contains the parameters for executing this intent.
	// Structure depends on the intent Name and target effect handler.
	Args map[string]any `json:"args"`

	// IdempotencyKey ensures the intent is executed only once.
	// Used to prevent duplicate side effects during retries or replays.
	IdempotencyKey string `json:"idempotency_key,omitempty"`
}

// Reducer defines the pure function interface for state transformation.
// Reducers are the core of the event-sourced architecture and must be:
//   - Pure functions with no side effects
//   - Deterministic given the same inputs
//   - Idempotent for safe replay
//   - Fast and efficient for real-time execution
//
// Reducers receive the current state and an event, then return:
//   - The new state (or unchanged state if no modification needed)
//   - A list of intents for side effects to be executed
//   - Any error that occurred during processing
//
// Reducers should not perform I/O operations or call external services.
// All side effects must be expressed as intents for effect handlers.
type Reducer interface {
	// Reduce transforms the current state based on an incoming event.
	//
	// Parameters:
	//   - ctx: Context for cancellation and request-scoped values
	//   - current: The current immutable state
	//   - event: The event to process
	//
	// Returns:
	//   - next: The new state (or current if unchanged)
	//   - intents: Side effects to be executed by effect handlers
	//   - err: Any error that occurred during processing
	//
	// The reducer must be pure and deterministic. Given the same inputs,
	// it must always produce the same outputs.
	Reduce(ctx context.Context, current State, event Event) (next State, intents []Intent, err error)
}

// EffectHandler executes side effects represented by intents.
// Effect handlers are responsible for:
//   - Performing I/O operations (API calls, database writes, etc.)
//   - Executing tool calls and external integrations
//   - Generating new events based on side effect results
//   - Handling errors and retries
//
// Effect handlers must be:
//   - Idempotent when possible (using IdempotencyKey)
//   - Resilient to failures with appropriate retry logic
//   - Observable with proper logging and metrics
//   - Scoped to specific intent types via CanHandle
type EffectHandler interface {
	// CanHandle determines if this handler can process the given intent.
	// Used for routing intents to the appropriate handler.
	//
	// Parameters:
	//   - intent: The intent to evaluate
	//
	// Returns:
	//   - true if this handler can process the intent
	//   - false if the intent should be routed elsewhere
	CanHandle(intent Intent) bool

	// Handle executes the side effect represented by the intent.
	//
	// Parameters:
	//   - ctx: Context for cancellation and request-scoped values
	//   - s: The current state (read-only)
	//   - intent: The intent to execute
	//
	// Returns:
	//   - events: New events generated by the side effect
	//   - err: Any error that occurred during execution
	//
	// The handler should:
	//   - Check idempotency using the IdempotencyKey
	//   - Perform the actual side effect
	//   - Generate appropriate events for the results
	//   - Handle errors gracefully and return structured errors
	Handle(ctx context.Context, s State, intent Intent) (events []Event, err error)
}
