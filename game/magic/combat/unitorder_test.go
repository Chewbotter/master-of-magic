package combat

import (
    "image"
    "testing"
)

func orderPlaces(units []*ArmyUnit) []image.Point {
    var out []image.Point
    for _, unit := range units {
        out = append(out, image.Pt(unit.X, unit.Y))
    }
    return out
}

func samePlaces(a []image.Point, b []image.Point) bool {
    if len(a) != len(b) {
        return false
    }
    for index := range a {
        if a[index] != b[index] {
            return false
        }
    }
    return true
}

// like the days of a calendar: along the row, then the row below
func TestUnitsInOrder(test *testing.T) {
    units := []*ArmyUnit{
        {X: 11, Y: 18}, {X: 10, Y: 17}, {X: 12, Y: 17}, {X: 9, Y: 18}, {X: 11, Y: 17}, {X: 10, Y: 19},
    }

    all := orderPlaces(unitsInOrder(units, image.Point{}, false))
    expected := []image.Point{{10, 17}, {11, 17}, {12, 17}, {9, 18}, {11, 18}, {10, 19}}
    if !samePlaces(all, expected) {
        test.Errorf("from the first: %v", all)
    }

    // after the second of the first row: the rest, then around to the start
    after := orderPlaces(unitsInOrder(units, image.Pt(11, 17), true))
    expected = []image.Point{{12, 17}, {9, 18}, {11, 18}, {10, 19}, {10, 17}, {11, 17}}
    if !samePlaces(after, expected) {
        test.Errorf("after 11,17: %v", after)
    }

    // after a place nobody stands on any more
    after = orderPlaces(unitsInOrder(units, image.Pt(10, 18), true))
    expected = []image.Point{{11, 18}, {10, 19}, {10, 17}, {11, 17}, {12, 17}, {9, 18}}
    if !samePlaces(after, expected) {
        test.Errorf("after 10,18: %v", after)
    }

    // after the last: from the first
    after = orderPlaces(unitsInOrder(units, image.Pt(10, 19), true))
    expected = []image.Point{{10, 17}, {11, 17}, {12, 17}, {9, 18}, {11, 18}, {10, 19}}
    if !samePlaces(after, expected) {
        test.Errorf("after the last: %v", after)
    }
}
