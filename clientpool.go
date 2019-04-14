package main

import (
	"container/list"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

const maxBufferSize = 1024
const heartbeatTTL = 10 // seconds

type ClientPool struct {
	clients   list.List
	clientMap map[string]*list.Element
	server    net.PacketConn
	mux       sync.Mutex
}

type Client struct {
	Addr      net.Addr
	Timestamp int64
}

// Constructor for ClientPool
func MakeClientPool(address string) (ClientPool, error) {
	pc, err := net.ListenPacket("udp", address)
	if err != nil {
		return ClientPool{}, err
	}

	log.Printf("Listening at %s\n", address)

	return ClientPool{
		clientMap: make(map[string]*list.Element),
		clients:   *list.New(),
		server:    pc,
	}, nil
}

// Broadcasts a message to all "live" clients. Prunes the list of active clients as a side effect.
func (cp *ClientPool) Send(message HerdCommand) error {
	wireMessage, err := proto.Marshal(&message)
	if err != nil {
		return err
	}

	now := time.Now().Unix()

	cp.mux.Lock()
	defer cp.mux.Unlock()

	for e := cp.clients.Front(); e != nil; e = e.Next() {
		if e.Value == nil {
			continue
		}
		client := e.Value.(Client)

		if now-client.Timestamp < heartbeatTTL {
			_, err := cp.server.WriteTo(wireMessage, client.Addr)
			if err != nil {
				return err
			}
		} else {
			// this client has expired. remove it from the pool
			cp.clients.Remove(e)
			delete(cp.clientMap, client.Addr.String())
		}

		log.Printf("packet-written: message=%s to=%s\n", message.String(), client.Addr.String())
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

			// 1. check the map. if exists then remove existing linked list node.
			// 2. prepend new node with Timestamp, Addr to the linked list
			// 3. add entry to map of Addrs pointing to this new node
			cp.mux.Lock()
			existingPointer, ok := cp.clientMap[addr.String()]
			if ok {
				cp.clients.Remove(existingPointer)
			}

			// clients can send "bye" if they want to proactively disconnect. in that case,
			// do not add them back to the pool.
			if !strings.HasPrefix(string(buffer), "bye") {
				newNode := cp.clients.PushFront(Client{Addr: addr, Timestamp: time.Now().Unix()})
				cp.clientMap[addr.String()] = newNode
			}
			cp.mux.Unlock()
		}
	}()

	return doneChan
}
