package ce

import (
	"errors"
)

var (
	ErrLoadProviderFailed = errors.New("load provider failed")
	ErrMergeConfigFailed  = errors.New("merge config failed")
	ErrWatchConfigFailed  = errors.New("watch config failed")
	ErrWatchNotSupported  = errors.New("the configuration provider does not support watching")
)
