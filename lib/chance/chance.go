// Package chance is math/rand/v2 with a seed: the game imports it under the name rand, so every
// call (rand.N, rand.IntN, rand.Perm...) stays as it is. Unseeded it is math/rand/v2 itself. Seed
// puts every call on one generator behind a lock, so a run without a window (-sim-seed) draws the
// same numbers in the same order. Go's own top-level functions can not be seeded.
package chance

import (
    "fmt"
    "io"
    "math/rand/v2"
    "runtime"
    "slices"
    "strings"
    "sync"
)

type Rand = rand.Rand
type Source = rand.Source
type PCG = rand.PCG
type ChaCha8 = rand.ChaCha8

func New(source Source) *Rand {
    return rand.New(source)
}

func NewPCG(seed1 uint64, seed2 uint64) *PCG {
    return rand.NewPCG(seed1, seed2)
}

var lock sync.Mutex
var seeded *rand.Rand
var draws uint64

// every draw from now on comes from one generator of this seed
func Seed(seed uint64) {
    lock.Lock()
    defer lock.Unlock()
    seeded = rand.New(rand.NewPCG(seed, seed ^ 0x9e3779b97f4a7c15))
    draws = 0
}

// the draws since Seed: two runs of one seed that differ in it have gone separate ways
func Draws() uint64 {
    lock.Lock()
    defer lock.Unlock()
    return draws
}

// development: every draw of a seeded run writes the line of the game that made it, up to a limit,
// so two runs of one seed show where they part (a loop over a map, whose order Go picks by chance)
var traceTo io.Writer
var traceLeft int

func TraceTo(writer io.Writer, limit int) {
    lock.Lock()
    defer lock.Unlock()
    traceTo = writer
    traceLeft = limit
}

// the generator of a draw: nil when unseeded
func take(argument int64) *rand.Rand {
    if seeded == nil {
        return nil
    }
    draws += 1
    if traceTo != nil && traceLeft > 0 {
        traceLeft -= 1
        // the caller of the function of this package that called take
        _, file, line, ok := runtime.Caller(2)
        if ok {
            // and the line that called that, for helpers such as a game's own roll
            _, file2, line2, ok2 := runtime.Caller(3)
            if ok2 {
                fmt.Fprintf(traceTo, "%v:%v %v < %v:%v\n", file, line, argument, file2, line2)
            } else {
                fmt.Fprintf(traceTo, "%v:%v %v\n", file, line, argument)
            }
        }
    }
    return seeded
}

type integer interface {
    ~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr
}

// a number in [0, n), as rand.N
func N[Int integer](n Int) Int {
    lock.Lock()
    source := take(int64(n))
    if source == nil {
        lock.Unlock()
        return rand.N(n)
    }
    defer lock.Unlock()
    if n <= 0 {
        panic("invalid argument to N")
    }
    return Int(source.Uint64N(uint64(n)))
}

func IntN(n int) int {
    lock.Lock()
    source := take(int64(n))
    if source == nil {
        lock.Unlock()
        return rand.IntN(n)
    }
    defer lock.Unlock()
    return source.IntN(n)
}

func Int() int {
    lock.Lock()
    source := take(-1)
    if source == nil {
        lock.Unlock()
        return rand.Int()
    }
    defer lock.Unlock()
    return source.Int()
}

func Int64N(n int64) int64 {
    lock.Lock()
    source := take(n)
    if source == nil {
        lock.Unlock()
        return rand.Int64N(n)
    }
    defer lock.Unlock()
    return source.Int64N(n)
}

func Float64() float64 {
    lock.Lock()
    source := take(-1)
    if source == nil {
        lock.Unlock()
        return rand.Float64()
    }
    defer lock.Unlock()
    return source.Float64()
}

func Float32() float32 {
    lock.Lock()
    source := take(-1)
    if source == nil {
        lock.Unlock()
        return rand.Float32()
    }
    defer lock.Unlock()
    return source.Float32()
}

func NormFloat64() float64 {
    lock.Lock()
    source := take(-1)
    if source == nil {
        lock.Unlock()
        return rand.NormFloat64()
    }
    defer lock.Unlock()
    return source.NormFloat64()
}

func Uint64() uint64 {
    lock.Lock()
    source := take(-1)
    if source == nil {
        lock.Unlock()
        return rand.Uint64()
    }
    defer lock.Unlock()
    return source.Uint64()
}

func Uint32() uint32 {
    lock.Lock()
    source := take(-1)
    if source == nil {
        lock.Unlock()
        return rand.Uint32()
    }
    defer lock.Unlock()
    return source.Uint32()
}

func Perm(n int) []int {
    lock.Lock()
    source := take(int64(n))
    if source == nil {
        lock.Unlock()
        return rand.Perm(n)
    }
    defer lock.Unlock()
    return source.Perm(n)
}

func Shuffle(n int, swap func(i, j int)) {
    lock.Lock()
    source := take(int64(n))
    if source == nil {
        lock.Unlock()
        rand.Shuffle(n, swap)
        return
    }
    // the swaps are made without the lock held, so a swap may draw again
    order := source.Perm(n)
    lock.Unlock()
    // the permutation as swaps: the element at i goes where order says
    position := make([]int, n)
    at := make([]int, n)
    for i := range n {
        position[i] = i
        at[i] = i
    }
    for target := range n {
        want := order[target]
        from := position[want]
        if from != target {
            swap(target, from)
            moved := at[target]
            at[target], at[from] = want, moved
            position[want], position[moved] = target, from
        }
    }
}

// the keys of a map in one order, by their text: a range over a map goes in an order Go picks by
// chance, which a seeded run can not repeat. For a pick from a map of weights and the like
func SortedKeys[Key comparable, Value any](values map[Key]Value) []Key {
    keys := make([]Key, 0, len(values))
    for key := range values {
        keys = append(keys, key)
    }
    slices.SortFunc(keys, func(a Key, b Key) int {
        return strings.Compare(fmt.Sprint(a), fmt.Sprint(b))
    })
    return keys
}
