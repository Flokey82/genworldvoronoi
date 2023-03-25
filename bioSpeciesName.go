package genworldvoronoi

import (
	"math/rand"

	"github.com/Flokey82/go_gens/genlanguage"
)

func (b *Bio) genSpeciesNameByFamily(fam SpeciesFamily) string {
	baseName := fam.String()
	// Get a random name from the list.
	names := speciesFamilyToCommonNames[fam]
	if names != nil {
		baseName = names[rand.Intn(len(names))]
	}

	switch rand.Intn(3) {
	case 0:
		adjs := genlanguage.GenBase[genlanguage.GenBaseAdjective]
		return adjs[rand.Intn(len(adjs))] + " " + baseName
	case 1:
		cols := genlanguage.GenBase[genlanguage.GenBaseColor]
		return cols[rand.Intn(len(cols))] + " " + baseName
	case 2:
		noun := genlanguage.GenBase[genlanguage.GenBaseGenitive]
		return noun[rand.Intn(len(noun))] + " " + baseName
	}

	return baseName
}

// TODO: Move species name generation to a separate package.
var speciesFamilyToCommonNames = map[SpeciesFamily][]string{
	SpeciesFamilyTree: {
		"tree",
		"trunk",
		"branch",
		"bark",
		"wood",
		"oak",
		"pine",
		"maple",
		"birch",
		"willow",
		"elm",
		"ash",
		"cedar",
	},
	SpeciesFamilyShrub: {
		"shrub",
		"bush",
		"thorn",
		"thicket",
		"bramble",
		"rose",
	},
	SpeciesFamilyGrass: {
		"grass",
		"reeds",
		"reed",
		"blade",
	},
	SpeciesFamilyReed: {
		"reed",
		"grass",
		"pipe",
	},
	SpeciesFamilyHerb: {
		"herb",
		"weed",
		"green",
		"leaf",
		"grass",
		"flower",
	},
	SpeciesFamilyFlower: {
		"flower",
		"petal",
		"blossom",
		"bloom",
		"bud",
		"leaf",
		"grass",
	},
	SpeciesFamilyFern: {
		"fern",
		"frond",
		"leaf",
		"grass",
	},
	SpeciesFamilyMoss: {
		"moss",
		"lichen",
		"green",
		"carpet",
	},
	SpeciesFamilyVine: {
		"vine",
		"grape",
		"creeper",
		"climber",
		"curtain",
	},
	SpeciesFamilyCactus: {
		"cactus",
		"spine",
		"prickly",
		"thorn",
		"needle",
	},
	SpeciesFamilySucculent: {
		"succulent",
		"thorn",
		"prickly",
		"spine",
		"needle",
	},
	SpeciesFamilyInsect: {
		"insect",
		"bug",
		"fly",
		"ant",
		"bee",
		"butterfly",
		"moth",
		"dragonfly",
		"grasshopper",
		"beetle",
		"ladybug",
		"mosquito",
		"fly",
	},
	SpeciesFamilyArachnid: {
		"arachnid",
		"spider",
		"tarantula",
		"scorpion",
		"tick",
	},
	SpeciesFamilyMammal: {
		"animal",
		"mammal",
		"beast",
		"creature",
		"dog",
		"cat",
		"horse",
	},
	SpeciesFamilyBird: {
		"bird",
		"chicken",
		"duck",
		"swan",
		"eagle",
		"owl",
		"raven",
		"crow",
		"robin",
		"parrot",
		"pigeon",
		"peacock",
		"penguin",
		"flamingo",
		"swallow",
		"vulture",
		"seagull",
	},
	SpeciesFamilyFish: {
		"fish",
		"shark",
		"whale",
		"dolphin",
		"seal",
		"salmon",
		"trout",
		"eel",
		"sturgeon",
		"bass",
		"pike",
	},
	SpeciesFamilyCrustacean: {
		"crustacean",
		"crab",
		"lobster",
		"shrimp",
		"prawn",
		"crayfish",
		"krill",
		"crayfish",
	},
	SpeciesFamilyMollusk: {
		"mollusk",
		"snail",
		"slug",
		"octopus",
		"clam",
		"oyster",
		"mussel",
		"conch",
		"whelk",
		"shell",
	},
	SpeciesFamilyAmphibian: {
		"amphibian",
		"frog",
		"toad",
		"newt",
		"salamander",
	},
	SpeciesFamilyReptileSerpent: {
		"reptile",
		"snake",
		"serpent",
		"cobra",
		"anaconda",
		"viper",
		"boa",
		"python",
		"lizard",
		"gecko",
		"iguana",
		"chameleon",
	},
	SpeciesFamilyReptileLizard: {
		"reptile",
		"lizard",
		"gecko",
		"iguana",
		"chameleon",
	},
	SpeciesFamilyRodent: {
		"rodent",
		"rat",
		"mouse",
		"hamster",
		"gerbil",
		"rabbit",
		"hare",
		"beaver",
		"mole",
		"shrew",
		"squirrel",
		"chipmunk",
		"porcupine",
	},
	SpeciesFamilyMolluskSnail: {
		"snail",
		"slug",
		"mollusk",
	},
	SpeciesFamilyWorm: {
		"worm",
		"creeper",
		"string",
	},
	SpeciesFamilyMushroom: {
		"fungus",
		"mushroom",
		"toadstool",
		"mold",
		"mildew",
		"moss",
		"lichen",
		"fungi",
		"cap",
		"hat",
	},
	SpeciesFamilyMold: {
		"mold",
		"mildew",
		"fungus",
		"fluff",
	},
}
