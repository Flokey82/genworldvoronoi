package geo

import (
	"math/rand"

	"github.com/Flokey82/genbiome"
)

func (m *Geo) GenerateRegPropertyDescription(p RegProperty) string {
	str := "The region"
	if p.OnIsland {
		// TODO: Note if the region is located on an island or peninsula.
		str += ", located on an island,"
	}
	str += " is covered by " + genbiome.WhittakerModBiomeToString(p.Biome) + ".\n"

	// Add info on the potential dangers of the region.
	if p.DistanceToVolcano < 3 {
		if p.Danger.Volcano > 0.2 {
			if p.DistanceToVolcano < 1 {
				str += " It's location on a volcano"
			} else {
				str += " The proximity to a volcano"
			}
			if p.Danger.Volcano > 0.5 {
				str += " results in a constant danger of destruction by a volcanic eruption"
			} else {
				str += " poses a looming threat of a possible volcanic eruption"
			}
			str += ". "
		} else {
			if p.DistanceToVolcano < 1 {
				str += " It is located on a volcano"
			} else {
				str += " It is close to a volcano"
			}
			str += ". "
		}
	} else if p.DistanceToMountain < 3 {
		if p.Danger.RockSlide > 0.2 {
			if p.DistanceToMountain < 1 {
				str += " The exposed location on a mountain"
			} else {
				str += " The proximity to a mountain"
			}
			if p.Danger.RockSlide > 0.5 {
				str += " poses a constant danger of a deadly rockslide"
			} else {
				str += " results in occasional rockslides that threaten anyone nearby"
			}
			str += ". "
		}
	} else if p.DistanceToFaultline < 3 {
		if p.Danger.Earthquake > 0.2 {
			if p.DistanceToFaultline < 1 {
				str += " The exposed location on a faultline"
			} else {
				str += " The proximity to a faultline"
			}
			if p.Danger.Earthquake > 0.5 {
				str += " results in a constant danger of a deadly earthquake"
			} else {
				str += " poses a looming threat of a possible earthquake"
			}
			str += ". "
		}
	}
	if p.DistanceToRiver < 1 {
		str += " The nearby river provides access to fresh water"
		if p.Danger.Flood > 0.2 {
			if p.Danger.Flood > 0.5 {
				str += " and is infamous for frequent floods"
			} else {
				str += " and might cause occasional flooding"
			}
		}
		if p.HasWaterfall {
			str += " and ends in a treacherous waterfall"
		}
		str += ". \n"
	}
	return str
}

type BiomeDescription struct {
	Adjectives []string
	Nouns      []string
	Part1      []string
	Part2      []string
	Part3      []string
	Part4      []string
}

func GenerateFlavorText(desc BiomeDescription) string {
	text := "The " + desc.Adjectives[rand.Intn(len(desc.Adjectives))] + " " + desc.Nouns[rand.Intn(len(desc.Nouns))] + " stretches out as far as the eye can see.\n"
	if len(desc.Part1) > 0 {
		text += desc.Part1[rand.Intn(len(desc.Part1))] + " \n"
	}
	if len(desc.Part2) > 0 {
		text += desc.Part2[rand.Intn(len(desc.Part2))] + " \n"
	}
	if len(desc.Part3) > 0 {
		text += desc.Part3[rand.Intn(len(desc.Part3))] + " \n"
	}
	if len(desc.Part4) > 0 {
		text += desc.Part4[rand.Intn(len(desc.Part4))] + " \n"
	}
	return text
}

func GenerateFlavorTextForBiome(seed int64, biome int) string {
	rand.Seed(seed)
	switch biome {
	case genbiome.WhittakerModBiomeSubtropicalDesert:
		return GenerateFlavorText(desertDescription)
	case genbiome.WhittakerModBiomeColdDesert:
		return GenerateFlavorText(coldDesertDescription)
	case genbiome.WhittakerModBiomeTropicalRainforest:
		return GenerateFlavorText(tropicalRainforestDescription)
	case genbiome.WhittakerModBiomeTropicalSeasonalForest:
		return GenerateFlavorText(tropicalSeasonalForestDescription)
	case genbiome.WhittakerModBiomeTemperateRainforest:
		return GenerateFlavorText(temperateRainforestDescription)
	case genbiome.WhittakerModBiomeTemperateSeasonalForest:
		return GenerateFlavorText(temperateSeasonalForestDescription)
	case genbiome.WhittakerModBiomeWoodlandShrubland:
		return GenerateFlavorText(shrublandDescription)
	case genbiome.WhittakerModBiomeBorealForestTaiga:
		return GenerateFlavorText(borealForestDescription)
	case genbiome.WhittakerModBiomeTundra:
		return GenerateFlavorText(tundraDescription)
	case genbiome.WhittakerModBiomeHotSwamp:
		return GenerateFlavorText(hotSwampDescription)
	case genbiome.WhittakerModBiomeWetlands:
		return GenerateFlavorText(temperateWetlandDescription)
	case genbiome.WhittakerModBiomeSavannah:
		return GenerateFlavorText(savannahDescription)
	case genbiome.WhittakerModBiomeSnow:
		return GenerateFlavorText(snowDescription)
	default:
		return "The " + genbiome.WhittakerModBiomeToString(biome) + " stretches out as far as the eye can see."
	}
}

var desertDescription = BiomeDescription{
	Adjectives: []string{"arid", "bleak", "dry", "hot", "scorching", "sizzling", "sunny", "torrid"},
	Nouns:      []string{"desert", "dune", "oasis", "sand", "sand dune", "sandstorm", "sands", "wasteland"},
	Part1: []string{
		"The hot sun beats down relentlessly, baking the earth and sapping the energy from all who dare to venture into the wilderness.",
		"The dry air is thin and crisp, with a biting wind that cuts through clothing and chaps the skin.",
	},
	Part2: []string{
		"Mirages dance on the horizon, tempting travelers with the promise of water and shade.",
		"The only respite from the searing heat is found in the scarce oases that dot the landscape, offering a brief moment of coolness before the journey continues.",
	},
	Part3: []string{
		"The subtropical desert is home to a diverse array of plants and animals, including cacti, small mammals, and reptiles.",
		"The subtropical desert is a harsh and unforgiving place, with only the most resilient species able to survive the extreme conditions.",
	},
}

var coldDesertDescription = BiomeDescription{
	Adjectives: []string{"arid", "bleak", "cold", "frigid", "frozen", "icy", "snowy", "windy"},
	Nouns:      []string{"desert", "dune", "tundra", "wasteland"},
	Part1: []string{
		"The ground is dry and rocky, with patches of scrubby grass and scattered clusters of bushes and cacti dotting the landscape.",
		"The ground is cracked and parched, with only a few hardy plants able to survive the harsh conditions.",
	},
	Part2: []string{
		"The air is thin and biting, making it hard to catch one's breath.",
		"The frigid landscape is a stark and barren wasteland, with barely a sign of life for miles.",
	},
	Part3: []string{
		"The cold desert is home to a small array of plants and animals, including succulents, small mammals, and reptiles.",
		"The cold desert is a harsh and unforgiving place, with only the most resilient species able to survive the extreme conditions.",
	},
}

var temperateWetlandDescription = BiomeDescription{
	Adjectives: []string{"damp", "humid", "marshy", "moist", "muddy", "soggy", "swampy"},
	Nouns:      []string{"marsh", "swamp", "wetland", "landscape"},
	Part1: []string{
		"The air is heavy with the smell of mud and decay, as waterlogged plants and animals rot in the sun.",
		"The ground is soft and spongy, giving way beneath the feet with each step.",
	},
	Part2: []string{
		"The sky is overcast and gray, with a constant drizzle that keeps everything damp and humid.",
		"The sun shines brightly, illuminating the lush greenery and sparkling waters of the wetlands.",
	},
	Part3: []string{
		"The wetlands are home to a diverse array of plants and animals, including tall reeds, swaying grasses, and brightly-colored flowers.",
		"The wetlands are teeming with life, from the smallest insects to the largest predators.",
	},
}

var hotSwampDescription = BiomeDescription{
	Adjectives: []string{"damp", "humid", "marshy", "moist", "muddy", "soggy", "swampy", "tropical"},
	Nouns:      []string{"marsh", "swamp", "wetland", "bog"},
	Part1: []string{
		"The air is heavy with the smell of mud and decay, as waterlogged plants and animals rot in the hot sun.",
		"The ground is soft and spongy, giving way beneath the feet with each step.",
	},
	Part2: []string{
		"The sun beats down mercilessly, turning the swamp into a steamy, mosquito-infested hellscape.",
		"The humidity is suffocating, making it hard to catch one's breath.",
	},
	Part3: []string{
		"The swamp is home to a diverse array of plants and animals, including tall reeds, swaying grasses, and brightly-colored flowers.",
		"The swamp is teeming with life, from the smallest insects to the largest predators.",
	},
}

var temperateRainforestDescription = BiomeDescription{
	Adjectives: []string{"damp", "humid", "lush", "moist", "rainy", "verdant"},
	Nouns:      []string{"forest", "rainforest", "jungle"},
	Part1: []string{
		"The air is thick with the sound of dripping water, as rain patters against the leaves and falls to the ground.",
		"The ground is soft and spongy, muffling the footsteps of those who walk among the trees.",
	},
	Part2: []string{
		"The canopy overhead is dense and impenetrable, blocking out most of the sunlight and creating a dim and shadowy realm.",
		"The sunlight filters through the leaves in shafts of gold, casting a warm and peaceful glow over the forest.",
	},
	Part3: []string{
		"The rainforest is home to a diverse array of plants and animals, including tall trees, colourful flowers, and exotic birds.",
		"The rainforest teems with life, from the tiniest insects to the largest mammals.",
	},
}

var tropicalRainforestDescription = BiomeDescription{
	Adjectives: []string{"damp", "humid", "lush", "moist", "rainy", "verdant", "tropical"},
	Nouns:      []string{"forest", "rainforest", "jungle"},
	Part1: []string{
		"The air is thick with the sound of dripping water, as rain patters against the leaves and falls to the ground.",
		"The ground is soft and spongy, muffling the footsteps of those who walk among the trees.",
	},
	Part2: []string{
		"The canopy overhead is dense and impenetrable, blocking out most of the sunlight and creating a dim and shadowy realm.",
		"The sunlight filters through the leaves in shafts of gold, casting a warm and peaceful glow over the forest.",
	},
	Part3: []string{
		"The rainforest is home to a diverse array of plants and animals, including tall trees, colourful flowers, and exotic birds.",
		"The rainforest teems with life, from the tiniest insects to the largest mammals.",
	},
}

var shrublandDescription = BiomeDescription{
	Adjectives: []string{"brushy", "scrubby", "thick", "verdant"},
	Nouns:      []string{"scrubland", "shrubland", "thicket"},
	Part1: []string{
		"The ground is covered in a thick layer of brush and scrub, making it difficult to traverse.",
		"The shrubs and bushes are dense and impenetrable, blocking out most of the sunlight and creating a dim and shadowy realm.",
	},
	Part2: []string{
		"The air is dry and dusty, with a constant breeze rustling the leaves of the shrubs.",
		"The air is heavy and still, with the sound of buzzing insects filling the air.",
	},
	Part3: []string{
		"The shrubland is home to a diverse array of plants and animals, including hardy grasses, colorful flowers, and small mammals.",
		"The shrubland is dotted with clusters of hardy bushes and low-lying trees, providing shelter and food for a variety of species.",
	},
}

var temperateGrasslandDescription = BiomeDescription{
	Adjectives: []string{"flat", "grassy", "rolling", "verdant"},
	Nouns:      []string{"grassland", "meadow", "prairie"},
	Part1: []string{
		"The grass sways gently in the breeze, creating a sea of green that stretches out to the horizon.",
		"The ground is flat and open, with only a few scattered clusters of bushes and trees dotting the landscape.",
	},
	Part2: []string{
		"The air is fresh and clean, with a light breeze carrying the scent of wildflowers.",
		"The sun beats down mercilessly, baking the earth and turning the grassland into a dry, parched wasteland.",
	},
	Part3: []string{
		"The grassland is home to a diverse array of plants and animals, including tall grasses, colourful wildflowers, and small mammals.",
		"The grassland is teeming with life, from the smallest insects to the largest predators.",
	},
}

var tundraDescription = BiomeDescription{
	Adjectives: []string{"barren", "frozen", "icy", "snowy", "treeless"},
	Nouns:      []string{"tundra", "wasteland", "waste"},
	Part1: []string{
		"The ground is frozen solid, with a layer of snow covering the surface.",
		"The snow is deep and drifts are piled high, making it difficult to traverse the landscape.",
	},
	Part2: []string{
		"The air is crisp and cold, biting at the exposed skin.",
		"The air is still and silent, with a deathly chill that seems to seep into the bones.",
	},
	Part3: []string{
		"The tundra is home to a hardy array of plants and animals, including lichens, mosses, and small mammals.",
		"The tundra is a barren and lifeless place, with only a few scattered patches of vegetation surviving in the harsh conditions.",
	},
}

var savannahDescription = BiomeDescription{
	Adjectives: []string{"arid", "dry", "grassy", "savannah"},
	Nouns:      []string{"plain", "landscape", "expanse"},
	Part1: []string{
		"The ground is dry and dusty, with patches of grass and scattered clusters of bushes and trees dotting the landscape.",
		"The grass is tall and waving, with a few scattered trees providing shade and shelter.",
	},
	Part2: []string{
		"The sun beats down mercilessly, baking the earth and turning the savannah into a dry, parched wasteland.",
		"The air is dry and hot, with a light breeze providing some relief from the scorching sun.",
	},
	Part3: []string{
		"The savannah is home to a diverse array of plants and animals, including tall grasses, colourful wildflowers, and large mammals.",
		"The savannah teems with life, from the smallest insects to the largest predators.",
	},
}

var borealForestDescription = BiomeDescription{
	Adjectives: []string{"dense", "frozen", "snowy"},
	Nouns:      []string{"forest", "trees", "taiga"},
	Part1: []string{
		"The canopy overhead is dense and impenetrable, blocking out most of the sunlight and creating a dim and shadowy realm.",
		"The sunlight filters through the leaves in shafts of gold, casting a warm and peaceful glow over the forest.",
	},
	Part2: []string{
		"The ground is frozen solid, with a layer of snow covering the surface.",
		"The snow is deep and drifts are piled high, making it difficult to traverse the landscape.",
	},
	Part3: []string{
		"The air is crisp and cold, biting at the exposed skin.",
		"The air is still and silent, with a deathly chill that seems to seep into the bones.",
	},
	Part4: []string{
		"The boreal forest is home to a hardy array of plants and animals, including conifers, small mammals, and migratory birds.",
		"The boreal forest is a harsh and unforgiving place, with only the most resilient species able to survive the long, cold winters.",
	},
}

var temperateSeasonalForestDescription = BiomeDescription{
	Adjectives: []string{"dense", "lush", "temperate"},
	Nouns:      []string{"forest", "woods"},
	Part1: []string{
		"The canopy overhead is dense and verdant, with a riot of leaves and branches creating a green, leafy ceiling.",
		"The sunlight filters through the leaves in dappled patterns, casting a warm and peaceful glow over the forest.",
	},
	Part2: []string{
		"The ground is soft and spongy, with a thick layer of leaves and debris covering the surface.",
		"The ground is hard and rocky, with only a few patches of moss and lichen providing a splash of color.",
	},
	Part3: []string{
		"The air is fresh and clean, with a light breeze carrying the scent of pine and wood smoke.",
		"The air is humid and thick, with the sound of buzzing insects and chirping birds filling the air.",
	},
	Part4: []string{
		"The forest is home to a diverse array of plants and animals, including deciduous trees, colourful wildflowers, and small mammals.",
		"The woods are teeming with life, from the smallest insects to the largest predators.",
	},
}

var tropicalSeasonalForestDescription = BiomeDescription{
	Adjectives: []string{"dense", "lush", "tropical"},
	Nouns:      []string{"forest", "jungle", "rainforest"},
	Part1: []string{
		"The canopy overhead is dense and verdant, with a riot of leaves and branches creating a green, leafy ceiling.",
		"The sunlight filters through the leaves in dappled patterns, casting a warm and peaceful glow over the forest.",
	},
	Part2: []string{
		"The ground is soft and spongy, with a thick layer of leaves and debris covering the surface.",
		"The ground is hard and rocky, with only a few patches of moss and lichen providing a splash of color.",
	},
	Part3: []string{
		"The air is hot and humid, with the sound of buzzing insects and chirping birds filling the air.",
		"The air is heavy with the scent of flowers and the sound of distant thunder, as a tropical rainstorm approaches.",
	},
	Part4: []string{
		"The forest is home to a diverse array of plants and animals, including tall trees, colourful wildflowers, and exotic creatures.",
		"The jungle is teeming with life, from the smallest insects to the largest predators.",
	},
}

var snowDescription = BiomeDescription{
	Adjectives: []string{"frozen", "glacial", "icy", "snowy"},
	Nouns:      []string{"plain", "wasteland", "landscape"},
	Part1: []string{
		"The ground is frozen solid, with a layer of snow covering the surface.",
		"The snow is deep and drifts are piled high, making it difficult to traverse the terrain.",
	},
	Part2: []string{
		"The air is crisp and cold, biting at the exposed skin.",
		"The air is still and silent, with a deathly chill that seems to seep into the bones.",
	},
	Part3: []string{
		"The snowy environment is barren and lifeless, with only a few hardy plants and animals able to survive the harsh conditions.",
		"The snowy landscape is home to a hardy array of plants and animals, including lichens, mosses, and small mammals.",
	},
}
