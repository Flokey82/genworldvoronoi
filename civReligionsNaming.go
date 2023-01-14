package genworldvoronoi

import (
	"math/rand"
	"regexp"
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
func (m *Civ) getReligionName(form, deity string, regCenter int) (string, string) {
	random := func() string {
		c := m.GetCulture(regCenter)
		return c.Language.MakeName()
	}
	rType := func() string {
		return rw(types[form])
	}
	deitySplit := regexp.MustCompile(`/[ ,]+/`)
	supreme := func() string {
		return deitySplit.Split(deity, -1)[0]
	}
	culture := func() string {
		c := m.GetCulture(regCenter)
		return c.Name
	}
	place := func(adj string) string {
		// Check if we have a city at the center.
		var base string
		for _, city := range m.Cities {
			if city.ID == regCenter {
				base = city.Name
				break
			}
		}
		if base == "" {
			// Check if we have a state at the center.
			if stateId := m.RegionToCityState[regCenter]; stateId != 0 {
				for _, city := range m.Cities {
					if city.ID == stateId {
						base = city.Name
						break
					}
				}
			}
		}
		if base == "" {
			// Check if we have a culture at the center.
			if cultureId := m.RegionToCulture[regCenter]; cultureId != 0 {
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
			return ra([]string{"Faith", "Way", "Path", "Word", "Witnesses"}) + " of " + supreme(), ReligionExpGlobal
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
func getDeityName(culture *Culture) string {
	if culture == nil {
		return "ERROR"
	}
	meaning := generateDeityMeaning()
	cultureName := culture.Language.MakeName() // Names.getCulture(culture, nil, nil, "", 0.8)
	return cultureName + ", The " + meaning
}

// generateDeityMeaning generates a meaning for a deity name.
func generateDeityMeaning() string {
	switch ra(approaches) { // select generation approach
	case ApproachNumber:
		return ra(genBase["number"])
	case ApproachBeing:
		return ra(genBase["being"])
	case ApproachAdjective:
		return ra(genBase["adjective"])
	case ApproachColorAnimal:
		return ra(genBase["color"]) + " " + ra(genBase["animal"])
	case ApproachAdjectiveAnimal:
		return ra(genBase["adjective"]) + " " + ra(genBase["animal"])
	case ApproachAdjectiveBeing:
		return ra(genBase["adjective"]) + " " + ra(genBase["being"])
	case ApproachAdjectiveGenitive:
		return ra(genBase["adjective"]) + " " + ra(genBase["being"])
	case ApproachColorBeing:
		return ra(genBase["color"]) + " " + ra(genBase["being"])
	case ApproachColorGenitive:
		return ra(genBase["color"]) + " " + ra(genBase["genitive"])
	case ApproachBeingOfGenitive:
		return ra(genBase["being"]) + " of " + ra(genBase["genitive"])
	case ApproachBeingOfTheGenitive:
		return ra(genBase["being"]) + " of the " + ra(genBase["genitive"])
	case ApproachAnimalOfGenitive:
		return ra(genBase["animal"]) + " of " + ra(genBase["genitive"])
	case ApproachAdjectiveBeingOfGenitive:
		return ra(genBase["adjective"]) + " " + ra(genBase["being"]) + " of " + ra(genBase["genitive"])
	case ApproachAdjectiveAnimalOfGenitive:
		return ra(genBase["adjective"]) + " " + ra(genBase["animal"]) + " of " + ra(genBase["genitive"])
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

var genBase = map[string][]string{
	"number": {"One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Eleven", "Twelve"},
	"being": {
		"Ancestor",
		"Ancient",
		"Brother",
		"Chief",
		"Council",
		"Creator",
		"Deity",
		"Elder",
		"Father",
		"Forebear",
		"Forefather",
		"Giver",
		"God",
		"Goddess",
		"Guardian",
		"Lady",
		"Lord",
		"Maker",
		"Master",
		"Mother",
		"Numen",
		"Overlord",
		"Reaper",
		"Ruler",
		"Sister",
		"Spirit",
		"Virgin",
	},
	"animal": {
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
	"adjective": {
		"Aggressive",
		"Almighty",
		"Ancient",
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
		"Cheerful",
		"Crazy",
		"Cruel",
		"Dead",
		"Deadly",
		"Devastating",
		"Distant",
		"Disturbing",
		"Divine",
		"Dying",
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
		"Lost",
		"Loud",
		"Lucky",
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
		"Strong",
		"Sunny",
		"Superior",
		"Sustainable",
		"Troubled",
		"Unhappy",
		"Unknown",
		"Waking",
		"Wild",
		"Wise",
		"Worried",
		"Young",
	},
	"genitive": {
		"Cold",
		"Day",
		"Death",
		"Doom",
		"Fate",
		"Fire",
		"Fog",
		"Frost",
		"Gates",
		"Heaven",
		"Home",
		"Ice",
		"Justice",
		"Life",
		"Light",
		"Lightning",
		"Love",
		"Nature",
		"Night",
		"Pain",
		"Snow",
		"Springs",
		"Summer",
		"Thunder",
		"Time",
		"Victory",
		"War",
		"Winter",
	},
	"theGenitive": {
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
	"color": {
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
