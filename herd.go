package main

import (
	"log"
	"time"

	"github.com/rakyll/portmidi"
)

const messageDeadline = 100 // milliseconds

func main() {
	start := time.Now()

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

			msSinceStart := int64(time.Since(start) / time.Millisecond)

			message := HerdCommand{
				Status:   midiEvent.Status,
				Data1:    midiEvent.Data1,
				Data2:    midiEvent.Data2,
				Deadline: msSinceStart + messageDeadline,
			}
			err := cp.Send(message)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
