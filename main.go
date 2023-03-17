package main

import (
	"encoding/json"
	"log"
	"sort"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"golang.org/x/exp/slices"
)

//	{
//	  "type": "broadcast",
//	  "message": 1000
//	}
type BroadcastRequestBody struct {
	Type    string `json:"type"`
	Message uint64 `json:"message"`
}

//	{
//	  "type": "broadcast_ok"
//	}
type BroadcastResponseBody struct {
	Type string `json:"type"`
}

//	{
//	  "type": "read"
//	}
type ReadRequestBody struct {
	Type string `json:"type"`
}

//	{
//	  "type": "read_ok",
//	  "messages": [1, 8, 72, 25]
//	}
type ReadResponseBody struct {
	Type     string   `json:"type"`
	Messages []uint64 `json:"messages"`
}

//	{
//	  "type": "topology",
//	  "topology": {
//	    "n1": ["n2", "n3"],
//	    "n2": ["n1"],
//	    "n3": ["n1"]
//	  }
//	}
type TopologyRequestBody struct {
	Type     string              `json:"type"`
	Topology map[string][]string `json:"topology"`
}

type TopologyResponseBody struct {
	Type string `json:"type"`
}

type UnansweredNodeBroadcast struct {
	Message uint64
	Dest    string
	SentAt  time.Time
}

func main() {
	var messages []uint64
	var topology map[string][]string
	var unansweredNodeBroadcasts []UnansweredNodeBroadcast
	// Prevent race condition on shared unansweredNodeBroadcasts access
	var unansweredNodeBroadcastsMutex sync.Mutex
	// Prevent race conditions when writing messages
	var messagesMutex sync.Mutex

	unansweredNodeBroadcastsEventLoopRunning := true

	// Stop the event loop on node teardown
	defer func() { unansweredNodeBroadcastsEventLoopRunning = false }()

	n := maelstrom.NewNode()

	// This goroutine acts over messages that haven't been answered for >=250ms and resends them
	go func() {
		var wg sync.WaitGroup

		for unansweredNodeBroadcastsEventLoopRunning {
			for _, unansweredNodeBroadcast := range unansweredNodeBroadcasts {
				if time.Now().UTC().Add(-150*time.Millisecond).UnixMicro() <= unansweredNodeBroadcast.SentAt.UTC().UnixMicro() {
					continue
				}

				// If the message hasn't been replied to >150ms, re-send it

				wg.Add(1)

				go func(destination string, message uint64) {
					defer wg.Done()

					n.Send(destination, BroadcastRequestBody{
						Type:    "broadcast",
						Message: message,
					})
				}(unansweredNodeBroadcast.Dest, unansweredNodeBroadcast.Message)
			}

			wg.Wait()

			time.Sleep(100 * time.Millisecond)
		}
	}()

	n.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		var body BroadcastRequestBody
		var idxsToRemove []int

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		unansweredNodeBroadcastsMutex.Lock()
		defer unansweredNodeBroadcastsMutex.Unlock()
		for idx, broadcast := range unansweredNodeBroadcasts {
			if msg.Src != broadcast.Dest || body.Message != broadcast.Message {
				continue
			}

			idxsToRemove = append(idxsToRemove, idx)
		}

		// Reverse the array so removing elements is stable
		sort.Sort(sort.Reverse(sort.IntSlice(idxsToRemove)))

		// Remove unanswered broadcasts so they don't get re-sent by the event loop goroutine
		for _, idx := range idxsToRemove {
			unansweredNodeBroadcasts = append(unansweredNodeBroadcasts[:idx], unansweredNodeBroadcasts[idx+1:]...)
		}

		return nil
	})

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body BroadcastRequestBody
		var wg sync.WaitGroup

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// If we already received the message, stop broadcasting
		if slices.Contains(messages, body.Message) {
			return n.Reply(msg, BroadcastResponseBody{
				Type: "broadcast_ok",
			})
		}

		messagesMutex.Lock()
		messages = append(messages, body.Message)
		messagesMutex.Unlock()

		destinations := topology[n.ID()]

		if destinations == nil {
			return n.Reply(msg, BroadcastResponseBody{
				Type: "broadcast_ok",
			})
		}

		// Send messages to the node's topology
		for _, node := range destinations {
			wg.Add(1)

			go func(node string) {
				defer wg.Done()

				unansweredNodeBroadcastsMutex.Lock()
				unansweredNodeBroadcasts = append(unansweredNodeBroadcasts, UnansweredNodeBroadcast{
					Message: body.Message,
					Dest:    node,
					SentAt:  time.Now().UTC(),
				})
				unansweredNodeBroadcastsMutex.Unlock()

				n.Send(node, body)
			}(node)
		}

		wg.Wait()

		return n.Reply(msg, BroadcastResponseBody{
			Type: "broadcast_ok",
		})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		return n.Reply(msg, ReadResponseBody{
			Type:     "read_ok",
			Messages: messages,
		})
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body TopologyRequestBody

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		topology = body.Topology

		return n.Reply(msg, TopologyResponseBody{
			Type: "topology_ok",
		})
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
