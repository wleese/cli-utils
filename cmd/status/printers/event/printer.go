// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package event

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling/collector"
	pollevent "sigs.k8s.io/cli-utils/pkg/kstatus/polling/event"
	"sigs.k8s.io/cli-utils/pkg/object"
)

// Printer implements the Printer interface and outputs the resource
// status information as a list of events as they happen.
type Printer struct {
	IOStreams genericclioptions.IOStreams
}

// NewPrinter returns a new instance of the eventPrinter.
func NewPrinter(ioStreams genericclioptions.IOStreams) *Printer {
	return &Printer{
		IOStreams: ioStreams,
	}
}

// Print takes an event channel and outputs the status events on the channel
// until the channel is closed. The provided cancelFunc is consulted on
// every event and is responsible for stopping the poller when appropriate.
// This function will block.
func (ep *Printer) Print(ch <-chan pollevent.Event, identifiers object.ObjMetadataSet,
	cancelFunc collector.ObserverFunc) error {
	coll := collector.NewResourceStatusCollector(identifiers)
	// The actual work is done by the collector, which will invoke the
	// callback on every event. In the callback we print the status
	// information and call the cancelFunc which is responsible for
	// stopping the poller at the correct time.
	done := coll.ListenWithObserver(ch, collector.ObserverFunc(
		func(statusCollector *collector.ResourceStatusCollector, e pollevent.Event) {
			ep.printStatusEvent(e)
			cancelFunc(statusCollector, e)
		}),
	)
	// Listen to the channel until it is closed.
	var err error
	for msg := range done {
		err = msg.Err
	}
	return err
}

func (ep *Printer) printStatusEvent(se pollevent.Event) {
	switch se.Type {
	case pollevent.ResourceUpdateEvent:
		id := se.Resource.Identifier
		printResourceStatus(id, se, ep.IOStreams)
	case pollevent.ErrorEvent:
		id := se.Resource.Identifier
		gk := id.GroupKind
		fmt.Fprintf(ep.IOStreams.Out, "%s error: %s\n", resourceIDToString(gk, id.Name),
			se.Error.Error())
	}
}

// resourceIDToString returns the string representation of a GroupKind and a resource name.
func resourceIDToString(gk schema.GroupKind, name string) string {
	return fmt.Sprintf("%s/%s", strings.ToLower(gk.String()), name)
}

func printResourceStatus(id object.ObjMetadata, se pollevent.Event, ioStreams genericclioptions.IOStreams) {
	fmt.Fprintf(ioStreams.Out, "%s is %s: %s\n", resourceIDToString(id.GroupKind, id.Name),
		se.Resource.Status.String(), se.Resource.Message)
}
