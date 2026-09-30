# Combat AI of Chewbot

The setting "Enemy AI" picks the AI of the computer players: "Clone original" is the AI the clone
came with (its author made it up, he did not know the original's), "Chewbot" is ours: first the
original game's AI ported as closely as it can be, then changed from there (user, 2026-09-29).

This file is the combat AI of Chewbot in words: what the original does, where the fork differs,
which of the original's mistakes are kept. The facts were read from the ReMoM reconstruction
(`MoM/src/CMBTAI.c`, `MoM/src/Combat.c`); the code is ours, `game/magic/combat/aichewbot.go`.
The player's army on auto plays by the same rules, as in the original.

Rule for the original's mistakes (user, 2026-09-29): "visible quirks are good, since we're
rebuilding this anyway we don't need to force the internal quirks over unnecessarily". A mistake
that changes how the AI plays is kept, behind a switch (`quirk...`); one that only wastes work
inside the routine is left out.

## The turn of a side

When a side's turn starts (Auto_Do_Combat_Turn):

1. Neutral attackers (raiders, monsters) that found no way to move in more than 3 turns die.
2. A wizard's army may flee (AI_Retreat_Check, asked by Check_For_Winner): not in the first turn,
   not when a wizard's city is under siege, only when the enemy is more than 4 times as strong
   (player mode 0), and then only if it has a hero to save or only builders (settlers,
   engineers) left.
3. The walls it sees: stone walls, wall of fire, wall of darkness of the city.
4. A defender keeps to its walls until the attacker has a ranged strength above 30 while its own
   is 0, or the attacker has a magic vortex, Wrack, Call Lightning, or Mana Leak (against a
   wizard). Then it leaves them for the rest of the battle.
5. Every unit gets a mode (AI_Set_All_Action_Modes): shoot if it has a ranged attack (missile,
   boulder or magic, with shots left), else fight. By the player mode (below) and the difference
   of ranged power (ranged strength of the side plus a quarter of its casters' mana, against the
   enemy's), with a random offset of -3 to -1, some become "advance and fight" or "advance and
   shoot".
   - modes 0, 1 (outnumbered badly): fighters advance when the side's ranged power beats the
     enemy's by more than the offset and the side has any.
   - mode 2: fighters advance as above, if the enemy is more than twice as strong as the side's
     fighters; shooters advance when the side's fighters are at least half the enemy and its
     ranged power is not less (not a defender behind walls).
   - mode 3 (about even): fighters advance when the side's ranged power is higher, shooters when
     it is lower.
   - modes 4, 5 (stronger): shooters advance one time in two.
   THE MODES ONLY DECIDE THE STAGE POINT: every unit decides again when it acts (AI_Set_Unit_
   Action_Mode sets "shoot" or "fight"), so no unit of the original ever advances and shoots.
6. The stage point, when the side's ranged strength is at least the enemy's: the fighters sorted
   slowest first, the one in the middle; if it advances, where it stands; else the farthest cell
   it reaches this turn on its way to its target of the turn before (in the first turn there is
   none). Fighters do not move past it in the direction they move, so they gather there.
   A defender outside its walls ignores it.

Player mode (AI_Player_Mode) compares the effective strength of both sides: 0 the enemy is more
than 4 times as strong, 1 more than 3 times, 2 at least 1.5 times, 3 about even, 4 the side more
than 3 times, 5 more than 4 times. After the first turn a unit of a computer counts only while it
has a target.

Effective strength (Effective_Battle_Unit_Strength): melee threat plus hits plus ranged threat.
- An attack of strength s by f figures: s above 3 is f * 20 * (s - 2); 1, 2, 3 are f * 40 / 7,
  / 3, * 2 / 3. Thrown and breath the same with 30 and 60.
- Armor piercing * 5/4 (doom damage * 2 instead), poison, life steal, destruction, stoning touch,
  death touch + f * 60 each, illusion * 5, first strike and eldritch weapon * 5/4 each (melee).
  A gaze + 300, three gazes (Chaos Spawn) + 900 more.
- Hits h with defense d: 3h, times ((d - 2)^2 / 2) above 3, else * 1/2, 3/5, 7/10, 4/5 for 0 to 3.

## The turn of a unit

A unit picks its target (Choose_Target_And_Action) by value (Target_Unit_Value):
- attack minus the target's defense against it (Battle_Unit_Defense_Special: large shield +2
  against ranged, immunities make it 50, weapon immunity at least 10, bless +3 and righteousness
  against chaos and death, elemental armor +10 or resist elements +3 against chaos and nature
  ranged, armor piercing halves, an illusion attack makes it 0)
- a target with a ranged attack + its ranged / 3 + 2; a confused one - 10
- a target that can hardly hurt it (its defense / 2 above the target's best attack) - 10, but at
  a node a defender wants spirits that meld (+ 3)
- its target from before + 10
- attackers without flight, noncorporeal, teleport or merging: a target on the wall - 20, the
  gate + 5, one in the four middle cells of the city - 20; behind a wall of fire - 20
- a sleeping target + 20; a target it can reach this turn + 25
- a flying target it cannot reach at all - 200
Ranged units skip invisible targets and, behind a wall of darkness, targets inside the city; a
defender inside its walls only shoots at what is inside or next to it. Heroes do not go into
melee with a target whose defense is above their attack, or whose attack (or thrown or breath)
is more than 1.5 times their defense. Ranged ties within 3 go to the weaker target, melee ties to
the healthier one. A missile unit against missile immunity fights instead if its melee * 3 is at
least its ranged * 2. A shooter with no target fights. The target is then moved to a closer unit
of the same kind and defense, and a fighter joins what a unit before it in the army goes for.

Then it acts (AI_Execute_Unit_Action): a shooter shoots until its target is gone or its moves
are; a fighter walks toward its target within the stage box and attacks when it gets there, over
and over while it makes progress. With stone walls and no target, attackers and defenders go to
the gate. An attacker that cannot get to its target (inside the walls) first goes to the nearest
cell of the city it can get to, which is the gate. A unit that teleports or merges jumps to the
cell before its target. Outsiders do not enter the city but by the gate; a defender that keeps
to walls of fire or darkness does not leave them.

Heroes act twice: after all other units, a hero with moves left acts again, and when no unit of a
computer player moved this turn it forgets its caution for that second turn.

## The original's mistakes that are kept (visible)

- `quirkRangedStrengthOfLastUnit`: the ranged strength of a side is the one of its LAST unit with
  a ranged attack, not the sum (Total_Ranged_Attack_Strength).
- `quirkHealingWastesTurn`: a unit with the Healing ability that sees a hurt friend picks it
  once, and the original has nothing to do for it: the unit does nothing that turn.
- The advance modes only move the stage point (see 5).
- The stage point follows the targets of the turn before, none in the first turn.
- Wall of darkness counts only when the city has it, not when it was cast in the battle.

## Left out (internal)

- Paths built twice, the stage point on a unit that has no target, and the random walk of the
  index of the first melee unit when a side has no fighters.
- A defender inside stone walls that walks out is drawn without its walk in the original; here it
  walks.

## Not the original's yet, and differences of the fork

- SPELLS: the spells of units (casters, Doom Bolt, Fireball, Web, Healing, Summon Demon, items)
  are cast as the clone casts them (one time in five, any charge), and the wizard's own spells too
  (`doAiCast`). The original's combat spell AI (CMBMAGIC.c, AITP_Combat_Spell) is the next part.
- Movement costs and how far a unit gets are the fork's (`StepCost`, `CanFollowPath`), not the
  original's cost map.
- A retreat is decided when the side's turn starts; the original can decide it after any action.
- The wall of fire is weighed as "does the attacker take damage from fire at all" where the
  original computes the damage of a Fireball on the attacker.
- The fork's grid is larger than the original's; the facts of cells (city, gate, middle) are
  mapped with `toCell` (see combat-overhaul.md).

## Checking it

- `-capture-ai-log` writes every plan and every unit's decision to the log.
- Random battle on auto: `bash dev.sh a -capture-random-battle -capture-same-battle -capture-auto
  -capture-ai-log -capture probe/x.png -capture-frames 4000`.
- A walled city: `-capture-city-battle "walls" -capture-auto -capture-ai-log`.
- Tests: `TestChewbotStrengthFormulas`, `TestChewbotCellsMapBack`, `TestChewbotBoxStopsAtStagePoint`,
  `TestChewbotHeroWaitsOnce`.
