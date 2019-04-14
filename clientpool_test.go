package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
)

// This is an integration test that creates a few UDP clients and sends
// some fan-out messages via a ClientPool instance, then asserts that
// they all arrive as expected.
func TestClientPool(t *testing.T) {
	cp, err := MakeClientPool("127.0.0.1:5000")
	if err != nil {
		log.Fatal(err)
	}
	_ = cp.Listen()

	clients := make([]net.Conn, 4)

	// make a pool of 4 clients
	for i := 0; i < 4; i++ {
		conn, err := net.Dial("udp", "127.0.0.1:5000")
		if err != nil {
			log.Fatal(err)
		}

		_, err = conn.Write([]byte("hi"))
		if err != nil {
			t.Errorf("Some error %v\n", err)
		}
		clients[i] = conn
	}

	// TODO there is a race condition bug! this is a workaround...
	time.Sleep(100 * time.Millisecond)

	// send them each 2 messages
	for i := 0; i < 2; i++ {
		message := HerdCommand{
			Status:   int64(1 + i),
			Data1:    int64(2 + i),
			Data2:    int64(3 + i),
			Deadline: int64(4),
		}
		err = cp.Send(message)
		if err != nil {
			log.Fatal(err)
		}
	}

	// TODO there is a race condition bug! this is a workaround...
	time.Sleep(100 * time.Millisecond)

	// assert that each client got both commands correctly
	for i := 0; i < 4; i++ {
		conn := clients[i]
		p := make([]byte, 2048)
		// read 2 messages off the connection
		for i := 0; i < 2; i++ {
			_, err = bufio.NewReader(conn).Read(p)
			if err == nil {
				command := &HerdCommand{}
				err = proto.Unmarshal(p, command)
				if command.Deadline != 4 {
					t.Errorf("Got %v for command.Deadline", command.Deadline)
				}
			} else {
				t.Errorf("Some error %v\n", err)
			}
		}
	}

	// TODO there is a race condition bug! this is a workaround...
	time.Sleep(100 * time.Millisecond)

	// have 2 clients say bye
	for i := 0; i < 2; i++ {
		conn := clients[i]
		_, err = conn.Write([]byte("bye"))
		if err != nil {
			t.Errorf("Some error %v\n", err)
		}
	}

	// TODO there is a race condition bug! this is a workaround...
	time.Sleep(100 * time.Millisecond)

	// send them all a message
	message := HerdCommand{
		Status:   int64(1),
		Data1:    int64(2),
		Data2:    int64(3),
		Deadline: int64(4),
	}
	err = cp.Send(message)
	if err != nil {
		t.Errorf("Some error %v\n", err)
	}

	// TODO there is a race condition bug! this is a workaround...
	time.Sleep(100 * time.Millisecond)

	fmt.Println("--- checking dead clients")

	// assert that first 2 clients did not get messages
	for i := 0; i < 2; i++ {
		conn := clients[i]
		_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		p := make([]byte, 2048)
		_, err := conn.Read(p)
		e, _ := err.(net.Error)
		if !e.Timeout() {
			t.Errorf("Got %v on even though we said bye", p)
		}
	}

	fmt.Println("--- checking live clients")

	// assert that last 2 clients got 1 message each
	for i := 2; i < 4; i++ {
		conn := clients[i]
		p := make([]byte, 2048)
		_, err = bufio.NewReader(conn).Read(p)
		if err == nil {
			command := &HerdCommand{}
			err = proto.Unmarshal(p, command)
			if command.Deadline != 4 {
				t.Errorf("Got %v for command.Deadline", command.Deadline)
			}
		} else {
			t.Errorf("Some error %v\n", err)
		}
	}
}
