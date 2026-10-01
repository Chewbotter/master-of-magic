# Rules survey: the fork's rules against the original's

User 2026-09-30: "Go ahead with the rules survey next". Every system of the game's rules compared with
the original (ReMoM, read for facts), by read-only helpers, one per system. What was already ported
in this project (the computer players, diplomacy, the world builder, the difficulty table, the quick
battle, the neutral player's turn) was left out. Each entry: what the original does, what the fork
does (file:line in `game/magic/`), and the size of the port (S small, M medium). Impact: HIGH changes
how the game plays, MED is noticeable, LOW is a detail. Things marked DOUBT need a look at the real
game or the program before they are ported (the reconstruction may be wrong there).

Nothing is changed yet. The port follows the original, mistakes included (improvements.md).

## A fix to our own port first

- DIFFICULTY UPKEEP (two surveys agree): the original takes the table's maintenance off a computer
  player's upkeep only in the summary it shows (Player_Resource_Income_Total); what it really pays is
  full, but for a cut of each unit's GOLD upkeep inside Unit_Gold_Upkeep (Hard -25%, Impossible
  -50%). Our port (`player/difficulty.go` difficultyUpkeep) takes the maintenance off real gold, food
  and mana. S.

## The top of the list (HIGH)

1. Defeated wizards are not cleaned up and keep playing: the original gives the conqueror +5 fame,
   half the gold and mana and 2 spells, turns the loser's other cities neutral (normal garrisons
   neutral, heroes and creatures dismissed), dismisses its units, frees its nodes, removes its global
   and city enchantments; the fork only sets `Defeated` and its AI keeps moving (game.go:3754). M.
2. Taking a capital: the original banishes only with 2 or more cities left, and fully defeats a
   computer wizard whose casting skill (with heroes) is under 40 or whose mana is under 1; the human
   is asked whether to resign. Banishment gives +5 fame, half the loser's mana and up to 2 of its
   spells; a banished wizard has no power income. The fork always banishes, with nothing more
   (game.go:3732, 3763). M.
3. No win or loss by conquest: the original ends the game with the lose screen when the human is
   defeated, the win screen when no computer wizard has a city left, and a score (known spells, half
   the population, 50 per wizard banished, fame x2, 2000 - 2 x turn, 250 for Mastery, times the
   difficulty; a Hall of Fame of 10). The fork has none of it; an AI's Spell of Mastery shows the
   human the winner's vortex. M.
4. Spell of Mastery costs swapped: the original's research cost is the table's, less half the
   research cost of every spell learned, and it is cast at the table's casting cost (with Runemaster);
   the fork researches at the table cost and casts at its reduced research cost of up to 60000
   (player.go:464-471, 959-970, 1264-1268; game.go:7781). And it only enters the research list when
   no other spell is left; the fork has it from the start (model.go:182). S.
5. Channeler halves all mana upkeep and Conjurer cuts a creature's upkeep to (x*3+3)/4; the fork
   has neither (player.go:1336, units/overworld.go:611). S.
6. Disbanding for lack of gold, food or mana: in the original only the human loses units (computer
   players' reserves stop at 0); never heroes or ships for gold and food, only creatures for mana;
   food counts only the cities with a surplus; from the newest unit back. The fork disbands for every
   player, heroes and ships too (game.go:7486, 7517). M.
7. Population above the maximum: the original never lets growth go below 0 and a city above its
   maximum shrinks only by starving, 1 a turn; the fork cuts it to the maximum at once
   (city/city.go:1780), so Famine or a lost Granary takes several people in one turn. S.
8. Plague: the original takes a person when Random(10) is below the people (above 2) and ends at 5% a
   turn from the 5th (about 25 turns); the fork takes one every turn from 3 people and ends in about
   10 (city.go:1738, model.go:1320). Population Boom lasts about 25 turns too (fork 10). S.
9. Melee costs half the unit's moves with no floor: a 1-move unit spends 1/2 and can attack twice; the
   fork's floor of 1 gives it one attack (combat/model.go:4029). S.
10. Stack limit of 9 when moving (own stacks, cities, Earth Gate, plane travel): the fork does not
    check it on moves (model.go:441, game.go:3817, player.go:1654). S.
11. Ship capacity: land units board only while the ships' transport covers them; after a sea battle
    riders beyond what is left drown; the fork has no count (model.go:466). S to M.
12. Heroes and mercenaries: computer players get +10 points on both offer chances and use the
    human's fame (the original's bug); Charismatic halves a hero's fee; the fork has neither
    (game.go:1913, 2023). S.
13. Merchants: only the human gets offers, always a random item (Create_Random_Item), at 3 times its
    cost; the fork sells premade items at their cost, to computer players too (game.go:2185). S.
14. Items of dead heroes (both sides, up to 18) go to the winner of the battle; the fork gives them
    back to the owner's vault (game.go:5562). M.
15. Disenchant Area/True against units and warped nodes rolls the chance out of 250 against
    `rand.N(100)`, so 40% and more always works (game/cast.go:1587, 1605). S.
16. Black Wind: every figure rolls Death resistance at -1, a failed roll kills one figure; the fork
    rolls once and kills the whole unit, and has no counter check (cast.go:552-561). S.
17. Combat spells: Psionic Blast is an Illusion attack (no defense unless immune), fork: normal
    (model.go:6491); Dispel Evil targets any enemy unit (normal ones at -4), fork: chaos and death
    creatures only (model.go:4987). S.

## Cities (MEDIUM and LOW)

- MED: rebels are taken from workers, then farmers: a city in revolt can starve; the fork keeps the
  farmers it needs (city.go:735). S.
- MED: half the food surplus goes to gold only when it is positive; the fork takes gold for a deficit
  (player.go:1288). S.
- MED: outposts: one growth roll (terrain food + race + Gaia 20 + Stream 10 + minerals, times the
  difficulty) for 1 to 3 houses, one shrink roll (5, Evil Presence +5, Pestilence, Famine, Chaos Rift
  +10 each) for 1 to 2; the fork rolls 3 and 2 times and grows about half again faster
  (city.go:1594). S.
- MED: selling: not a building another building needs; switching production off one that is needed;
  the fork sells anything (cityview/city-screen.go:799). S.
- MED: Dark Elf growth -20 (fork -10, city.go:1139). S.
- MED: Great Wasting and Armageddon set the pacification of temples and the like to -1 and -2 (the
  original's bug, which wipes them out); the fork adds 1 or 2 unrest (city.go:987). S.
- MED: Gaia's Blessing: +50% terrain food, forest and nature node production 6%, outposts +20; the
  fork adds a growth bonus and +20% farmer food and misses the node and outposts (city.go:1153,
  1286); fork bug: its desert-to-grassland turns the wrong square (city.go:1810). S.
- MED: Settlers from a city of 1 leave an outpost of 3 houses; the fork abandons the city
  (city.go:1774). S.
- MED: Evil Presence keeps religious pacification for an owner with Death books, and takes no power;
  the fork always removes pacification and zeroes religious power (city.go:612-667, 1016). S.
- LOW-MED: food in the original's order (Foresters before the limit of over-farming, Famine halving
  before it, Granary/Market/Wild Game after, wild game halved on shared squares; Halflings with an
  Animists' Guild 3, fork 4) (city.go:1264-1327). S.
- LOW-MED: production and gold as whole numbers each turn; the fork keeps fractions
  (city.go:1418-1452, 1547). S.
- LOW-MED: unrest rounding (people x unrest% first, half a unit floored). S.
- LOW: growth beyond a whole person is lost; a city of 1 never goes below 1; gold capped at 255 a
  city; neutral cities build buildings at half; no buying when it finishes in under 2 turns; mineral
  discounts halved on shared squares; plain desert 0% production (DOUBT); Consecration clears curses
  every turn (fork: corruption only). S.
- DOUBT: guild mana: the reconstruction gives the Animists' Guild +3 and not the Alchemists' (the fork
  and the manual: Alchemists' +3, Wizards' Guild -3); Dwarf taxes doubled (not in the manual).

## Units

- MED: a unit built with no better weapons gets magic weapons when its owner has Alchemy; the fork
  needs the Alchemists' Guild (game.go:7879, city.go:1660). S.
- MED: moves: Flight at least 3 then Endurance +1, Chaos Channels wings at least 2, Haste nothing
  overland, Magic Spirit 1 move; the fork doubles for Haste overland and gives the spirit 2
  (units/overworld.go:648, unit.go:1189). S.
- MED: Stream of Life does not heal; the fork heals fully in its city (game.go ~7930). S.
- MED: data: Gorgons 4 figures (fork 2); LOW: Lizardman Swordsmen 2 hits (4), Sprites melee 2 (4),
  Demon defense 5 (6), Greyfairer 5 hits (6), High Elf Settlers melee 0 (1). The unit costs of some
  creatures differ from their spell prices (Chaos Spawn 400, Werewolves 250, Guardian Spirit 50,
  Demon Lord 900, Nagas 120). S.
- LOW-MED: heroes get +1 hit a level only; the fork adds the normal units' bonus too (hero.go:1331). S.
- LOW: healing rounding; a city lists at most 12 units (without Spearmen and Swordsmen when more); the
  weakest unit is pushed off a full stack (fork: random); road time = terrain turns / construction
  points (Chaos node 5, fork 8); purifying by counters (5 turns alone, 3 with 2-3, 2 with 4); meld again
  to upgrade a Magic Spirit to a Guardian Spirit, a spirit lives on a warped node; settling on
  corruption allowed, undead settlers not; Crusade not for undead; Charm of Life +hits/4; weapon to-hit
  for melee only; War College 60 experience (fork 61). S.

## Tactical combat

- MED: First Strike with Haste strikes once; the fork gives a second strike (model.go:4190). S.
- MED: Cause Fear: only the attacker's works, and a second roll takes from its own figures (the
  original's bug); the fork: both sides, no self-penalty (model.go:4166). S.
- MED: Wall of Fire burns only an attacker from outside hitting inside, not fliers, teleporters or
  mergers; the fork burns both (model.go:4138). S.
- MED: an adjacent enemy is shot only when ranged is more than half the melee; else melee
  (model.go:2588). S.
- MED: thrown attacks roll every point; the fork one roll per figure for all or nothing
  (model.go:3580). S.
- MED: poison rolls per attacking figure; the fork once (model.go:3610). S.
- MED: a defender's to-block (Lucky, Prayer, High Prayer) also comes off the attacker's to-hit in melee
  (the original's double count); the fork: Lucky against all attacks, Prayer none (model.go:1085). S.
- MED: city damage happens whoever wins (population and buildings); the fork only on an attacker's
  win (game.go:5454). S.
- MED: ships in a city battle and riders in a sea battle stay out and die if their side loses; the
  fork lets them live (game.go:5250). M.
- LOW-MED: fleeing: asleep and confused units die too, the human attacker loses nothing on Easy, a
  defender with no room around loses those; experience also for units that died fleeing and for the
  undead raised; fame (the loser loses 1 only when it is the human above 20; a dead hero costs
  (level+1)/2, the fork's hero level is always Recruit so it costs 0; rare foe +1 only). S.
- LOW: Weapon Immunity keeps 10 against armor piercing; thrown ignores Weapon Immunity; every attack
  wears the counterattack down; ranged to-hit -10 per 3 squares, 10% at least, ranged bonuses only;
  Blur removes 1 hit in 10; Invulnerability after every defense roll; magic ranged may shoot at Magic
  Immunity; rocks are no missiles; Wall of Darkness: True Sight only; wall bonus against thrown and
  breath and illusion; death by ties irreversible first; Death Touch values; 50 turns (fork 49);
  regeneration and confusion at the end; Zombie Mastery; melee on fliers; magic ranged with Haste. S.

## Combat spells

- MED: a sleeping target takes spells at full strength, no to-hit (model.go:1637); Ice Bolt strength
  5 (fork 10, DOUBT: check the data); the distance to the fortress is the larger axis, not a straight
  line (maplib/map.go:1543); Prayer +1 resistance (fork +3, model.go:1189); Dispel Magic takes every
  curse off your own units; enemy enchantments cast overland count 5 times, Invulnerability can not be
  dispelled, only an overland Spell Lock shields; Runemaster never doubles combat dispels (the
  original's bug); Warp Creature on any enemy unit; Wall of Stone raises the wall in a city battle
  (fork: unhandled, model.go:5940); 9 units at most (summons, Raise and Animate Dead); Blur; Life Drain
  resisted as Death, the wizard gains 3 times the damage in casting skill; Wrack squared (the
  original's bug). S, Wall of Stone and dispels M.
- LOW: Disenchant Area reaches vortexes, warped nodes, city walls of fire and darkness; Holy Word -2
  always; Death Spell not irreversible; Wraith Form against Flame Strike and Call Lightning; Call
  Lightning of nature; Mana Leak drains ammunition and reserve, not skill; Entangle on fliers too; a
  countered spell costs little mana; Counter Magic before the node; no combat skill while casting Spell
  of Return; Possession of heroes; Cracks Call on a square (and walls); Raise Dead by figures;
  Elemental Armor and Iron Skin replace, not add; web; no target filters on buffs; Summon Demon's
  square. S.

## Overland magic

- MED-HIGH: the arcane rarities: tiers 1-3, 4-8, 9-12 of the arcane spells, Mastery none; the fork's
  labels make Disenchant Area common and Spell of Return very rare (spellbook/data.go:546). S.
- MED: research candidates: every knowable spell of each realm's lowest tier, picked evenly, sorted
  by turns to research (player.go:987). M.
- MED: no power at all while casting Spell of Return (model.go:1354). S.
- MED: computer wizards start knowing a summoning spell of their realm (and one more researchable from
  Hard), common picks from a fixed list (game.go:3479). S.
- MED: a spell is finished early with next turn's skill when that suffices (game.go:7754). S.
- MED: out of mana: only the human loses things; a group by chance (unit, city, global enchantments,
  creatures) (game.go:7724). S (the order within groups is unknown: TODO in the reconstruction).
- MED: Time Stop drains 200 and ends at 0; while on, events, diplomacy, Great Wasting, Meteor Storm,
  volcanoes stop; the fork drops it at 150 and runs those (game.go:7629, 8291). S.
- MED: wards block every spell of their realm aimed at a city, Consecration death and chaos; the
  counter comes after the target is picked (the spell fizzles, mana spent); Earthquake, Raise Volcano,
  Warp Node check nothing (cast.go:2202, city.go:1057). M.
- MED: Drain Power 10 rolls of 1-20; Stasis frees on 1-10 <= resistance - 5 from the second turn;
  Call the Void hits every unit for 10; Spell Binding's chance for a computer wizard by its cost (fork
  20000); Spell Lock protection and own units' Stasis for Disenchant; overland extra mana up to 100%
  (fork 4 times); computer wizards triple Disenchant and Disjunction; Enchant Item's 200 limit for
  abilities too, jewelry doubles only stats; Mastery announced for every caster; Nature's Wrath spares
  fliers and non-corporeal; hero skill at the fortress: heroes only, and not for instant casts. S.
- LOW: Subversion; Plane Shift fails on any stack there; Warp Node only enemy unwarped nodes; Raise
  Volcano's buildings; Spell Blast on Spell of Return; Cruel Unminding; Change Terrain on volcanoes;
  Corruption, Floating Island, Word of Recall, Wall of Stone checks; Move Fortress (the original moves
  the human's, its bug); Armageddon and Great Wasting 4-6 tries; Meteor Storm per caster; Nightshade
  (more buildings, the caster's multiplier); 10 books all very rare; cost and power rounding; Sage
  heroes (DOUBT); summons with no circle go to the fortress; Summon Champion's books; combat Dispel
  Magic prices overland enchantments 5 times. S.

## Heroes, items, fame

- MED: hero picks: Priestess and Paladin need life books, Black Knight and Necromancer death; 10 draws
  that may miss; fame strictly above the need (DOUBT: nobody with 0 fame gets one) (game.go:1923).
- MED: mercenaries of both planes, not Nomad Magicians or road-building Draconians, no Veteran level
  (DOUBT) (game.go:2042, 2101); Enchant Item and jewelry (above); Sage (DOUBT); non-hero casters add
  to the fortress skill in the fork (player.go:1141); fame losses of battles and dead heroes; +5 fame
  for defeating a wizard. S.
- LOW: the order of the hero offer's cap; Charismatic's check; the first mercenary turn; destroying an
  item gives half; a dismissed hero's items are lost; the heroes' starting picks (Lucky falls into
  Agility, and others: the original's bugs); Super Blademaster +6; Might not on ranged; a Town's fame
  3. S.

## Events, conquest, the turn

- MED: events start at turn 50 from 0 pressure (the fork jumps to 7% a turn); bad events hit the
  strong, good ones the weak (by the astrologer's numbers; fork: by chance); Meteor and Earthquake
  hit a city of the victim; Donation 105 to 600 (fork up to 2099); loot of a neutral city 1 to 10 a
  person, razing pays 10% of the buildings; outposts taken are always destroyed; a captured city
  builds Trade Goods. S.
- LOW: the choice of event; moons' endings; Piracy (DOUBT); Diplomatic Marriage gives the garrison;
  Rebellion's vetoes; Gift makes a new item; Depletion and New Mine unknown (stubs); the order of the
  turn; a Town's fame. S.

## Overland movement

- MED: terrain costs of the original's table (TERRSTAT.LBX, readable): nature node 2 (forester 1),
  chaos node 4 (mountaineer 1), volcano 3 (mountaineer no help), swimmers 1 on river, swamp, tundra,
  volcano, desert and water (model.go:536). S.
- MED: the original's costs are set at the start and only roads change them (a raised or cooled
  volcano keeps its cost: the original's bug). S.
- MED: a stack with a Wind Walker or a ship moves at that unit's pace; a multi-turn path stops short
  of enemies and lairs (no attack, no question); new land is explored only by moving (and new
  outposts, an Oracle, towers), not by standing sight; fliers pay nothing on roads (DOUBT). S, the
  exploring M.
- LOW: non-corporeal stacks pay 1/2 everywhere; walls give sight 3, an Oracle beats Nature's Eye, patrol
  +1, towers see both planes; no move onto the polar rows; refusing a lair still costs the moves; no
  extra cost for paths through the unknown; road and purify times; meld; settling; plane travel. S.
- DOUBT: the table gives mountaineers 3 moves on grassland.

## Doubts to settle before porting (the reconstruction may be wrong)

Guild mana; Dwarf taxes; plain desert; Ice Bolt 5; Sage research; hero offers at 0 fame; the
mercenary Veteran level; flyers on roads; mountaineers on grass; ship capacity details; Wind Walking
mode; Piracy; Depletion and New Mine; the damage to a city's buildings (read literally it destroys
85%); human wizards' discounts in combat; mass healing; Pick_Random_City; Great Unsummoning and
Death Wish (stubs).

## Suggested order

1. The fix to our difficulty port (upkeep).
2. The end of a wizard and of the game: defeat, banishment, win and loss, score (1-3 above).
   DONE 2026-09-30, see conquest.md.
3. The economy of mana and units: Channeler, Conjurer, disbanding, the Spell of Mastery's costs,
   research, arcane rarities (4-6, the magic MEDs).
   DONE 2026-09-30, see economy.md (what is left is at the end of improvements.md).
4. Cities: population cap, plague and boom, rebels, the smaller city rules.
   DONE 2026-09-30, see cities.md (what is left is at the end of improvements.md).
5. Movement: stack limit, ships, terrain table, Wind Walking, go-to.
   DONE 2026-09-30, see movement.md (what is left is at the end of improvements.md).
6. Combat: melee cost, then the MEDs of combat and combat spells.
   DONE 2026-09-30 for the HIGH and the MEDs, see combat-rules.md (what is left is at the end of
   improvements.md).
7. Heroes, items, merchants, mercenaries, fame.
   DONE 2026-09-30, see heroes.md (what is left is at the end of improvements.md).
8. Events and conquest details.
   DONE 2026-09-30, see events.md (what is left is at the end of improvements.md).
9. The LOWs by system.
