package source

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// NewChannel creates a new Channel event source with a given event channel.
func NewChannel(evntSrc <-chan event.GenericEvent) *source.Channel {
	// TODO: Add option to support setting destination buffered channel size.
	return &source.Channel{
		Source: evntSrc,
	}
}
