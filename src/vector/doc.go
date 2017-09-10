// TODO: Cite the paper: "Consistent Global States of Distributed Systems": by
// Özalp Babaoğlu and Keith Marzullo.

// TODO: REORGANIZE
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
//    1. local events
//    2. send(m)         (send events)
//    3. receive(m)      (receive events)
//
//    NOTE: monitoring does not generate any new events
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
// TODO: IMPORTANT
// - "observation" (of a distributed computation by a "monitor" process)
//   + a permutation of a subset of a run, which can thus be consistent OR
//     inconsistent
//   + a "consistent observation" MUST correspond to a consistent run
//   + observations are constructed by monitors from events received from
//     processes in a system
//   + NOTE: we can use consistent observations to construct consistent global
//     states
//     * because consistent observations correspond to consistent runs,
//       consistent runs can be used to construct consistent cuts, and
//       consistent cuts have frontiers that correspond to consistent global
//       states
//
// - "deliver"
//   + the _act_ of presenting received messages to a process
//   + a "delivery rule" is a rule for when to deliver received messages
//
// - "causal delivery"
//   + send_i(m) -> send_j(m) => receive_i(m) -> receive_j(m)
//   + the property of delivering messages in an order that preserves causal
//     precedence (i.e. that yields consistent observations)
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
package vector
