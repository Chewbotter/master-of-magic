# Cities

The original's rules of cities. Read from the ReMoM reconstruction (`MoM/src/CITYCALC.c`,
`NEXTTURN.c` Apply_City_Changes, `EVENTS.c`, `CityScr.c`, `RACETYPE.c`, `Terrain.c`,
`Spells130.c`; the unrest of races from TERRSTAT.LBX). City_Growth_Rate, City_Maximum_Size, the
food of the land, the growth of outposts, Apply_City_Changes and the events are marked in the
reconstruction as checked against the program. Code: `city/classiccity.go`, switch
`citylib.ClassicCities`, dev flag `-classic-cities=false`. Tests `TestClassicApplyGrowth`,
`TestClassicGrowthAtMaximum` and upstream's city tests (their values follow the switch,
`classicOr`).

## People

- Maximum size: the food of the land (full on own squares, half on shared ones, half again with
  Gaia's Blessing, rounded down once), halved by Famine, +2 Granary, +3 Farmers' Market, + wild
  game (2 a square, 1 shared). The city screen shows at most 25.
- Growth a turn, in tens of people: a city exactly at its maximum does not change, even starving
  (the original's). Starving: 50 people less for every food missing. Else half the gap to the
  maximum plus 1, rounded down, plus the race (Barbarians +2, Lizardmen +1, Draconians, Gnolls,
  Klackons, Nomads -1, Dark Elves, Dwarves, High Elves, Trolls -2, the rest 0), +2 Granary, +3
  Farmers' Market; none at 25 people; Stream of Life and Population Boom each twice; then a percent:
  Dark Rituals -25, and while building Housing the share of workers (50 for a city of 1), Sawmill
  +10, Builders' Hall +15; neutral towns half and none from (difficulty + 1) x 2 people, computer
  wizards times the difficulty; never below 0.
- A whole person more when the growth reaches 1000 (what is over is lost), one person less below 0
  (the rest carried), one a turn at most; a city of 1 never starves away (it is set to 50 people);
  neutral towns stop at 8. A city above its maximum is never cut down: it shrinks only by starving.
- Pestilence: a person less when the people are more than a roll of 1 to 10. Plague: a person
  less when a roll of 1 to 10 is below the people and they are more than 2; it ends in a city under
  2 (set to 2). Plague and Population Boom end one time in 20 from the 5th turn.
- Settlers built in a city of 1 destroy it.

## Rebels

- Unrest percent: the race of the city under the race of the capital (the original's table, tens of
  percent; a Klackon city under a Klackon capital -20), + the tax (0, 10, 20, 30, 45, 60, 75),
  + 25 Famine. Rebels are the people times that, rounded down.
- Less: Shrine, Temple, Parthenon, Cathedral 1 each, half again with Divine or Infernal Power
  (none of them under Evil Presence unless the owner has death books); Oracle 2; Animists' Guild
  1; Just Cause 1; Gaia's Blessing 2. Another wizard's Great Wasting or Armageddon SETS all of that
  to -1 or -2 (`quirkWastingSetsPacification`). Dark Rituals, Cursed Lands 1 more, Pestilence 2.
  Only when that is still below the rebels: the garrison, half its normal units and heroes.
- Stream of Life: no rebels. Never more rebels than people.
- Rebels come from the workers first, then from the farmers, even below the farmers the city needs:
  a city in revolt can starve.

## Food

2 a farmer (3 for Halflings or with an Animists' Guild, not both), +2 Foresters' Guild, halved by
Famine, over the food of the land only half counts, then +2 Granary, +3 Farmers' Market, + wild
game. The farmers a city needs are counted the same way. Food over what the people eat sells for
gold (economy.md).

## Production

Workers 2 each (Dwarves, Klackons 3), farmers half each, the sum rounded up; times 100 percent +
the land (mountains, chaos nodes 5; forests, hills, deserts 3, forests and nature nodes 6 with
Gaia's Blessing; shared squares half, rounded down) + Foresters' Guild 25, Sawmill 25, Miners'
Guild 50, Mechanicians' Guild 50, Inspirations 100; rounded down once; Cursed Lands half. Neutral
towns put half of it into buildings. Unit costs: iron 5 (10 with a Miners' Guild), coal 10 (20),
Dwarves twice, shared half, at most 50 percent off.

## Gold and research

Taxes: the people who are not rebels times half the tax step. Minerals: silver 2, gold 3, gems 5,
Dwarves twice, Miners' Guild half again, shared half. Both together times one percent: 100 + the
city's square (river 20, sea 10) + trade by road (a connected city's people, half for the same
race) + Nomads 50, all that at most 3 per person, + Merchants' Guild 100, Bank 50, Marketplace 50,
Prosperity 100. Trade Goods: half the production. At most 255 a city. Research: Library 2, Sages'
Guild 3, University 5, Wizards' Guild 8.

## Outposts

One roll to grow (land food + the race's rate + 20 Gaia's Blessing + 10 Stream of Life + 5 for iron
or silver and 10 for another mineral of the area; times the difficulty, the human's too) by 1 to 3
houses; one roll to shrink (5, Evil Presence +5, Pestilence, Famine, Chaos Rift +10 each) by 1 or
2. At 10 houses it is a city of 1, at 0 it is gone.

## Selling and buying

A building another building of the city needs can not be sold ("You cannot sell back the X because
it is required by the Y."). Selling one that what the city produces needs warns ("Selling back your
X will cease production of your Y.") and the city turns to Housing. One a turn; a third of the
cost, City Walls half that. Buying is not possible when it would be done in under 2 turns anyway.

## Consecration

Every turn it ends Chaos Rift, Evil Presence, Cursed Lands, Famine and Pestilence on the city, and
clears corruption on a block of 4 by 4 squares from 2 up and left of it
(`quirkConsecrationBlock`).

## Added with the smaller rules

- The plain inner desert (one of its four pictures) gives no production; a nature node gives 6
  with Gaia's Blessing (`quirkPlainDesertBarren`, `classicSquareProduction`).
