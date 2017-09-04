// Logical clocks and operations
package logical

import "math/big"

var ZERO = new(big.Int)
var ONE = big.NewInt(1)

// A Clock represents a logical clock
//
// The zero value for Clock is a zeroed clock ready to use.
type Clock struct {
	counter *big.Int
}

// Value returns a text representation of the clock value in the given base
func (clk *Clock) Text(base int) string {
	if clk.counter == nil {
		clk.counter = new(big.Int)
	}
	return clk.counter.Text(base)
}

// Tick increments the Clock by 1
func (clk *Clock) Tick() {
	if clk.counter == nil {
		clk.counter = new(big.Int)
	}
	clk.counter.Add(clk.counter, ONE)
}

// Cmp compares the result of cmparing clock (clk) to another clock (other)
//
//
// The result is:
//   -1 if clk < other
//    0 if clk > other
//    1 if clk = other
func (clk *Clock) Cmp(other *Clock) int {
	if clk.counter == nil {
		clk.counter = new(big.Int)
	}
	if other.counter == nil {
		other.counter = new(big.Int)
	}
	return clk.counter.Cmp(other.counter)
}

// SetString sets the clock to the value specified in the given base, which
// must be a natural number (i.e. n >= 0), returning the clock and boolean
// indicating success
//
// If the operation fails, the clock value is unchanged
func (clk *Clock) SetString(value string, base int) (*Clock, bool) {
	newValue, succ := new(big.Int).SetString(value, base)
	if succ && newValue.Cmp(ZERO) != -1 {
		clk.counter = newValue
		return clk, true
	}
	return clk, false
}

// TickReceive sets the Clock to max{clk, other} + 1
func (clk *Clock) TickReceive(other *Clock) {
	switch clk.Cmp(other) {
	case 0, 1:
		clk.Tick()
	case -1:
		// NOTE: other.counter != nil because of the call to clk.Cmp
		clk.counter.Add(other.counter, ONE)
	}
}
