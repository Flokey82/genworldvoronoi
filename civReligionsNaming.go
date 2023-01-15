package genworldvoronoi

import (
	"math/rand"
	"strings"
)

var approaches []string

func init() {
	approaches = weightedToArray(genMeaningApproaches)
}

func (m *Civ) getFolkReligionName(c *Culture, form string) string {
	return c.Name + " " + rw(types[form])
}

// getReligionName generates a name for the given form and deity at the given center.
// This code is based on:
// https://github.com/Azgaar/Fantasy-Map-Generator/blob/master/modules/religions-generator.js
func (m *Civ) getReligionName(form, deity string, r int) (string, string) {
	// Returns a random name from the culture at the given region.
	random := func() string {
		c := m.GetCulture(r)
		return c.Language.MakeName()
	}

	// Returns a random, weighted type for the religion form.
	rType := func() string {
		return rw(types[form])
	}

	// Splits the deity name into parts and returns the first part.
	// Example: "Grognark, The Supreme Being" -> "Grognark"
	supreme := func() string {
		return strings.Split(deity, (", "))[0]
	}

	// Returns the name of the culture at the given region.
	culture := func() string {
		c := m.GetCulture(r)
		return c.Name
	}

	// Returns the name of the cit or state at the given region.
	place := func(adj string) string {
		// Check if we have a city at the region.
		var base string
		for _, city := range m.Cities {
			if city.ID == r {
				base = city.Name
				break
			}
		}
		if base == "" {
			// Check if we have a state at the region.
			if stateId := m.RegionToCityState[r]; stateId != 0 {
				for _, city := range m.Cities {
					if city.ID == stateId {
						base = city.Name
						break
					}
				}
			}
		}
		if base == "" {
			// Check if we have a culture at the region.
			if cultureId := m.RegionToCulture[r]; cultureId != 0 {
				base = m.Cultures[cultureId].Name
			}
		}
		if base == "" {
			return "TODO_PLACE"
		}
		name := trimVowels(strings.Split(base, " ")[0], 3)
		if adj != "" {
			return getAdjective(name)
		}
		return name
	}

	switch rw(genReligionMethods) {
	case MethodRandomType:
		return random() + " " + rType(), ReligionExpGlobal
	case MethodRandomIsm:
		return trimVowels(random(), 3) + "ism", ReligionExpGlobal
	case MethodSurpremeIsm:
		if deity != "" {
			return trimVowels(supreme(), 3) + "ism", ReligionExpGlobal
		}
	case MethodFaithOfSupreme:
		if deity != "" {
			// Select a random name from the list.
			// but ensure that the name is not a subset of the deity name
			// and vice versa. This is to avoid names like "The Way of The Way".
			var prefix string
			for i := 0; i < 100; i++ {
				prefix = ra([]string{
					"Faith",
					"Way",
					"Path",
					"Word",
					"Truth",
					"Law",
					"Order",
					"Light",
					"Darkness",
					"Gift",
					"Grace",
					"Witnesses",
					"Servants",
					"Messengers",
					"Believers",
					"Disciples",
					"Followers",
					"Children",
					"Brothers",
					"Sisters",
					"Brothers and Sisters",
					"Sons",
					"Daughters",
					"Sons and Daughters",
					"Brides",
					"Grooms",
					"Brides and Grooms",
				})
				if !strings.Contains(strings.ToLower(deity), strings.ToLower(prefix)) &&
					!strings.Contains(strings.ToLower(prefix), strings.ToLower(deity)) {
					break
				}
			}
			return prefix + " of " + supreme(), ReligionExpGlobal
		}
	case MethodPlaceIsm:
		return place("") + "ism", ReligionExpState
	case MethodCultureIsm:
		return trimVowels(culture(), 3) + "ism", ReligionExpCulture
	case MethodPlaceIanType:
		return place("adj") + " " + rType(), ReligionExpState
	case MethodCultureType:
		return culture() + " " + rType(), ReligionExpCulture
	}
	return trimVowels(random(), 3) + "ism", ReligionExpGlobal
}

// getDeityName returns a deity name for the given culture.
func getDeityName(culture *Culture, approach string) (string, string) {
	if culture == nil {
		return "TODO_DEITY", "TODO_DEITY"
	}
	meaning := generateDeityMeaning(approach)
	cultureName := culture.Language.MakeName()
	return cultureName, meaning
}

// generateDeityMeaning generates a meaning for a deity name.
func generateDeityMeaning(approach string) string {
	switch approach { // select generation approach
	case ApproachNumber:
		return ra(genBase[GenBaseNumber])
	case ApproachBeing:
		return ra(genBase[GenBaseBeing])
	case ApproachAdjective:
		return ra(genBase[GenBaseAdjective])
	case ApproachColorAnimal:
		return ra(genBase[GenBaseColor]) + " " + ra(genBase[GenBaseAnimal])
	case ApproachAdjectiveAnimal:
		return ra(genBase[GenBaseAdjective]) + " " + ra(genBase[GenBaseAnimal])
	case ApproachAdjectiveBeing:
		return ra(genBase[GenBaseAdjective]) + " " + ra(genBase[GenBaseBeing])
	case ApproachAdjectiveGenitive:
		return ra(genBase[GenBaseAdjective]) + " " + ra(genBase[GenBaseGenitive])
	case ApproachColorBeing:
		return ra(genBase[GenBaseColor]) + " " + ra(genBase[GenBaseBeing])
	case ApproachColorGenitive:
		return ra(genBase[GenBaseColor]) + " " + ra(genBase[GenBaseGenitive])
	case ApproachBeingOfGenitive:
		return ra(genBase[GenBaseBeing]) + " of " + ra(genBase[GenBaseGenitive])
	case ApproachBeingOfTheGenitive:
		return ra(genBase[GenBaseBeing]) + " of the " + ra(genBase[GenBaseTheGenitive])
	case ApproachAnimalOfGenitive:
		return ra(genBase[GenBaseAnimal]) + " of " + ra(genBase[GenBaseGenitive])
	case ApproachAdjectiveBeingOfGenitive:
		return ra(genBase[GenBaseAdjective]) + " " + ra(genBase[GenBaseBeing]) + " of " + ra(genBase[GenBaseGenitive])
	case ApproachAdjectiveAnimalOfGenitive:
		return ra(genBase[GenBaseAdjective]) + " " + ra(genBase[GenBaseAnimal]) + " of " + ra(genBase[GenBaseGenitive])
	default:
		return "ERROR"
	}
}

// genMeaningApproaches contains a map of name generation
// approaches and their relative chance to be selected.
var genMeaningApproaches = map[string]int{
	ApproachNumber:                    1,
	ApproachBeing:                     3,
	ApproachAdjective:                 5,
	ApproachColorAnimal:               5,
	ApproachAdjectiveAnimal:           5,
	ApproachAdjectiveBeing:            5,
	ApproachAdjectiveGenitive:         1,
	ApproachColorBeing:                3,
	ApproachColorGenitive:             3,
	ApproachBeingOfGenitive:           2,
	ApproachBeingOfTheGenitive:        1,
	ApproachAnimalOfGenitive:          1,
	ApproachAdjectiveBeingOfGenitive:  2,
	ApproachAdjectiveAnimalOfGenitive: 2,
}

const (
	ApproachNumber                    = "Number"
	ApproachBeing                     = "Being"
	ApproachAdjective                 = "Adjective"
	ApproachColorAnimal               = "Color + Animal"
	ApproachAdjectiveAnimal           = "Adjective + Animal"
	ApproachAdjectiveBeing            = "Adjective + Being"
	ApproachAdjectiveGenitive         = "Adjective + Genitive"
	ApproachColorBeing                = "Color + Being"
	ApproachColorGenitive             = "Color + Genitive"
	ApproachBeingOfGenitive           = "Being + of + Genitive"
	ApproachBeingOfTheGenitive        = "Being + of the + Genitive"
	ApproachAnimalOfGenitive          = "Animal + of + Genitive"
	ApproachAdjectiveBeingOfGenitive  = "Adjective + Being + of + Genitive"
	ApproachAdjectiveAnimalOfGenitive = "Adjective + Animal + of + Genitive"
)

const (
	MethodRandomType     = "Random + type"
	MethodRandomIsm      = "Random + ism"
	MethodSurpremeIsm    = "Supreme + ism"
	MethodFaithOfSupreme = "Faith of + Supreme"
	MethodPlaceIsm       = "Place + ism"
	MethodCultureIsm     = "Culture + ism"
	MethodPlaceIanType   = "Place + ian + type"
	MethodCultureType    = "Culture + type"
)

// genReligionMethods contains a map of religion name generation
// methods and their relative chance to be selected.
var genReligionMethods = map[string]int{
	MethodRandomType:     3,
	MethodRandomIsm:      1,
	MethodSurpremeIsm:    5,
	MethodFaithOfSupreme: 5,
	MethodPlaceIsm:       1,
	MethodCultureIsm:     2,
	MethodPlaceIanType:   6,
	MethodCultureType:    4,
}

func ra(array []string) string {
	return array[rand.Intn(len(array))]
}

func rw(mp map[string]int) string {
	return ra(weightedToArray(mp))
}

const (
	GenBaseNumber      = "number"
	GenBaseBeing       = "being"
	GenBaseAnimal      = "animal"
	GenBaseAdjective   = "adjective"
	GenBaseGenitive    = "genitive"
	GenBaseTheGenitive = "theGenitive"
	GenBaseColor       = "color"
)

// genBase contains a map of word lists used for name generation.
// TODO: Group individual entries by logical categories.
// So we can build up a pantheon of gods, each associated with different domains.
// For example:
// {North, South, East, West} -> {Direction}
// {Bride, Groom, Widow, Widower, Wife, Husband} -> {Marriage}
// {Giver, Taker, Destroyer, Creator, Maker, Breaker} -> {Action}
// {Sky, Earth, Water, Fire, Air, Spirit} -> {Elements}
// {Light, Dark, Bright, Shining, Shadow, Darkness} -> {Light}
var genBase = map[string][]string{
	GenBaseNumber: {"One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Eleven", "Twelve"},
	GenBaseBeing: {
		"Ancestor",
		"Ancient",
		"Angel",
		"Betrayer",
		"Bride",
		"Brother",
		"Chief",
		"Child",
		"Council",
		"Creator",
		"Deity",
		"Elder",
		"Father",
		"Forebearer",
		"Forefather",
		"Foremother",
		"Forgiven",
		"Forgotten",
		"Giver",
		"God",
		"Goddess",
		"Golem",
		"Groom",
		"Guardian",
		"Guide",
		"Keeper",
		"King",
		"Lady",
		"Lord",
		"Lover",
		"Maker",
		"Master",
		"Mistress",
		"Mother",
		"Numen",
		"Orphan",
		"Overlord",
		"Reaper",
		"Ruler",
		"Seducer",
		"Seductress",
		"Servant",
		"Sister",
		"Spirit",
		"Virgin",
		"Warrior",
		"Watcher",
		"Widow",
		"Widower",
		"Wife",
		"Witch",
		"Wizard",
	},
	GenBaseAnimal: {
		"Antelope",
		"Ape",
		"Badger",
		"Basilisk",
		"Bear",
		"Beaver",
		"Bison",
		"Boar",
		"Buffalo",
		"Camel",
		"Cat",
		"Centaur",
		"Chimera",
		"Cobra",
		"Crane",
		"Crocodile",
		"Crow",
		"Cyclope",
		"Deer",
		"Dog",
		"Dragon",
		"Eagle",
		"Elk",
		"Falcon",
		"Fox",
		"Goat",
		"Goose",
		"Hare",
		"Hawk",
		"Heron",
		"Horse",
		"Hound",
		"Hyena",
		"Ibis",
		"Jackal",
		"Jaguar",
		"Kraken",
		"Lark",
		"Leopard",
		"Lion",
		"Mantis",
		"Marten",
		"Moose",
		"Mule",
		"Narwhal",
		"Owl",
		"Ox",
		"Panther",
		"Pegasus",
		"Phoenix",
		"Rat",
		"Raven",
		"Rook",
		"Scorpion",
		"Serpent",
		"Shark",
		"Sheep",
		"Snake",
		"Sphinx",
		"Spider",
		"Swan",
		"Tiger",
		"Turtle",
		"Unicorn",
		"Viper",
		"Vulture",
		"Walrus",
		"Wolf",
		"Wolverine",
		"Worm",
		"Wyvern",
	},
	GenBaseAdjective: {
		"Aggressive",
		"Almighty",
		"Ancient",
		"Angry",
		"Anxious",
		"Awful",
		"Beautiful",
		"Benevolent",
		"Big",
		"Blind",
		"Blond",
		"Bloody",
		"Brave",
		"Broken",
		"Brutal",
		"Burning",
		"Calm",
		"Careful",
		"Charming",
		"Cheerful",
		"Chosen",
		"Clean",
		"Crazy",
		"Cruel",
		"Dead",
		"Deadly",
		"Deaf",
		"Deathless",
		"Deep",
		"Defiant",
		"Delicate",
		"Delightful",
		"Desperate",
		"Devastating",
		"Distant",
		"Disturbing",
		"Divine",
		"Dying",
		"Enchanting",
		"Ephemeral",
		"Eternal",
		"Evil",
		"Explicit",
		"Fair",
		"Far",
		"Fat",
		"Fatal",
		"Favorable",
		"Flying",
		"Friendly",
		"Frozen",
		"Giant",
		"Good",
		"Grateful",
		"Great",
		"Greedy",
		"Happy",
		"High",
		"Holy",
		"Honest",
		"Huge",
		"Hungry",
		"Immutable",
		"Infallible",
		"Inherent",
		"Last",
		"Latter",
		"Lonely",
		"Lost",
		"Loud",
		"Loving",
		"Lucky",
		"Lustful",
		"Mad",
		"Magical",
		"Main",
		"Major",
		"Marine",
		"Naval",
		"New",
		"Old",
		"Patient",
		"Peaceful",
		"Pregnant",
		"Prime",
		"Proud",
		"Pure",
		"Sacred",
		"Sad",
		"Scary",
		"Secret",
		"Selected",
		"Severe",
		"Silent",
		"Sleeping",
		"Slumbering",
		"Spiteful",
		"Strong",
		"Sunny",
		"Superior",
		"Sustainable",
		"Troubled",
		"Undying",
		"Unhappy",
		"Unknown",
		"Waking",
		"Wild",
		"Wise",
		"Worried",
		"Young",
	},
	GenBaseGenitive: {
		"Cold",
		"Darkness",
		"Dawn",
		"Day",
		"Death",
		"Doom",
		"Dreams",
		"Dusk",
		"Fate",
		"Fire",
		"Fog",
		"Frost",
		"Gates",
		"Heaven",
		"Home",
		"Hope",
		"Ice",
		"Justice",
		"Kings",
		"Life",
		"Light",
		"Lightning",
		"Love",
		"Nature",
		"Night",
		"Pain",
		"Snow",
		"Springs",
		"Stars",
		"Summer",
		"Sun",
		"Sunset",
		"Thunder",
		"Time",
		"Victory",
		"War",
		"Wealth",
		"Winter",
		"Wisdom",
	},
	GenBaseTheGenitive: {
		"Abyss",
		"Blood",
		"Dawn",
		"Earth",
		"East",
		"Eclipse",
		"Fall",
		"Harvest",
		"Moon",
		"North",
		"Peak",
		"Rainbow",
		"Sea",
		"Sky",
		"South",
		"Stars",
		"Storm",
		"Sun",
		"Tree",
		"Underworld",
		"West",
		"Wild",
		"Word",
		"World",
	},
	GenBaseColor: {
		"Amber",
		"Black",
		"Blue",
		"Bright",
		"Brown",
		"Dark",
		"Golden",
		"Green",
		"Grey",
		"Light",
		"Orange",
		"Pink",
		"Purple",
		"Red",
		"White",
		"Yellow",
	},
}
