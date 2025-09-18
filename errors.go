package wp

import "errors"

var (
	ErrSigterm    = errors.New("SIGTERM")
	ErrIdleWorker = errors.New("idle worker")
)
