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

// String returns a base 10 string representation of the clock's value
func (clk *Clock) String() string {
	if clk.counter == nil {
		clk.counter = new(big.Int)
	}
	return clk.counter.String()
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

// Max sets clk to the maximum of clk or other and returns clk
func (clk *Clock) Max(other *Clock) *Clock {
	if clk.Cmp(other) < 0 {
		clk.counter.Set(other.counter)
	}
	return clk
}

// TickReceive sets the Clock to max{clk, other} + 1
func (clk *Clock) TickReceive(other *Clock) {
	clk.Max(other).Tick()
}
