package main

import (
	"encoding/json"
	"log"

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

func main() {
	var messages []uint64
	var topology map[string][]string

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

		messages = append(messages, body.Message)

		destinations := topology[n.ID()]

		if destinations == nil {
			return n.Reply(msg, BroadcastResponseBody{
				Type: "broadcast_ok",
			})
		}

		for _, node := range destinations {
			go func(node string) {
				n.Send(node, body)
			}(node)
		}

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
