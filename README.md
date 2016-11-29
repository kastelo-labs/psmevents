psmevents
=========

This program connects to the PSM event stream and pretty prints the
events for human consumption. Object update events can be printed in
`diff -u` format.

`psmevents` is an open source component provided without any warranty or
support. Please read the LICENSE file.

## Usage

```
Usage:
  psmevents [options] [PSM JSON-RPC address]

Options (with their default values):
  -diff=true: Use diff for object.updated events
  -diff-context=2: Number of lines in diff context
  -groups=true: Subscribe to group events
  -sessions=true: Subscribe to session events
  -subscribers=true: Subscribe to subscriber events

When no address is given, events are read from stdin.

Examples:
  psmevents < events.json                # parse and clarify events from a capture file
  psmevents -diff=false 192.0.2.23:3994  # connect to PSM at 192.0.2.23, subscribe to all events
```

