package set

import (
    "testing"
)

func TestSet(test *testing.T){
    s := MakeSet[int]()

    s.Insert(2)
    s.Insert(8)

    if !s.Contains(2) {
        test.Errorf("Set should contain 2")
    }

    if s.Contains(5) {
        test.Errorf("Set should not contain 5")
    }

    if s.Size() != 2 {
        test.Errorf("Set should have size 2")
    }

    s.Remove(2)

    if s.Contains(2) {
        test.Errorf("Set should not contain 2")
    }

    if s.Size() != 1 {
        test.Errorf("Set should have size 1")
    }

    s = NewSet(1, 2, 3)

    if s.Size() != 3 {
        test.Errorf("Set should have size 3")
    }

    s.RemoveMany(1, 3)

    if s.Size() != 1 {
        test.Errorf("Set should have size 1")
    }

    if s.Contains(1) || s.Contains(3) {
        test.Errorf("Set should not contain removed element")
    }

    if !s.Contains(2) {
        test.Errorf("Set should still contain 2")
    }
}

func TestValuesInInsertOrder(test *testing.T) {
    set := MakeSet[int]()
    for _, value := range []int{5, 3, 9, 1, 3} {
        set.Insert(value)
    }
    set.Remove(9)
    set.Insert(7)
    values := set.Values()
    want := []int{5, 3, 1, 7}
    if len(values) != len(want) {
        test.Fatalf("values %v, want %v", values, want)
    }
    for i := range want {
        if values[i] != want[i] {
            test.Fatalf("values %v, want %v", values, want)
        }
    }
    for i := range 100 {
        set.Insert(100 + i)
    }
    for i := range 90 {
        set.Remove(100 + i)
    }
    if set.Size() != 14 || set.Values()[4] != 190 {
        test.Errorf("after removals: size %v, values %v", set.Size(), set.Values())
    }
}
