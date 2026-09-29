package combat

import (
    "math/rand/v2"
    "testing"
)

func TestDamageNumbersKeepApart(test *testing.T) {
    random := rand.New(rand.NewPCG(1, 2))
    pick := func(count int) int { return random.IntN(count) }

    // a volley: numbers on one unit, a few ticks apart
    var numbers []DamageIndicator
    for index := range 8 {
        for place := range numbers {
            numbers[place].Count += 3
        }

        number := DamageIndicator{X: 10, Y: 12, Damage: index, Width: 5, Height: 6}
        placeDamageNumber(&number, numbers, pick)
        numbers = append(numbers, number)
    }

    check := func(when string) {
        for first := range numbers {
            for second := first + 1; second < len(numbers); second++ {
                if numbers[first].touches(&numbers[second]) {
                    test.Errorf("%v: numbers %v and %v touch", when, first, second)
                }
            }
        }
    }

    check("placed")

    // they rise together, so they stay apart
    for index := range numbers {
        numbers[index].Count += 40
    }
    check("risen")
}
