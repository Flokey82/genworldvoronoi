# Civilization module

The civ package contains everything related to civilizations.

## Tribes

A tribe is a group of people that are not bound to a specific location. They can move around the map (as long as regions are unpopulated), develop preferences for region properties, new skills, and may settle in different places. Tribes have the ability to attack settlements or other tribes if they are blocking their way or if they are in a region that the tribe wants to settle in.

### TODO

- [ ] merge with other tribes
- [ ] Resources
    - [ ] better storage
    - [ ] smarter generation

## Settlements

Settlements (or cities) are just that. Settlements.

### TODO

- [ ] Unify Tick() for cities and tribes that run cities
- [ ] Construction
    - [ ] construction queue / management
    - [ ] buildings (simulating actual homes for people to live in)
    - [ ] features / infrastructure (walls, mills, docks, etc.)
    - [ ] resource management and worker allocation
- [ ] Resources
    - [ ] smarter generation (farms produce food, mines produce ore, etc.)
    - [ ] consumption
    - [ ] trading
    - [ ] storage
        - [ ] storage buildings
        - [ ] better storage in general
- [ ] Population
    - [ ] City specific actions for people
- [ ] Leadership and factions
    - [ ] city specific actions
- [ ] Scores
    - [ ] re-think what and how we score
    - [ ] account for potential modifiers

## City States

City states are settlements that have developed beyond a mere settlement. City states occupy more than one region and can have control over multiple settlements or cities.

### TODO

- [ ] Unify Tick() for city states and settlements that run city states
- [ ] Expansion
    - [ ] conquering
    - [ ] colonization
    - [ ] vassalization
    - [X] growth (simply expanding into unoccupied regions)
- [ ] Leadership and factions
    - [ ] city state specific actions

## Empires

Empires are created when a city state reaches a certain size and power. Empires occupy the regions of one or more city states that are under their control.

### TODO

- [ ] Unify Tick() for empires and city states that run empires
- [ ] Expansion
    - [ ] conquering
    - [ ] colonization
    - [ ] vassalization
- [ ] Leadership and factions
    - [ ] empire specific actions
    - [X] replace emperor with leadership (actual person)

## Religions

Religions are a belief system that can be adopted by tribes, settlements, city states, and empires. Religions can have a significant impact on the development of a civilization.

There are two main types of religions: Folk religions and Organized religions.

### Folk religions

Folk religions are the belief systems of tribes. They are usually based on the worship of natural elements, spirits, or ancestors. Folk religions are not organized and are usually passed down through oral tradition. Folk religions can be adopted by settlements, city states, and empires.

### Organized religions

Organized religions are the belief systems of settlements, city states, and empires. Organized religions are usually based on the worship of a deity or deities and have a formal structure. Organized religions can be adopted by settlements, city states, and empires.

## Cultures

Cultures are the customs, traditions, and social institutions of a group of people. Cultures are influenced by the way of life of a tribe, settlement, city state, or empire.

### TODO

- [ ] Properties
    - [-] expansionism
    - [-] martialism
    - [-] spirituality
    - [-] openness
    - [ ] sophistication
    - [ ] extremism
- [ ] Evolution
    - [ ] forking and merging
    - [ ] diffusion
    - [ ] evolution
- [ ] Traits
    - [ ] traditions
    - [ ] skills and specializations

## Languages

Languages are randomly generated initially and can create "forks" or "branches" of languages as they evolve (when tribes split off or a group of people decides to leave a settlement).

## Leadership and factions

Leadership and factions are the political structures that govern people. Both are technically the same, but only the leadership may decide on the direction of the entity they govern. 

Depending on fate or decisions made by the leadership, factions may form or even cause an uprising. A popular non-governing faction may even take over the leadership (or the leadership may be overthrown).

### Forms of government

Each faction or leadership decides how it wants to run things. Democracy, monarchy, oligarchy, and dictatorship are just a few examples of forms of government. Ultimately, the form of government should determine the direction of the entity and limit which actions can be taken.

For example, a democracy may not simply declare war on another entity without the consent of the people. A dictatorship, on the other hand, may declare war on a whim, but the people may not be happy about it which could lead to an uprising.

### Leaders

Currently leaders are chosen for life and either picked based on inheritance, election, or by force. Leaders can be overthrown, assassinated, or die of old age. Leaders can also be exiled or retire.

### TODO

- [ ] Actions
    - [ ] multi-leader actions
    - [ ] multi-step actions
    - [ ] long term goals
- [ ] Leadership
    - [ ] multiple leaders
    - [ ] succession
        - [ ] elections
        - [-] inheritance
        - [-] assassination
        - [ ] exile
        - [ ] retirement
        - [-] overthrown
- [ ] Diplomacy
- [ ] War and peace
- [ ] Trade
- [ ] Schemes

## Storage

There is a unified storage system, which is used by all entities, but it sucks. It really needs to be improved. For example, it currently has two storage implementations, one for resources that stores each sub-type of resource separately (e.g. birch, oak, and pine wood are all stored separately) and one for categories of resources (e.g. all wood is stored as one number). This is a hack to make it easier to handle construction stuff where we don't yet care what type of wood or stone we are using.

### TODO

- [ ] Drop the hack
- [ ] Move away from the fixed resource IDs

## Artifacts

Artifacts are special items that can be found in the world or are created by individuals. They can have conditions attached to them (like blessings or curses). When artifacts are found, any condition might be transferred to the person who found it. If an artifact is stolen or lost, the condition might be transferred to the thief or the person who found it.

### TODO

- [ ] Locations for artifacts (when dropped, lost, or initallly placed)
- [ ] History or flavor text for artifacts
- [ ] Move away from only having a string that describes the artifact
- [ ] Could be "visible" for others when not hidden?
- [ ] Could be worn or carried by a person?

## Armies (TODO)

Right now, the only entity that can fight is a tribe. There are a lot of features that would overlap with tribes, so it might be a good idea to factor out a lot of the potentially common code.

### TODO

- [ ] Factor out tribes combat code
- [ ] Add commanders
- [ ] Add heroes
- [ ] Add unit types
- [ ] Add artifacts

## People

People are the individuals that make up a tribe, settlement, city state, or empire. People have skills, traits, and preferences. People take actions based on their skills, traits, and preferences.
People can be born, die, marry, have children.

### TODO

- [ ] Actions
    - [ ] multi-step actions
    - [ ] long term goals
- [ ] Movement
    - [ ] people can move between regions and entities