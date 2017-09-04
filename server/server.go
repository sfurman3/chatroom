// TODO: Server implementation

// TODO: Cite the paper: "Consistent Global States of Distributed Systems": by
// Özalp Babaoğlu and Keith Marzullo.

// Concepts & Terminology:
// -----------------------
// - "distributed system"
//   + a collection of sequential 'processes' p_1, ..., p_n and a network
//     capable of implementing unidirectional communication 'channels' between
//     pairs of processes for message exchange. Channels are reliable but may
//     deliver messages out of order.
//   + p_0 denotes a monitor process, which observes the system in order to
//     construct global states
//
// - "events": we define 3 possible events
//   1. local events
//   2. send(m)         (send events)
//   3. receive(m)      (receive events)
//
//   + e_i^n ::= the n'th event of process p_i
//
// - "causal precedence"
//   + e -> e'    (e causally precedes e' (i.e. "happens before"))
//   + causal precedence implies that event e MAY HAVE influenced event e'
//
// - "concurrency"
//   + e || e' <=> NOT(e -> e') AND NOT(e' -> e)    (e and e' are concurrent)
//   + by definition concurrent events ARE NOT causally related (i.e. neither
//     influenced the other)
//
// - "local history"
//   + a (possibly infinite) sequence h_i of events local to a process p_i
//
// - "global history"
//   + a set H ::= h_1 (union) h_2 ... (unordered) containing all events in a
//     system
//
// - "global state"
//   + an n-tuple of states (sigma_i, sigma_j, ..., sigma_n) one for each
//     process
//
// - "cut"
//   + a subset C of a system's global history H, containing the union of the
//     first x events in each process, where x may be different for each
//     process
//   + the set containing the most recent event in each process is called the
//     "frontier" of the cut, which has a corresponding "global state"
//
// - "run"
//   + a total ordering R that contains all the events in the global history
//     and is 'consistent' with each local history (i.e. the events of a
//     process in the run occur in the same relative order as they do in their
//     local history)
//
// - "consistency"
//   + multiple objects defined here can be consistent:
//     * cut: (e in C) AND (e' -> e) => e' in C
//       - i.e. for every event in the cut, there is also every event that
//         causally precedes it
//	 - simply, for every receive in the cut, there is also a corresponding
//	   send
//	 - each consistent cut corresponds to a "consistent global state"
//     * NOTE: a run corresponds to a sequence of global states. if a run is
//       consistent, then so are the global states in the sequence
//
// - "receive"
//   + the 'event' of a message arriving at its destination (but before it is
//     passed to the recipient process)
//
// - "observation" (of a distributed computation by a "monitor" process)
//   + the sequence of events corresponding to the order in which the
//     notification messages arrive
//   + any permutation of a run is a possible observation of it
//   + a "consistent observation" MUST correspond to a consistent run
//
// - "deliver"
//   + the _act_ of presenting received messages to a process
//   + a "delivery rule" is a rule for when to deliver received messages
//
// - "causal delivery"
//   + send_i(m) -> send_j(m) => receive_i(m) -> receive_j(m)
//
// - "clock condition":
//   + e' -> e => C(e') -> C(e)    (C(e) is the clock timestamp of event e)
//
// - "logical clocks"
//   + a local variable that maps events to the positive natural numbers
//   + satisfy the "clock condition" (above)
//   + each process initializes LC to 0
//   + each sent message m contains a timestamp TS(m) == LC(send(m))
//   + update rules:
//     * LC(e_i) := {
//          LC + 1,                 (if e_i is a local or send event send(m))
//          max{LC, TS(m)} + 1      (if e_i == receive(m))
//       }
//
// - "gap-detection"
//   + the ability to determine, given two events e and e' with clock
//     values LC(e) < LC(e'), whether there exists some e'' such that
//     LC(e) < LC(e'') < LC(e')
//
// - "stable"
//   + a message m received by process p is stable if no future messages
//     with timestamps smaller than TS(m) can be received by p.
//   + NOTE: because we assume channels are reliable, stability of a message m
//     at p_0 can be guaranteed when p_0 has received at least one message from
//     ALL other processes with a timestamp greater than TS(m)
//     * because some processes may not communicate p_0 after a certain point,
//       stability of a message can be guaranteed by requesting an
//       acknowledgement from all processes to a periodic empty message, which
//       “flushes out” messages that may have been in the channels.
//
// TODO
// - "strong clock condition"
//   + e -> e' <=> TC(e) < TC(e')

// - "vector clock"
// - "hidden channels"
// - "snapshot protocols"
// - "properties of snapshots"
// - "global predicates"
// - "stable predicates"
// - "nonstable predicates"
// - "multiple monitors"

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/sfurman3/chatroom/logical"
)

type Message struct {
	SenderId  int           // process id of the sender
	timestamp logical.Clock // timestamp of the send event
}

var (
	ID                 = -1 // id of the process {0, ..., NUM_PROCS-1}
	NUM_PROCS          = -1 // total number of processes
	PORT               = -1 // number of the master-facing port
	REQUIRED_ARGUMENTS = []*int{&ID, &NUM_PROCS, &PORT}
)

func init() {
	flag.IntVar(&ID, "id", ID, "id of the process {0, ..., n-1}")
	flag.IntVar(&NUM_PROCS, "n", NUM_PROCS, "total number of processes")
	flag.IntVar(&PORT, "port", PORT, "number of the master-facing port")
	flag.Parse()

	setArgsPositional()
}

func main() {
}

// If no arguments were provided via flags, parse the first three arguments
// into ID, NUM_PROCS, and PORT respectively
func setArgsPositional() {
	getIntArg := func(i int) int {
		arg := flag.Arg(i)
		if arg == "" {
			fmt.Fprintf(os.Stderr,
				"%v: missing one or more arguments (there are %d)\n",
				os.Args, len(REQUIRED_ARGUMENTS))
			flag.PrintDefaults()
			os.Exit(1)
		}
		val, err := strconv.Atoi(arg)
		if err != nil {
			fmt.Printf("could not parse: '%v' into an integer\n", arg)
			os.Exit(1)
		}
		return val
	}

	for idx, val := range REQUIRED_ARGUMENTS {
		if *val == -1 {
			*val = getIntArg(idx)
		}
	}
}
