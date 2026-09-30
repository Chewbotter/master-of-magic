package ai

import (
    "testing"
)

// Get_Weighted_Choice: by weight, the first when all are 0, and weights halved at 512
func TestChewbotWeightedChoice(test *testing.T) {
    if chewWeightedChoice([]int{0, 0, 0}) != 0 {
        test.Errorf("all weights 0 does not give the first")
    }

    for range 200 {
        if choice := chewWeightedChoice([]int{0, 5, 0}); choice != 1 {
            test.Fatalf("the only weight was not picked: %v", choice)
        }
    }

    // a weight of 10 next to one of 1000 is halved to nothing: 1000 -> 500 -> 250, 10 -> 5 -> 2
    counts := make([]int, 2)
    for range 2000 {
        counts[chewWeightedChoice([]int{10, 1000})] += 1
    }
    if !quirkWeightHalving {
        return
    }
    if counts[0] == 0 || counts[0] > 60 {
        test.Errorf("the small weight was picked %v times of 2000", counts[0])
    }
}

// the objectives of the original by realm: never a Pragmatist but for a wizard without books
func TestChewbotRealmObjectives(test *testing.T) {
    for realm, weights := range chewRealmObjective {
        if weights[chewPragmatist] != 0 {
            test.Errorf("realm %v can give a Pragmatist", realm)
        }
    }
}
