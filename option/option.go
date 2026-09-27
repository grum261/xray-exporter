// Package option provides a generic functional-option type shared by the
// exporter's configurable constructors (expvar.Client, server.Server,
// collector builders).
//
// Each consuming package parameterizes Option over its own unexported
// config type, e.g. `type Option = option.Option[clientConfig]`. Because
// that type argument cannot be named from outside the package, external
// code cannot construct an Option for it — encapsulation comes from the
// unexported type parameter, not from an unexported method.
package option

// Option configures a value of type T.
type Option[T any] interface {
	Apply(*T)
}

type funcOption[T any] struct {
	f func(*T)
}

func (o funcOption[T]) Apply(c *T) { o.f(c) }

// New builds an Option from a function that mutates *T.
func New[T any](f func(*T)) Option[T] {
	return funcOption[T]{f: f}
}
