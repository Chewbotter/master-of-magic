# Diplomacy and personalities

The original game's diplomacy, ported from the ReMoM reconstruction (MoM/src/DIPLOMAC.c and the
functions named in the code). The fork's own diplomacy was taken out: its hostility levels, its
turn updates of relations and its screens. It applies under both Enemy AI settings (Clone
original and Chewbot).

Code: `game/magic/relations/` (the rules), `game/magic/diplomacy/` (the screen and the messages),
`game/magic/game/relations.go` (what the rules need from the world, the turn, the screens).

## Personality and objective

Every computer wizard gets both at the start of a game (`relations.PickPersonality`, the original's
Init_Magic_Personalities_Objectives): weights by the realm of most books, more for some retorts.
Life is never Maniacal, Death never Peaceful, nobody is a Pragmatist. The personality gives a bonus
in every judgment of a treaty (Maniacal 0, Ruthless 10, Aggressive 20, Chaotic 30, Lawful 40,
Peaceful 50) and a warlike bonus in the choice of war. The objective steers what Chewbot's cities
build. A save from before personalities gets them picked on load (a Pragmatist is one never
picked).

## What a pair of wizards keeps

`a.PlayerRelations[b]` is a's view of b: the treaty (none, pact, alliance, war), the visible
relation (-100 Hate to 100 Harmony, the same both ways, the eyes of the gargoyles), the hidden trust
(broken treaties lower it for good, gifts raise it), the relation it drifts back to, the patience
for treaties, exchanges and peace (every proposal wears it down, every turn some comes back), the
turns of a peace, the hostility (0, 2 hostile, 3 war, 4 holy war), the warnings given, the treaty
broken last, and the grievance of the turn. In the human's view of a computer wizard the grievance
is what that wizard says to the human at the end of the turn.

## What changes relations

Battles against a wizard's troops or cities, conquered and razed cities, curses on cities, global
enchantments (each wizard minds them by its books), the Spell of Mastery, units standing near its
cities, an empire that grows too large, the human's army being larger, treaties (goodwill every
turn), broken treaties (as the original has it, the other way round: the breaker's trust in its
victim rises, every other wizard trusts the victim a little less). Bad news counts twice against a good relation; without an alliance the relation stays at
most 65.

## The turn

After every player had its turn: hostility is looked at again (from turn 100, every 16 to 25
turns), peace runs out, relations grow and drift, new acquaintances are noticed, the computer
wizards talk to each other (pacts, alliances, exchanges of spells, peace, wars for a reason, wars of
a superior wizard against a weaker one from the third level), then each decides what to say to the
human, then the patience comes back.

What a computer wizard says to the human: a greeting by its personality the turn after they met;
when it is angry enough, a warning about the grievance, then the treaty broken over it, or war; at
war now and then an offer of peace; when it likes the human, a proposal of a pact or an alliance,
sometimes with gold or a spell; a warning to take units away from its cities.

## The screens

What the wizards say comes as screens at the start of the human's turn, one per wizard, in the
original's words (DIPLOMSG.LBX, read from the game's data). A proposal comes after a greeting, with
Accept and Reject; a wizard at war with an ally of the human first asks the human to end that
alliance (Agree, Forget It). The first meeting opens no screen at once any more: the greeting comes
at the end of that turn, as in the original.

The talk the human opens (the magic screen): the wizard greets, grants a cool audience, or does not
talk at all when trust and patience are gone. Then:

- Propose Treaty: pact (relation above 10), alliance (above 50), peace (at war). The wizard judges
  by patience, trust, relation, personality, a roll, the threat of the human's army on its home
  land and the price of the treaty. A close answer is taken one time in 2 (the original's
  counteroffer is not reconstructed). War on another wizard and breaking an alliance with another
  are shown and can not be picked: the original never enables the first, and the second does
  nothing there.
- Threaten/Break Treaty: break the pact or alliance, or threaten. A threat always costs relation
  and patience; the wizard then declares war, ignores it, pays gold, or gives a spell. Visiting this
  list at all, even to forget it, wears the patience down ten times over (as in the original).
- Offer Tribute: gold (a quarter to all, in 25s) or a spell the wizard lacks.
- Exchange Spells: a spell of the wizard for one of the human's worth at least as much (see below).
- A wizard whose patience is gone says it is tired of talk and ends it.

Before the human's stack attacks a stack or city of a wizard it has a pact or alliance with, the
original's question comes: "You have a treaty with X.  Do you still wish to attack?" Yes breaks the
treaty (and the wizard minds it by 40); no keeps the stack where it was. Computer wizards never
attack their partners.

## Contact

Two wizards have relations only after they have met. In the original the human meets a wizard when
the human's scouting covers one of its cities or a unit that is not invisible, and any wizard with
Nature's Awareness meets every wizard with a visible unit; computer wizards never meet each other by
sight. This is the rule here too (`game/contactclassic.go`; `-classic-contact=false` for the fork's
rule, every wizard meeting every wizard it sees). The worlds are the original's 60 by 40.

## Kept from the original, on purpose

- `quirkAllianceBecomesPact`: when both wizards of a pair pass the test for an alliance in the same
  turn, the second makes it a pact again.
- `quirkPeaceNoGain`: peace raises the relation by nothing (the original adds 20 and overwrites it).
- The greetings by personality are the records 15 to 20 in the personality's order; the Peaceful
  one is the most hostile of the texts.
- One time in 4 a wizard the human gave a spell answers with its thanks for that spell instead of
  what it would say.
- The threat of an army counts the more the two armies on the wizard's home land differ, whichever
  is larger.

## As the original (put back 2026-09-30)

- Break_Treaties: size 10 (pact) or 20 (alliance), twice for a lawful breaker; the breaker's hidden
  relation to its victim goes UP by it, every other wizard's to the VICTIM down by 5, after an
  alliance the breaker's lasting relation up by it, and the lasting relation is copied to the
  victim's view (`quirkBreakTreatiesBackwards`; the reconstruction marks these as the program's
  bugs).
- A warning and a broken treaty say the words of the grievance alone
  (`quirkWarningWordsOfGrievance`; the messages have words of their own for both, records 15 and 24
  past the grievance, which no code of the original is known to say).
- A threat that pays gold gives it to the human and takes none from the wizard
  (`quirkTributeFromNowhere`).
- The exchange of spells: a spell's worth to a receiver is nothing when it knows it, -1 when it can
  not learn it, its research cost less a tenth when it can research it now, half again when it can
  later; the wizard's spell's worth is kept in a byte (what is left over 256, -1 is 255), and the
  human's spells worth at least that to the wizard are offered (`quirkExchangeWorthByte`).

## Our calls

- The lists of spells of a tribute and an exchange end with Forget It.
- The band of the item under the mouse is dark by eye (the original's remap is not read).
- A proposal with gold or a spell adds "What if we were to also offer ... as an incentive?".
- The human's answers to a proposal take effect when the screen is shown (the start of the human's
  turn), not in the middle of the end of the turn.

## Not done

- A computer wizard's proposal of an exchange of spells, its request for a war on another wizard,
  rewards for attacking its enemy, warnings about summoned creatures.
- The mirror of another wizard does not show relation, treaty, personality and objective yet.

## Dev

`-capture-screen diplomacy` (the human's talk), `diplomacymenu`, `diplomacypropose`,
`diplomacywar` (a message at the end of a turn), `diplomacygreeting`. The headless summary
(`-sim`) lists personalities, treaties and relations, and what was said to the human.
