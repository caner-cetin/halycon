package internal

// ContextKey is a custom type derived from string used as a key in context operations.
// It helps prevent key collisions in context values by providing a type-safe way to store and retrieve context data.
type ContextKey string

const (
	// APP_CONTEXT is the key for... you guessed it! use like: cmd.Context().Get(APP_CONTEXT) within cobra commands.
	// see [cmd.AppCtx]
	APP_CONTEXT ContextKey = "halycon.ctx"
)
