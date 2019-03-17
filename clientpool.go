package main

import (
	"log"
	"net"
	"sync"
	"time"
)

// plans
// - split clients up into groups based on desired startup options (given at runtime)
// - Clients should be a data structure with these properties:
//   - fast (better than O(n)) listing of all clients updated in the last n seconds. (use periodic bucketing?)
//   - fixed time updating of a single client
// - This should probably be solved in the same way an LRU cache is implemented. allocate a fixed size
//   list for Clients, keep recently seen clients at the top of the list and let old ones fall off.

const maxBufferSize = 1024
const heartbeatTTL = 10 // seconds

type ClientPool struct {
	Clients map[net.Addr]int64 // client id -> timestamp last seen
	server net.PacketConn
	mux sync.Mutex
}

// Constructor for ClientPool
func MakeClientPool(address string) (ClientPool, error) {
	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return ClientPool{}, err
	}

	log.Printf("Listening at %s\n", address)

	return ClientPool{
		Clients: make(map[net.Addr]int64),
		server: pc,
	}, nil
}

// Broadcasts a message to all "live" clients
func (cp *ClientPool) Send(message string) error {
	clients := cp.liveClients()
	for _, addr := range clients {
		_, err := cp.server.WriteTo([]byte(message), addr)
		if err != nil {
			return err
		}

		log.Printf("packet-written: message=%s to=%s\n", message, addr.String())
	}

	return nil
}

// Starts listening for heartbeats from clients
func (cp *ClientPool) Listen() (channel chan error) {
	doneChan := make(chan error, 1)

	go func() {
		buffer := make([]byte, maxBufferSize)

		defer cp.server.Close()

		for {
			_, addr, err := cp.server.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}

			cp.mux.Lock()
      cp.Clients[addr] = time.Now().Unix()
      cp.mux.Unlock()
		}
	}()

	return doneChan
}

func (cp *ClientPool) liveClients() []net.Addr {
	cp.mux.Lock()
	defer cp.mux.Unlock()

	clients := make([]net.Addr, 0)
	now := time.Now().Unix()
	for addr := range cp.Clients {
		if now - cp.Clients[addr] < heartbeatTTL {
			clients = append(clients, addr)
		}
	}
	return clients
}
