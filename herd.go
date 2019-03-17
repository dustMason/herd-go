package main

import (
	"log"

	"github.com/rakyll/portmidi"
)

// Emitter:
// - runs in a loop in its own goroutine
// - polls the same go channel that MidiInput is writing to
// - depending on the message itself, writes each message out to one or more clients

// HerdMidiMessage:
// - formed right away by MidiInput when it comes in, sent out to clients by Emitter
// - contains raw midi message, timestamp of when it was received by MidiInput

func main() {
	err := portmidi.Initialize()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("found %d MIDI devices\n", portmidi.CountDevices())

	in, err := portmidi.NewInputStream(0, 1024)
	if err != nil {
		log.Fatal(err)
	}
	defer in.Close()

	midiChan := in.Listen()

  cp, err := MakeClientPool("127.0.0.1:5000")
	if err != nil {
		log.Fatal(err)
	}
  clientPoolDoneChan := cp.Listen()

  for {
		select {
		case err := <-clientPoolDoneChan:
			log.Fatal(err)
			return
		case midiEvent := <-midiChan:
			log.Println(midiEvent)
			err := cp.Send(string(midiEvent.Data1))
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
