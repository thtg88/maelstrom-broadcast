package main

import (
	"encoding/json"
	"fmt"
	"log"
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

type BroadcastOkResponseBody struct {
	InReplyTo int    `json:"in_reply_to"`
	Type      string `json:"type"`
}

type UnansweredNodeBroadcast struct {
	Message uint64
	Dest    string
	SentAt  time.Time
}

func main() {
	var messages []uint64
	var destinations []string
	// Prevent race conditions when writing messages
	var messagesMutex sync.Mutex

	n := maelstrom.NewNode()

	n.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		return nil
	})

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body BroadcastRequestBody

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

		go func(destinations []string, body BroadcastRequestBody) {
			var successfulSentDestinations []string
			var successfulSentDestinationsMutex sync.Mutex
			// Send messages to the node's destinations
			// Until we got none left
			for len(destinations) != len(successfulSentDestinations) {
				for _, node := range destinations {
					n.RPC(node, body, func(msg maelstrom.Message) error {
						var broadcastOkBody BroadcastOkResponseBody

						if err := json.Unmarshal(msg.Body, &broadcastOkBody); err != nil {
							return err
						}

						if broadcastOkBody.Type != "broadcast_ok" {
							return fmt.Errorf("expeted type broadcast_ok, got %s", broadcastOkBody.Type)
						}

						log.Printf("from %s, to %s, sent %v", n.ID(), node, body)

						successfulSentDestinationsMutex.Lock()
						defer successfulSentDestinationsMutex.Unlock()

						// Check if we already marked the destination node as successfully sent
						if !slices.Contains(successfulSentDestinations, node) {
							successfulSentDestinations = append(successfulSentDestinations, node)
						}

						return nil
					})
				}

				time.Sleep(1 * time.Second)
			}
		}(destinations, body)

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

		// Only store the destinations for our node
		destinations = body.Topology[n.ID()]

		return n.Reply(msg, TopologyResponseBody{
			Type: "topology_ok",
		})
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
