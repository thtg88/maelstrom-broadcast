## Fly.io Gossip Gloomers Distributed Systems Challenges - Broadcast Challenge

This repo contains a Go implementation of the broadcast challenge for the [Fly.io Gossip Glomers](https://fly.io/dist-sys/) series of distributed systems challenges.

## Requirements

### Go 1.20

You can install Go 1.20 using [gvm](https://github.com/moovweb/gvm) with:

```bash
gvm install go1.20
gvm use go1.20
```

### Maelstrom

Maelstrom is built in [Clojure](https://clojure.org/) so you'll need to install [OpenJDK](https://openjdk.org/).

It also provides some plotting and graphing utilities which rely on [Graphviz](https://graphviz.org/) & [gnuplot](http://www.gnuplot.info/).

If you're using Homebrew, you can install these with this command:

```bash
brew install openjdk graphviz gnuplot
```

You can find more details on the [Prerequisites](https://github.com/jepsen-io/maelstrom/blob/main/doc/01-getting-ready/index.md#prerequisites) section on the Maelstrom docs. Next, you'll need to download Maelstrom itself. These challenges have been tested against [Maelstrom 0.2.3](https://github.com/jepsen-io/maelstrom/releases/tag/v0.2.3). Download the tarball & unpack it. You can run the maelstrom binary from inside this directory.

## Build

From the project's root directory:

```bash
go build .
```

## Test

To use the different Maelstrom test commands, please refer to the Fly.io instructions, starting with the [Single-Node Broadcast](https://fly.io/dist-sys/3a/).

### Challenge #3a: Single-Node Broadcast

https://fly.io/dist-sys/3a/

```bash
# Make sure to replace `~/go/bin/maelstrom-executable`
# with the full path of the executable you built above
./maelstrom test -w broadcast --bin ~/go/bin/maelstrom-broadcast \
  --node-count 1 --time-limit 20 --rate 10
```

### Challenge #3b: Multi-Node Broadcast

https://fly.io/dist-sys/3b/

Your node should propagate values it sees from broadcast messages to the other nodes in the cluster. It can use the topology passed to your node in the topology message or you can build your own topology.

The simplest approach is to simply send a node's entire data set on every message, however, this is not practical in a real-world system. Instead, try to send data more efficiently as if you were building a real broadcast system.

Values should propagate to all other nodes within a few seconds.

```bash
# Make sure to replace `~/go/bin/maelstrom-executable`
# with the full path of the executable you built above
./maelstrom test -w broadcast --bin ~/go/bin/maelstrom-broadcast \
  --node-count 5 --time-limit 20 --rate 10
```

### Challenge #3c: Fault Tolerant Broadcast

https://fly.io/dist-sys/3c/

Your node should propagate values it sees from broadcast messages to the other nodes in the cluster—even in the face of network partitions! Values should propagate to all other nodes by the end of the test. Nodes should only return copies of their own local values.

```bash
# Make sure to replace `~/go/bin/maelstrom-executable`
# with the full path of the executable you built above
./maelstrom test -w broadcast --bin ~/go/bin/maelstrom-broadcast \
  --node-count 5 --time-limit 20 --rate 10 --nemesis partition
```

### Challenge #3d: Efficient Broadcast, Part I

https://fly.io/dist-sys/3d/

```bash
# Make sure to replace `~/go/bin/maelstrom-executable`
# with the full path of the executable you built above
./maelstrom test -w broadcast --bin ~/go/bin/maelstrom-broadcast --node-count 25 --time-limit 20 --rate 100 --latency 100
```

### Challenge #3e: Efficient Broadcast, Part II

https://fly.io/dist-sys/3e/

With the same node count of 25 and a message delay of 100ms, your challenge is to achieve the following performance metrics:

- Messages-per-operation is below 20
- Median latency is below 1 second
- Maximum latency is below 2 seconds

```bash
# Make sure to replace `~/go/bin/maelstrom-executable`
# with the full path of the executable you built above
./maelstrom test -w broadcast --bin ~/go/bin/maelstrom-broadcast \
  --node-count 25 --time-limit 20 --rate 100 --latency 100
```
