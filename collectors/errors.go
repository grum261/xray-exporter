package collectors

import "errors"

// ErrClientRequired is returned by the collector constructors when the
// supplied client/snapshotter is nil.
var ErrClientRequired = errors.New("client is required")
