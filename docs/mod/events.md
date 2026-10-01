# Random events and conquest details

The original's rules. Read from the ReMoM reconstruction (`MoM/src/EVENTS.c` Determine_Event,
Event_Twiddle, Get_Event_Victim, Pick_Random_City; `INITGAME.c`; `CITYCALC.c` City_Gold;
`CityScr.c` Change_City_Ownership). Code: `game/eventsclassic.go` (switch `ClassicEvents`), hooks in
`GameModel.DoRandomEvents` (`game/model.go`) and `defeatCity` (`game/game.go`); dev flag
`-classic-events=false`. Tests `TestClassicEventPressure`, `TestClassicEventVictim`,
`TestClassicEventEndsAndDonation`.

## When an event comes

- The pressure is the number of turns since the last event; before the first event it is counted
  from turn 50, so nothing comes before turn 51.
- For 5 turns after an event, and for the first 5 turns of a session, the pressure is 0.
- The difficulty scales it: Intro 1/2, Easy 2/3, Average 3/4, Hard 4/5, Impossible all of it.
- An event comes when a roll of 1 to 512 is not above the pressure: 37 in 512 a turn after 50 quiet
  turns at Average.

## Which event, and who it hits

- Up to 5 tries. Each try picks one of the 18 events by chance, then its victim.
- The victim: each wizard (not the neutral player, not a defeated one) weighs its army, magic and
  research of the astrologer's numbers. A bad event (Meteor, Earthquake, Piracy, Plague, Rebellion,
  Depletion, Mana Short) hits the strong by that weight; a good one hits the weak by 600 less the
  weight (never below 0). The weights are halved until they sum to 500 or less.
- A try is thrown away when:
  - the victim is banished, or an event of the kind is running;
  - it is a moon, conjunction or mana short while one of those is running, or a good moon during a
    bad one and the other way around;
  - it is Piracy and the victim has under 100 gold;
  - a city is needed (Meteor, Marriage, Earthquake, Plague, Rebellion, Population Boom) and the
    victim has none whose race is content under the race of its capital (an outpost never counts);
  - it is Marriage and the neutral player has no such town;
  - it is Marriage or Meteor before turn 150;
  - it is Rebellion in a city with a fortress, a hero, or more summoned than normal units;
  - it is Disjunction and nobody has a global enchantment;
  - it is Plague in a city with a Population Boom, or the other way around.
- After 5 thrown tries, no event this turn.

## What changes in the events

What an event does is upstream's, but for these:

- Meteor, Earthquake, Plague, Population Boom, Rebellion hit the city that was picked.
- Donation: 105 to 600 gold, in steps of 5.
- Piracy: 30% to 50% of the gold, rounded down to tens.
- Gift: only the human gets it, a new random item as a lair's.
- Diplomatic Marriage: the neutral town comes with its garrison and builds Trade Goods.
- A moon, a conjunction or a mana short ends from its 5th turn, with a chance of (turns - 3) in 20.

## Taking and razing cities

- A neutral town gives 1 to 10 gold for every person.
- An outpost that is taken is always destroyed; nobody is asked.
- Razing pays a tenth of the cost of every building that stood.
- A city that is taken builds Trade Goods.
- The loser's gold never goes below 0.
