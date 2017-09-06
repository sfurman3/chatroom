// Vector clocks and operations
//
// TODO: Define string representation: [v1, v2, v3, ...]
package vector

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sfurman3/chatroom/logical"
)

// TODO
type ClockBuilder interface {
	Length(int) ClockBuilder
	Id(int) ClockBuilder
	Build() (*Clock, error)
}

// TODO
type Clock struct {
	id     int             // id of the process {0, ..., NUM_PROCS-1}
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

// implementation of ClockBuilder
type clockBuilder struct {
	id     int
	length int
}

// TODO
func (cb *clockBuilder) Length(n int) ClockBuilder {
	cb.length = n
	return cb
}

// TODO
func (cb *clockBuilder) Id(id int) ClockBuilder {
	cb.id = id
	return cb
}

// TODO
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

// TODO
func NewClock() ClockBuilder {
	return new(clockBuilder)
}

// Id returns the id of the process that owns the clock
func (clk *Clock) Id() int {
	return clk.id
}

// TODO
func (clk *Clock) Length() int {
	return len(clk.vector)
}

// TODO
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

// TODO
func (clk *Clock) Timestamp(base int) *Timestamp {
	vector := make([]string, len(clk.vector))
	for i, val := range clk.vector {
		vector[i] = val.Text(base)
	}
	return &Timestamp{
		Id:     clk.id,
		Vector: vector,
	}
}

// VectorClock returns a pointer to a new Clock with the value of the given
// Timestamp
//
//  Entries in the Vector field are interpreted in the given base
//  If conversion fails, the returned Clock is undefined
func (ts *Timestamp) VectorClock(base int) (*Clock, error) {
	if !(1 <= ts.Id && ts.Id <= len(ts.Vector)) {
		return nil, fmt.Errorf(
			"vector clock JSON does not satisfy: 1 <= id (%d) <= length (%d)",
			ts.Id, len(ts.Vector))
	}

	clk := new(Clock)
	clk.id = ts.Id
	clk.vector = make([]logical.Clock, len(ts.Vector))
	for i, val := range ts.Vector {
		_, succ := clk.vector[i].SetString(val, base)
		if !succ {
			errMsg := "could not parse: %s into a base %d vector " +
				"clock component (must be a nonnegative integer value)"
			return clk, fmt.Errorf(errMsg, val, base)
		}
	}

	return clk, nil
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
		return fmt.Errorf(
			"vector clock JSON does not satisfy: 1 <= id (%d) <= length (%d)",
			ts.Id, len(ts.Vector))
	}

	clk.id = ts.Id
	clk.vector = make([]logical.Clock, len(ts.Vector))
	for i, val := range ts.Vector {
		_, succ := clk.vector[i].SetString(val, logical.MaxBase)
		if !succ {
			errMsg := "could not parse: %s into a base %d vector " +
				"clock component (must be a nonnegative integer value)"
			return fmt.Errorf(errMsg, val, logical.MaxBase)
		}
	}

	return nil
}

// TickLocal increments the local component of clk (i.e. clk[clk.Id()-1])
func (clk *Clock) TickLocal() {
	if clk.Length() == 0 {
		return
	}
	clk.vector[clk.id-1].Tick()
}

// TODO: TEST
// NOTE: Returns an error if clk.ErrComparableTo(other) != nil, in which case
// clk and other are unmodified
func (clk *Clock) TickReceive(other *Clock) error {
	err := clk.ErrComparableTo(other)
	if err != nil {
		return err
	}

	// increment the local component of clk
	clk.vector[clk.id-1].Tick()

	// update the remaining components
	for i := range clk.vector[:clk.id-1] {
		clk.vector[i].Max(&other.vector[i])
	}
	for i := range clk.vector[clk.id:clk.Length()] {
		clk.vector[i].Max(&other.vector[i])
	}

	return err
}

// Equal returns whether the clock values of clk and other are equal (ignoring ids)
//
//  Returns true if both have length 0 (uninitialized)
//  Returns false if the lengths are unequal
//  Makes no assumptions about the value of clk.ErrComparableTo(other)
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

// NOTE: Returns false if clk.ErrComparableTo(other) != nil
// NOTE: Does not consider clock ids
func (clk *Clock) LessThan(other *Clock) bool {
	if clk.ErrComparableTo(other) != nil {
		return false
	}
	if clk.Equal(other) {
		return false
	}

	for i := 0; i < clk.Length(); i++ {
		if clk.vector[i].Cmp(&other.vector[i]) == 1 {
			return false
		}
	}
	return true
}

// NOTE: Does not consider clock ids
func (clk *Clock) ErrComparableTo(other *Clock) error {
	if clk.Length() == 0 {
		return errors.New("vector clock unitialized (length 0)")
	}
	if other.Length() != clk.Length() {
		return fmt.Errorf(
			"vector clocks have different lengths (%d != %d)",
			clk.Length(), other.Length())
	}
	if !(clk.vector[clk.id-1].Cmp(&other.vector[clk.id-1]) == -1) {
		return errors.New(
			"clk[clk.Id()-1] MUST be greater than other[clk.Id()-1]")
	}
	return nil
}
