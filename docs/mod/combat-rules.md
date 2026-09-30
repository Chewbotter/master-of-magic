# Rules of tactical combat

The original's rules of tactical combat and combat spells, as far as ported. Read from the ReMoM
reconstruction (`MoM/src/Combat.c` Battle_Unit_Attack, Battle_Unit_Attack_Target,
Battle_Unit_Process_Attack, Apply_Fear_Attack, Check_Wall_Of_Fire_Attack,
Battle_Unit_Defense_Special, End_Of_Combat; `COMBINIT.c` Battle_Unit_Special_Stats; `CMBMAGIC.c`
Combat_Casting_Cost_Multiplier; `Spells131.c` Combat_Cast_Dispel; `Spells133.c` Apply_Life_Drain,
Apply_Wrack, Apply_Warp_Creature; SPELLDAT.LBX). Code: `combat/classicrules.go` and the places it
is hooked into (`combat/model.go`, `combat/combat-screen.go`, `combat/ai.go`, `game/game.go` for
the end of a battle). Switch `combat.ClassicRules`, dev flag `-classic-combat=false`. Tests
`TestClassicMeleeCost`, `TestClassicFirstStrikeHaste`, `TestClassicShootsAdjacent` and upstream's
combat tests. The quick resolution of battles nobody watches is in ai-combat.md.

## Attacks

- A melee attack costs half the unit's full moves, rounded up in half moves, and nothing floors it:
  a unit of 1 move pays half a move and can strike twice. (Upstream paid at least 1 and then paid
  again for the wall.) Any move left allows an attack.
- A ranged unit next to an enemy shoots only when its ranged attack is more than half its melee;
  else it strikes.
- A natural 10 always hits: no attack hits on fewer than 10 percent.
- First Strike with Haste strikes once.
- Cause Fear: only the attacker's works; its feared figures of the defender do not strike back. A
  second roll of the defender's figures is taken off the attacker's own striking figures
  (`quirkFearHitsOwnFigures`, the original's).
- Wall of Fire burns only the attacker, when it strikes from outside the wall at a unit inside;
  fliers, teleporters and mergers never.
- Lucky, Prayer and High Prayer each give +10 to-block; in melee the defender's to-block also comes
  off the attacker's to-hit (`quirkToBlockDoubleCount`, the original's).
- Blur takes each point of damage one time in 10, before the defense roll; the loop shrinks with it
  and an illusion-immune defender loses its own side's Blur (`quirkBlurLoop`, the original's). It
  no longer lowers the to-hit.
- Ranged to-hit: -10 for every full 3 squares, Long Range at most -10, magic none.
- Thrown attacks roll every point of every figure; poison rolls for every attacking figure.
- Weapon Immunity keeps a defense of 10 after armor piercing; thrown attacks never meet it (the
  original's).

## The end of a battle

- A city fought over is damaged whoever wins: each person but one is lost at twice the collateral
  damage percent (+10 if the attacker won, +50 for raiders), at most 50; each building at the
  collateral damage percent (+10 or +50 the same way), at most 75.
- The items of every dead hero, of both sides, go to the winner, at most 18.
- The loser's ships left out of a city's battle and its riders left out of a sea battle die with it.

## Combat spells

- Ice Bolt: strength 5 (SPELLDAT) and a point for every extra mana.
- Psionic Blast: strength 5 and half the extra mana, an illusion: no defense but for the
  illusion-immune, nothing against magic immunity.
- Dispel Evil on any enemy unit (`quirkDispelEvilAnyTarget`, the original's).
- Warp Creature on any enemy unit.
- Damage spells on a sleeping unit do their full strength.
- Prayer: +1 resistance; High Prayer +3 in its place.
- Dispel Magic: the curses on one's own units always go; an enemy's enchantments cast on the world
  map count 5 times their cost; Invulnerability is never dispelled; only a Spell Lock cast on the
  world map shields; Runemaster does not double dispels in battle (`quirkNoRunemasterDispel`, the
  original's).
- Life Drain: resisted as Death, 1 for every 5 extra mana, no spell saving items; a wizard puts three
  times the damage into its casting skill.
- Wrack: a unit with n figures failing takes n times n (`quirkWrackSquared`, the original's);
  immunity to death and Wraith Form spare it.
- The cost of a spell by the distance to the fortress: the larger of the two distances.
