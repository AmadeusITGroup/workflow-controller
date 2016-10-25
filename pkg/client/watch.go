package client

import (
	"k8s.io/kubernetes/pkg/watch"
)

// Watcher abstracts runtime.Unstructured to watch wapi.Workflow
type Watcher struct {
	Result  chan watch.Event
	Stopped bool
}

var _ watch.Interface = &Watcher{}

// NewWatcher instantiates and initialize a Watcher
func NewWatcher() *Watcher {
	return &Watcher{
		Result: make(chan watch.Event),
	}
}

// ResultChan returns channel events
func (w *Watcher) ResultChan() <-chan watch.Event {
	return w.Result
}

// Stop stops the wathcer
func (w *Watcher) Stop() {
	w.Stopped = true
}
