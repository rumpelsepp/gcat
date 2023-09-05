package proxy

import "errors"

var (
	ErrNotSupported        = errors.New("method not supported")
	ErrProxyBusy           = errors.New("proxy is busy")
	ErrProxyNotInitialized = errors.New("proxy is not initialized")
	ErrNotImplemented      = errors.New("proxy method not implemented")
)
