package set

import (
    "fmt"
    "strings"
)

// The set type is a generic set implementation that uses a map as the underlying data structure.
// Its Values() come in the order the values were inserted (a value inserted again keeps its
// place): a range over a map goes in an order Go picks by chance, which a run of one seed of the
// game (-sim-seed) could not repeat.

type setEntry[T comparable] struct {
    value T
    alive bool
}

type Set[T comparable] struct {
    // the place of each value in order
    data map[T]int
    order []setEntry[T]
    // removed places still in order
    gone int
}

func MakeSet[T comparable]() *Set[T] {
    return &Set[T]{
        data: make(map[T]int),
    }
}

func NewSet[T comparable](values ...T) *Set[T] {
    set := MakeSet[T]()
    for _, value := range values {
        set.Insert(value)
    }
    return set
}

func (set *Set[T]) Clone() *Set[T] {
    newSet := MakeSet[T]()
    for _, value := range set.Values() {
        newSet.Insert(value)
    }
    return newSet
}

func (set *Set[T]) Insert(v T){
    if _, ok := set.data[v]; ok {
        return
    }
    set.data[v] = len(set.order)
    set.order = append(set.order, setEntry[T]{value: v, alive: true})
}

func (set *Set[T]) InsertMany(values ...T) {
    for _, value := range values {
        set.Insert(value)
    }
}

func (set *Set[T]) Clear() {
    set.data = make(map[T]int)
    set.order = nil
    set.gone = 0
}

func (set *Set[T]) Contains(v T) bool {
    _, ok := set.data[v]
    return ok
}

func (set *Set[T]) IsEmpty() bool {
    return len(set.data) == 0
}

func (set *Set[T]) Size() int {
    return len(set.data)
}

func (set *Set[T]) Remove(v T) {
    place, ok := set.data[v]
    if !ok {
        return
    }
    delete(set.data, v)
    set.order[place].alive = false
    set.gone += 1
    if set.gone > 32 && set.gone * 2 > len(set.order) {
        set.compact()
    }
}

// the removed places out of order
func (set *Set[T]) compact() {
    kept := make([]setEntry[T], 0, len(set.data))
    for _, entry := range set.order {
        if entry.alive {
            set.data[entry.value] = len(kept)
            kept = append(kept, entry)
        }
    }
    set.order = kept
    set.gone = 0
}

func (set *Set[T]) RemoveMany(values ...T) {
    for _, value := range values {
        set.Remove(value)
    }
}

// FIXME: turn this into an iterator
func (set *Set[T]) Values() []T {
    if set == nil {
        return nil
    }

    out := make([]T, 0, len(set.data))
    for _, entry := range set.order {
        if entry.alive {
            out = append(out, entry.value)
        }
    }
    return out
}

func (set *Set[T]) String() string {
    parts := make([]string, 0, len(set.data))
    for _, v := range set.Values() {
        parts = append(parts, fmt.Sprintf("%v", v))
    }

    return "{" + strings.Join(parts, ", ") + "}"
}
