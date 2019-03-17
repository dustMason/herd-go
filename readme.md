# Herd

Implementing herd in go as an excuse to learn the language better.

### Running It

Requires `portmidi` for MIDI connectivity. `brew install portmidi` to install.

Built with go 1.12. Install go with `brew install go` and then run `go build && ./herd`
to run a local server. The server expects a MIDI device to be available via CoreMIDI
and will connect to the one with id 0.

The server expects clients to send heartbeats via UDP. You can simulate this with
`socat` (`brew install socat`) using `socat - UDP:localhost:5000` and then hitting
enter to send a packet. After doing so, MIDI messages coming from the MIDI device
will be broadcast to this client. The server will consider the client to be "dead"
unless at least one packet arrives every 10 seconds. It can be revived at any time
by typing a character in the `socat` session.

