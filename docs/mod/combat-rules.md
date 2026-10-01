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

## Smaller rules

Code: `combat/classiclows.go` and hooks in `combat/model.go` (the same switch); tests
`TestClassicDeathReason`, `TestClassicRaiseDead`, `TestClassicMeleeOnFliers`, `TestClassicEntangle`.

- The end: 50 whole turns are played. Fleeing: at Intro and Easy the human loses nobody; asleep,
  confused and webbed units are lost anyway. A confused unit of the loser is lost. A unit drained
  of life does not regenerate. A recalled unit is no kill. The winner's undead get no experience.
  The items of a hero stoned or destroyed are lost. Zombie Mastery raises only the dead of normal
  damage that were not undead.
- A unit that dies of as much irreversible damage as any other is gone for good.
- A ranged shot has the ranged bonuses only: the items for shooting, and the weapon and Holy Weapon
  only for arrows and rocks, never for magic. Thrown and breath have neither, nor items.
- Invulnerability takes 2 after every defense roll, not once before (and ranged no longer twice).
- A magic shot may aim at a Magic Immune unit and meets a defense of 50.
- Rocks: the Large Shield and Weapon Immunity count against them.
- The walls' bonus counts for every kind of attack, an illusion too.
- Wall of Darkness: only True Sight shoots in.
- A unit that does not fly strikes a flier only with a ranged attack left, thrown, breath or a gaze;
  bows, rocks and magic strike it for nothing.
- A hasted magic shooter shoots twice with mana for it. Every attack, a shot too, wears the target's
  counterattack down.
- Holy Word: creatures and undead, a death roll at -2 always. Death Spell: normal damage, so the
  dead can regenerate or rise. Flame Strike, Holy Word, Death Spell and Call Lightning spare Wraith
  Form (`quirkWraithFormSpares`, the original's mistake).
- Call Lightning: 3 to 5 bolts, each try a unit by chance that must be an enemy and win a roll of 1
  in 2, up to 30 failed tries.
- Mana Leak: the wizard's reserve, not its skill; a magic shooter loses a shot.
- Entangle: a move of the turn from every unit, fliers too.
- A spell stopped by Counter Magic or a node costs the whole skill but only the range's tenth in mana
  (`quirkCounteredCheap`, the original's). Counter Magic is asked before the node.
- No skill in battle while casting the Spell of Return.
- The human pays the full cost of fixed cost spells in battle; the spellbook shows the reduced one
  (`quirkHumanNoCombatDiscount`, the original's as reconstructed).
- The human controls at most 9 units when summoning or raising the dead.
- Possession on any unit of a normal race, heroes too, not undead.
- Raise Dead: half the figures of a unit of several come back whole, a unit of one with half its
  hits; its enchantments of the world map are gone too.
- Elemental Armor in place of Resist Elements; Iron Skin in place of Stone Skin.
- Webs are torn at the start of every turn (missiles by the larger of melee and ranged, breath and
  gaze by both, others by melee); a unit that tore free acts that turn.
- Summon Demon: the demon comes in the middle of the caster's side.
- Disenchant Area and True also reach the other side's vortexes and the city's walls of fire and
  darkness (for the battle).
- The item power Death gives Death Touch.

## The last of the spells (2026-10-01)

User: "Let's update all spells to be classic". Code: `combat/classicspells.go` (the same switch)
and hooks in `combat/model.go`, `combat/combat-screen.go`, `combat/classiclows.go`, the game's
`doCombat` for the node. Read by helpers from ReMoM Combat.c Combat_Spell_Target_Screen (checked
against the program by ReMoM's review), COMBINIT.c Battle_Unit_Special_Stats, Spells131.c
Cast_Raise_Dead, Cast_Animate_Dead, Apply_Cracks_Call, Combat_Cast_Disenchant, Spells133.c
Apply_Call_Lightning, Combat.c Begin_Combat_Turn. Tests `TestClassicTargetFilter`,
`TestClassicCallLightning`, `TestClassicMetalFires`, `TestClassicRaiseDeadLionheart`. Dev:
`-capture-spell crackssquare@N`, `calllightning@N`.

- Targets of spells on a unit (`classicTargetFilter`, used by `DoTargetUnitSpell`): a buff goes on
  any unit of the side that does not have it yet (from the world map, an item or the battle) nor the
  ability it gives by nature (Cloak of Fear and Cause Fear, True Sight and Illusions Immunity,
  Invisibility, Magic Immunity, Wraith Form and Non-Corporeal, Regeneration, Immolation). No realm
  rule: undead may take Life buffs, life creatures Berserk. Iron Skin and Stone Skin each look at
  their own only. Heroism, Holy Armor, Holy Weapon, Eldritch Weapon, Flame Blade: normal units only
  (not creatures, undead, Black or Chaos Channels units), heroes too, Heroism at any level. Curses:
  refused when the unit has the curse or the immunity of its realm (Death Immunity for Black Sleep,
  Weakness, Possession; Illusions Immunity for Confusion, Vertigo, Mind Storm, Creature Binding);
  Magic Immunity, Righteousness and charms only make them resist. Possession and Creature Binding
  are never refused for having them (`quirkCurseMaskSkips`, the original's). Shatter and Possession
  for normal units, Creature Binding for the others, Web not on non-corporeal or webbed units. Mind
  Storm does nothing to a magic immune unit.
- Metal Fires: not for creatures, undead and Black Channels units, Chaos Channels units get it; no
  to-hit (shots and thrown attacks lost 10 to-hit under it before). Holy Weapon cast in the battle
  gives its +10 to-hit like the one of the world map.
- Raise Dead: the hits of a figure are taken with its enchantments and again without them: a unit
  of one figure comes back with its new hits less half its old ones (at least 1, MY CALL), several
  figures half of them whole by the new hits. Its web stays. Animate Dead clears the enchantments of
  the world map too.
- Cracks Call: any square, a unit of either side on it (invisible or merging too) unless it flies or
  is non-corporeal is swallowed one time in 4, gone for good, with no resistance; a wall on the
  square always falls. Not in the book in a battle at sea.
- Call Lightning: at the start of every round after Mana Leak, the attacker's then the defender's,
  never in the turn it was cast and whether the caster's side is alive or not. 3 to 5 bolts; a try
  picks any unit of the battle, alive or dead, and strikes it if it is an enemy, alive, has no
  Wraith Form cast in the battle (`quirkWraithFormSpares`) and wins 1 in 2; a bolt starts the tries
  anew, 30 failed in a row end it. A bolt is strength 8, armor piercing, of NATURE (the data of Wall
  of Stone): Magic Immunity stops it, Righteousness and Bless do not, Resist Elements and Elemental
  Armor count.
- Disenchant Area and True: no pass over the combat enchantments of the neutral player; the
  caster's own warped node under the battle is unwarped by a roll against Warp Node (75, no
  retorts), silently (`CombatModel.WarpedNode`, set by the game); an attacker also rolls the town's
  Heavenly Light and Cloud of Shadow (their True Light and Darkness end); a vortex on the square of a
  unit is rolled a second time.
- FOUND AND FIXED on the way: Wrack killing a unit at the start of a round crashed the battle (the
  list of units was walked while a death took a unit out of it).
