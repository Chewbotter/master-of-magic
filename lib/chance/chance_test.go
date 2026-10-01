package chance

import (
    "slices"
    "testing"
)

func draw() []int {
    var out []int
    for range 20 {
        out = append(out, N(1000))
    }
    out = append(out, Perm(8)...)
    values := []int{0, 1, 2, 3, 4, 5, 6, 7}
    Shuffle(len(values), func(i int, j int) {
        values[i], values[j] = values[j], values[i]
    })
    return append(out, values...)
}

func TestSeedRepeats(test *testing.T) {
    Seed(42)
    first := draw()
    Seed(42)
    second := draw()
    if !slices.Equal(first, second) {
        test.Errorf("one seed gave two sequences: %v and %v", first, second)
    }
    Seed(43)
    if slices.Equal(first, draw()) {
        test.Errorf("two seeds gave one sequence")
    }
}

func TestShuffleIsAPermutation(test *testing.T) {
    Seed(7)
    for size := range 12 {
        values := make([]int, size)
        for i := range values {
            values[i] = i
        }
        Shuffle(size, func(i int, j int) {
            values[i], values[j] = values[j], values[i]
        })
        sorted := slices.Clone(values)
        slices.Sort(sorted)
        for i := range sorted {
            if sorted[i] != i {
                test.Fatalf("shuffle of %v lost a value: %v", size, values)
            }
        }
    }
}

func TestSortedKeys(test *testing.T) {
    keys := SortedKeys(map[string]int{"c": 1, "a": 2, "b": 3})
    if !slices.Equal(keys, []string{"a", "b", "c"}) {
        test.Errorf("keys out of order: %v", keys)
    }
}
