package sse

import (
	"net/http"
)

// LoggerInterface - Logger in Options should implement it
type LoggerInterface interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

// Options holds server configurations.
type Options struct {
	// RetryInterval change EventSource default retry interval (milliseconds).
	RetryInterval int
	// Headers allow to set custom headers (useful for CORS support).
	Headers map[string]string
	// ChannelNameFunc allow to create custom channel names.
	// Default channel name is the request path.
	ChannelNameFunc func(*http.Request) (chName string, clientName string, version int)
	// OnCloseChannelFunc will be called on channel close if set
	OnClientDisconnectFunc func(chName string, clientName string)
	// All usage logs end up in Logger
	Logger LoggerInterface
	// Send heartbeat message every 15 seconds
	Heartbeat bool
}

func (opt *Options) hasHeaders() bool {
	return opt.Headers != nil && len(opt.Headers) > 0
}
