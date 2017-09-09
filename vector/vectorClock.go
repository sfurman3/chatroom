// Package vector implements vector clocks and operations, types for converting
// vector clock timestamps and messages to and from JSON, and a message
// receptacle type for storing received messages and removing them in an order
// that respects causal delivery (which enables monitors to build consistent
// observations and consistent global states for evaluating evaluating global
// predicates).
package vector

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sfurman3/chatroom/logical"
)

// ClockBuilder is an object for creating vector clocks that follows the
// builder pattern
type ClockBuilder interface {
	Length(int) ClockBuilder
	Id(int) ClockBuilder
	Build() (*Clock, error)
}

// Clock represents a vector clock with an integer ID and a length n array of
// counters (where n is the total number of processes in a distributed system,
// excluding the monitor process)
//
// Clocks can be created either with a ClockBuilder (creates a zeroed clock) or
// the ClockBase method of Timestamp (creates a vector clock with a specific
// initial value)
//
// NOTE: Vector clocks are indexed STARTING AT 1. Thus, after the first local
// increment of a vector clock with an ID of 1, the clock should be "[1, 0, 0]"
//
// See the String and MarshalJSON methods for information about the
// corresponding Clock representations
type Clock struct {
	id     int             // ID of the process {0, ..., NUM_PROCS-1}
	vector []logical.Clock // clock value
}

// A Timestamp represents the state of a Clock and can easily be marshaled into
// JSON
//
// The Vector field can be an array of strings in any base from 2 to
// logical.MaxBase
type Timestamp struct {
	Id     int      `json:"id"`
	Vector []string `json:"v"`
}

// Message represents a value to be sent or received
//
// The timestamp should correspond to the vector clock value of the send event
type Message struct {
	Content   string    `json:"msg"` // content of message
	Timestamp Timestamp `json:"ts"`  // Timestamp message was SENT
}

// implementation of ClockBuilder
type clockBuilder struct {
	id     int
	length int
}

// NewMessage returns a new Message with the given message and timestamp
// corresponding to the state of the given Clock
//
// NOTE: A send is an event, so the clock should be incremented before
// generating a message
func NewMessage(msg string, clk *Clock) Message {
	return Message{
		Content:   msg,
		Timestamp: clk.Timestamp(logical.MaxBase),
	}
}

// MessageReceptacle represents a set of messages received but not yet
// delivered to a monitor
//
// Receptacles can achieve causal delivery (i.e. presenting messages to a
// process in an order that preserves causal precedence) and can thus be used
// by monitors to build consistent observations and consistent global states
// for evaluating evaluating global predicates
type MessageReceptacle struct {
	counter  []logical.Clock
	received map[*Message]*Clock
}

// Returns a new MessageReceptacle of length n (i.e. for a distributed system
// of n processes)
//
// Returns nil if n < 0
func NewMessageReceptacle(n int) *MessageReceptacle {
	if n < 0 {
		return nil
	}
	rcp := new(MessageReceptacle)
	rcp.counter = make([]logical.Clock, n)
	rcp.received = make(map[*Message]*Clock)
	return rcp
}

// Receive adds a message to the receptacle's set of received messages, after
// which the provided message (and its fields) should not be modified
//
// For instance, you SHOULD NOT unmarshal the bytes of a message into the same
// message struct as this will overwrite the value stored in the receptacle
//
// Returns an error if the message's timestamp does not have the same length as
// the message receptacle, the message does not have a valid timestamp, or the
// message was already received (and not yet delivered)
//
// If an error is returned, the message is not added
//
// NOTE: In order for a receptacle to provide causal delivery, processes MUST
// only increment the local component of their vector clocks for events that
// are notified to the monitor (i.e. sends and local events but NOT receives)
func (rcp *MessageReceptacle) Receive(msg *Message) error {
	if rcp.Length() != len(msg.Timestamp.Vector) {
		return fmt.Errorf("message timestamp length (%d) != receptacle "+
			"length (%d) : ", len(msg.Timestamp.Vector), rcp.Length())
	}

	ts, err := msg.Timestamp.ClockBase(logical.MaxBase)
	if err != nil {
		return err
	}
	_, isPresent := rcp.received[msg]
	if isPresent {
		return fmt.Errorf("message already received: %v", msg)
	}
	rcp.received[msg] = ts
	return nil
}

// Size returns the number of messages stored in rcp, which corresponds to the
// number of received messages that have not been delivered
func (rcp *MessageReceptacle) Size() int {
	return len(rcp.received)
}

// Length returns the length of a message receptacle (the number of processes
// in the system)
func (rcp *MessageReceptacle) Length() int {
	return len(rcp.counter)
}

// Deliverables returns any messages in the receptacle that are ready to be
// delivered (i.e. the message can be safely passed to a process since all
// messages that causally precede it have already been delivered) in order of
// causal precedence (relative ordering is not defined for concurrent events)
//
// Returns an empty slice if no messages in the receptacle are deliverable
//
// Returns an error and the offending message if rcp cannot be delivered
// because of an inconsistency with the receptacle's counter. Otherwise both
// are nil.
//
// NOTE: If an error is encountered, the returned slice may still contain
// deliverable messages, so DON'T THROW IT AWAY!
//
// NOTE: Deliverables may check all received messages so it has a worst
// case time complexity of O(n). Try to avoid calling it often.
func (rcp *MessageReceptacle) Deliverables() ([]*Message, error, *Message) {
	var delivery []*Message
	for msg, ts := range rcp.received {
		err, offender := rcp.deliver(msg, ts, &delivery)
		if err != nil {
			return delivery, err, offender
		}
	}
	return delivery, nil, nil
}

// deliver determines if msg (whose timestamp is ts) is deliverable and, if so,
// appends it to delivery, updates the receptacle counter, and removes the
// message from the receptacle's set of received messages
//
// Returns an error and the offending message if rcp cannot be updated because
// ts is inconsistent with the value of its counter (i.e. rcp.counter[ts.id-1]
// < ts.vector[ts.id-1]), in which case the message is not added to delivery
func (rcp *MessageReceptacle) deliver(
	msg *Message, ts *Clock, delivery *[]*Message) (error, *Message) {

	id := ts.id
	noUndeliveredFromProcess :=
		rcp.counter[id-1].CmpOffset(+1, &ts.vector[id-1]) == 0
	noPriorFromOtherProcesses := true
	for oIdx, ctr := range rcp.counter {
		oId := oIdx + 1
		hasGap := oId != id && ctr.Cmp(&ts.vector[oIdx]) < 0
		if hasGap {
			noPriorFromOtherProcesses = false
			break
		}
	}
	if noUndeliveredFromProcess && noPriorFromOtherProcesses {
		if rcp.counter[ts.id-1].Cmp(&ts.vector[ts.id-1]) > 0 {
			delete(rcp.received, msg)
			errMsg := "failed to deliver message because" +
				" timestamp[%d] (%s) < receptacle[%d] (%s): %v"
			return fmt.Errorf(errMsg, ts.id-1, ts.vector[ts.id-1],
				ts.id-1, rcp.counter[ts.id-1], msg), msg
		}
		rcp.counter[ts.id-1].Set(&ts.vector[ts.id-1])
		*delivery = append(*delivery, msg)
		delete(rcp.received, msg)
	}

	return nil, nil
}

// Length sets the length of the ClockBuilder
func (cb *clockBuilder) Length(n int) ClockBuilder {
	cb.length = n
	return cb
}

// Id sets the ID of the ClockBuilder
func (cb *clockBuilder) Id(id int) ClockBuilder {
	cb.id = id
	return cb
}

// Build returns a new Clock initialized to all zeros (with values set to those
// in the ClockBuilder)
func (cb *clockBuilder) Build() (*Clock, error) {
	var clk *Clock
	if 1 <= cb.id && cb.id <= cb.length {
		clk = new(Clock)
		clk.id = cb.id
		clk.vector = make([]logical.Clock, cb.length)
		return clk, nil
	}
	return clk, fmt.Errorf(
		"vector clock does not satisfy: 1 <= id (%d) <= length (%d)",
		cb.id, cb.length)
}

// NewClockBuilder returns a new ClockBuilder
func NewClockBuilder() ClockBuilder {
	return new(clockBuilder)
}

// Id returns the id of the process that owns the clock
func (clk *Clock) Id() int {
	return clk.id
}

// Length returns the length of a clock (the number of processes in the system)
func (clk *Clock) Length() int {
	return len(clk.vector)
}

// String implements the Stringer interface
//
// Clocks are represented as a comma-delimited array of integers whose length
// is equal to that of the Clock itself (IDs are excluded)
func (clk *Clock) String() string {
	if clk == nil {
		return "<nil>"
	}
	if clk.Length() == 0 {
		return "[]"
	}

	var buffer bytes.Buffer
	buffer.WriteString("[")
	for _, val := range clk.vector[:clk.Length()-1] {
		buffer.WriteString(val.String())
		buffer.WriteString(", ")
	}
	buffer.WriteString(clk.vector[clk.Length()-1].String())
	buffer.WriteString("]")
	return buffer.String()
}

// Timestamp returns the Timestamp corresponding to the current state of clk
func (clk *Clock) Timestamp(base int) Timestamp {
	vector := make([]string, len(clk.vector))
	for i, val := range clk.vector {
		vector[i] = val.Text(base)
	}
	return Timestamp{
		Id:     clk.id,
		Vector: vector,
	}
}

// ClockBase returns a pointer to a new Clock with the value of the given
// Timestamp
//
//  Entries in the Vector field are interpreted in the given base
//  If conversion fails, the returned Clock is undefined
func (ts *Timestamp) ClockBase(base int) (*Clock, error) {
	if !(1 <= ts.Id && ts.Id <= len(ts.Vector)) {
		return nil, fmt.Errorf("timestamp vector does not satisfy: "+
			"1 <= id (%d) <= length (%d)", ts.Id, len(ts.Vector))
	}

	clk := new(Clock)
	clk.id = ts.Id
	clk.vector = make([]logical.Clock, len(ts.Vector))
	for i, val := range ts.Vector {
		_, succ := clk.vector[i].SetString(val, base)
		if !succ {
			errMsg := "could not parse: %s into a base %d vector " +
				"clock component (must be a nonnegative" +
				" integer value)"
			return clk, fmt.Errorf(errMsg, val, base)
		}
	}

	return clk, nil
}

// Clock returns the Clock corresponding to a timestamp of a message serialized
// with json.Marshal
//
// ts.Clock() is equivalent to ts.ClockBase(logical.MaxBase)
func (ts *Timestamp) Clock() (*Clock, error) {
	return ts.ClockBase(logical.MaxBase)
}

// MarshalJSON implements the json.Marshaler interface
func (clk *Clock) MarshalJSON() ([]byte, error) {
	return json.Marshal(clk.Timestamp(logical.MaxBase))
}

// UnmarshalJSON implements the json.Unmarshaler interface
//
// clk is undefined on failure
func (clk *Clock) UnmarshalJSON(jsonBytes []byte) error {
	var ts Timestamp
	err := json.Unmarshal(jsonBytes, &ts)
	if err != nil {
		return err
	}

	if !(1 <= ts.Id && ts.Id <= len(ts.Vector)) {
		return fmt.Errorf("vector clock JSON does not satisfy: 1 "+
			"<= id (%d) <= length (%d)", ts.Id, len(ts.Vector))
	}

	clk.id = ts.Id
	clk.vector = make([]logical.Clock, len(ts.Vector))
	for i, val := range ts.Vector {
		_, succ := clk.vector[i].SetString(val, logical.MaxBase)
		if !succ {
			errMsg := "could not parse: %s into a base %d" +
				" vector clock component (must be a " +
				"nonnegative integer value)"
			return fmt.Errorf(errMsg, val, logical.MaxBase)
		}
	}

	return nil
}

// TickLocal increments the local component of clk (i.e. clk[clk.Id()-1])
//
// Before every send event this function should be called and the new timestamp
// attached to the outgoing message
func (clk *Clock) TickLocal() {
	if clk.Length() == 0 {
		return
	}
	clk.vector[clk.id-1].Tick()
}

// TickReceive updates the clk for a message received from another process (by
// setting each non-local component to the maximum of the two clocks at that
// index). This function should be called for every receive event and the NEW
// timestamp attached to any receive event generated.
//
//  clk[i] = max{clk[i], other[i]}	(for all i != clk.id-1)
//
// NOTE: Returns an error if clk.ErrComparableTo(other) != nil or clk and other
// are pairwise inconsistent, in which case clk and other are unmodified
func (clk *Clock) TickReceive(other *Clock) error {
	err := clk.ErrComparableTo(other)
	if err != nil {
		return err
	}
	if clk.PairwiseInconsistent(other) {
		return errors.New("clocks are pairwise inconsistent: " +
			clk.String() + ", " + other.String())
	}

	// update the non-local components
	for i := range clk.vector {
		if i != clk.id-1 {
			clk.vector[i].Max(&other.vector[i])
		}
	}

	return err
}

// Equal returns whether the clock values of clk and other are equal (ignoring IDs)
//
// Returns false if clk and other have different lengths and true if both are
// uninitialized (length 0)
func (clk *Clock) Equal(other *Clock) bool {
	clockLen := clk.Length()
	otherLen := other.Length()
	if clockLen == 0 && clockLen == otherLen {
		return true
	}
	if clockLen != otherLen {
		return false
	}
	for i := 0; i < clockLen; i++ {
		if clk.vector[i].Cmp(&clk.vector[i]) != 0 {
			return false
		}
	}
	return true
}

// LessThan returns whether clk's value is less than that of other, which is
// equivalent to saying that an event that occurs at timestamp clk "happens
// before" or "causally precedes" an event that occurs at timestamp other
//
// Returns false if clk.ErrComparableTo(other) != nil or clk and other are
// pairwise inconsistent
//
// NOTE: Not all components are compared, ensuring O(1) complexity. This means
// that, for instance, clocks from the same process are only compared by their
// local components
func (clk *Clock) LessThan(other *Clock) bool {
	if clk.ErrComparableTo(other) != nil {
		return false
	}
	if clk.id == other.id {
		return clk.vector[clk.id-1].Cmp(&other.vector[clk.id-1]) < 0
	}
	if clk.PairwiseInconsistent(other) {
		return false
	}

	return clk.vector[clk.id-1].Cmp(&other.vector[clk.id-1]) <= 0
}

// Concurrent returns whether the current states of clk and other correspond to
// the timestamps of concurrent events (i.e. those for which there is no causal
// precedence relationship)
//
// Returns false for clocks from the same process (trivial)
// Returns false if clk.ErrComparableTo(other) != nil or clk and other are
// pairwise inconsistent
//
// NOTE: Requires that each clock has been ticked at least once (which
// naturally occurs at the time of the first event)
func (clk *Clock) Concurrent(other *Clock) bool {
	if clk.id == other.id {
		return false
	}
	if clk.ErrComparableTo(other) != nil {
		return false
	}
	if clk.PairwiseInconsistent(other) {
		return false
	}

	return clk.vector[clk.id-1].Cmp(&other.vector[clk.id-1]) > 0 &&
		other.vector[other.id-1].Cmp(&clk.vector[other.id-1]) > 0
}

// PairwiseInconsistent returns true if clk and other are pairwise inconsistent
// (the states of the two clocks denote impossible causal precedence such as a
// send happening before a receive (i.e. clk[clk.Id()-1] < other[clk.Id()-1]
// OR other[other.Id()-1] < clk[other.Id()-1]))
//
// Assumes clk.ErrComparableTo(other) == nil and that clock IDs are different
func (clk *Clock) PairwiseInconsistent(other *Clock) bool {
	return clk.vector[clk.id-1].Cmp(&other.vector[clk.id-1]) < 0 ||
		other.vector[other.id-1].Cmp(&clk.vector[other.id-1]) < 0
}

// ErrComparableTo returns a descriptive error if clk or other have different
// lengths OR if either is unitialized (i.e. has a length of 0). Otherwise nil
// is returned and the two clocks are safe for comparison (though may still be
// pairwise inconsistent)
func (clk *Clock) ErrComparableTo(other *Clock) error {
	if clk.Length() == 0 {
		return errors.New("vector clock unitialized (length 0)")
	}
	if other.Length() != clk.Length() {
		return fmt.Errorf(
			"vector clocks have different lengths (%d != %d)",
			clk.Length(), other.Length())
	}
	return nil
}
