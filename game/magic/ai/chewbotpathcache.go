package ai

// The original's store of the computer players' paths (ReMoM UnitMove.c Make_Move_Path,
// Cache_AI_Move_Path, Invalidate_AI_Move_Path; UNITSTK.c Stack_Move_To). After a move the rest of
// the path, from the square the stack stopped on to its destination, is kept under those two
// squares; the next search from that square to that destination takes the kept path as it is. So an
// army keeps to the way it chose. Without it every turn searched anew, and a way that units of
// others blocked one turn and not the next sent an army back and forth (two Swordsmen of a test run
// lost 20 turns so, 2026-10-01). A kept path is dropped only when a move along it does not get off
// its first square; the table holds 140 and none are added while it is full
// (quirkPathStoreFills); a path is kept only when its destination is within the 35 squares of the
// original's arrays. The search for a kept path asks no more than the squares: a path of a flier is
// taken by walkers from the same square to the same destination, whose move then fails (kept, as
// the original).
//
// ReMoM's rebuild jumps over the lookup ("HACK: hard-coded MovePath Cache Miss"); its notes of the
// program have it, and the notes win. One table for all computer wizards of a game, as the
// original's (whose neutral player uses it too; Chewbot's neutral player searches its own).

import (
    "image"

    "github.com/kazzmir/master-of-magic/game/magic/data"
    "github.com/kazzmir/master-of-magic/game/magic/pathfinding"
    playerlib "github.com/kazzmir/master-of-magic/game/magic/player"
)

// the store of paths is used (false: every path is searched anew, as before)
var ChewbotPathStore = true

const (
    chewPathStoreSize = 140
    // the arrays of a kept path: the stack's square and up to 34 more
    chewPathStoreLength = 35
    // the original frees a slot only when a move along a kept path fails, so in a long game the
    // table fills and no more paths are kept
    quirkPathStoreFills = true
)

type chewKeptPath struct {
    From image.Point
    To image.Point
    Plane data.Plane
    // the squares after From, the last one To; nil for a slot that was dropped
    Path pathfinding.Path
}

// the table of a game, shared by its computer wizards
type chewPathTable struct {
    Kept []chewKeptPath
}

// the table of the game an AI plays: one at a time in a process (a new game makes a new one)
var chewSharedPaths struct {
    services playerlib.AIServices
    table *chewPathTable
}

func chewPathTableOf(services playerlib.AIServices) *chewPathTable {
    if chewSharedPaths.table == nil || chewSharedPaths.services != services {
        chewSharedPaths.services = services
        chewSharedPaths.table = &chewPathTable{}
    }
    return chewSharedPaths.table
}

// a wizard's part: what it moved and which kept paths its moves were given
type chewPathStore struct {
    table *chewPathTable
    // the stacks that moved and the rest of their paths, kept when the wizard plans its next turn
    // (Cache_AI_Move_Path at the end of Stack_Move_To)
    moved []chewMovedStack
    // the kept paths that moves of this turn were given, to drop when a move does not get off its
    // square
    used []chewUsedPath
}

type chewMovedStack struct {
    Stack *playerlib.UnitStack
    Rest pathfinding.Path
}

type chewUsedPath struct {
    Group []chewUnitKey
    From image.Point
    Plane data.Plane
    Index int
}

// a kept path from a square to a destination: its index, or -1
func (store *chewPathStore) find(from image.Point, to image.Point, plane data.Plane) int {
    found := -1
    for index, kept := range store.table.Kept {
        if kept.Path != nil && kept.From == from && kept.To == to && kept.Plane == plane {
            found = index
        }
    }
    return found
}

// the paths of the stacks that moved in the last turn are kept (the end of Stack_Move_To)
func (store *chewPathStore) keepMoved() {
    for _, moved := range store.moved {
        stack := moved.Stack
        if stack == nil || stack.IsEmpty() || len(moved.Rest) == 0 || len(moved.Rest) >= chewPathStoreLength {
            continue
        }
        from := image.Pt(stack.X(), stack.Y())
        to := moved.Rest[len(moved.Rest) - 1]
        if store.find(from, to, stack.Plane()) >= 0 {
            continue
        }
        kept := chewKeptPath{From: from, To: to, Plane: stack.Plane(), Path: append(pathfinding.Path(nil), moved.Rest...)}
        // a move that followed a kept path writes the rest into the same slot (the slot the
        // lookup found is the one Cache_AI_Move_Path fills), so an army on its way takes one;
        // else the first dropped slot
        free := store.slotOf(stack)
        for index, old := range store.table.Kept {
            if free >= 0 {
                break
            }
            if old.Path == nil {
                free = index
            }
        }
        if free >= 0 {
            store.table.Kept[free] = kept
        } else if len(store.table.Kept) < chewPathStoreSize || !quirkPathStoreFills {
            store.table.Kept = append(store.table.Kept, kept)
        }
    }
    store.moved = nil
    store.used = nil
}

// the slot of the kept path a stack's move was given, -1 when it searched
func (store *chewPathStore) slotOf(stack *playerlib.UnitStack) int {
    for _, used := range store.used {
        if used.Index >= len(store.table.Kept) {
            continue
        }
        for _, unit := range stack.Units() {
            for _, key := range used.Group {
                if key == chewKey(unit) {
                    return used.Index
                }
            }
        }
    }
    return -1
}

// the rest of a stack's path after a step (MovedStack); the last one of the turn is kept
func (store *chewPathStore) noteMoved(stack *playerlib.UnitStack, rest pathfinding.Path) {
    for index := range store.moved {
        if store.moved[index].Stack == stack {
            store.moved[index].Rest = append(pathfinding.Path(nil), rest...)
            return
        }
    }
    store.moved = append(store.moved, chewMovedStack{Stack: stack, Rest: append(pathfinding.Path(nil), rest...)})
}

// a kept path was given to the move of a group
func (store *chewPathStore) noteUsed(group []chewUnitKey, from image.Point, plane data.Plane, index int) {
    store.used = append(store.used, chewUsedPath{Group: group, From: from, Plane: plane, Index: index})
}

// a move that failed: the kept path it was given is dropped when the stack did not get off its
// square (Stack_Move_To: Move_Failed, Invalidate_AI_Move_Path)
func (store *chewPathStore) moveFailed(stack *playerlib.UnitStack) {
    if store.table == nil || stack.IsEmpty() {
        return
    }
    first := chewKey(stack.Units()[0])
    for _, used := range store.used {
        if used.Index >= len(store.table.Kept) || image.Pt(stack.X(), stack.Y()) != used.From || stack.Plane() != used.Plane {
            continue
        }
        for _, key := range used.Group {
            if key == first {
                store.table.Kept[used.Index] = chewKeptPath{}
                break
            }
        }
    }
}
