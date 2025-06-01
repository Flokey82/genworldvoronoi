package civ

import (
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/Flokey82/go_gens/genlanguage"
	"github.com/Flokey82/go_gens/genstory"
	"github.com/Flokey82/go_gens/genstory/genbooks"
)

var artifactTextGen = genstory.NewTextGenerator(rand.New(rand.NewSource(time.Now().UnixNano())))

var artifactID int

func nextArtifactID() int {
	artifactID++
	return artifactID
}

// Artifact represents an artifact that can be found in the world.
// TODO:
// - Condition like "pristine", "tarnished", "broken"...
// - Value (priceless, valuable, cheap, worthless)
// - History (who made it, who owned it, where it was found)
type Artifact struct {
	ID        int
	Name      string // The name of the artifact
	Condition *Condition
}

// NewArtifact creates a new artifact.
func NewArtifact(name string) *Artifact {
	return &Artifact{
		ID:   nextArtifactID(),
		Name: name,
	}
}

// Ref returns the object reference of the artifact.
func (a *Artifact) Ref() ObjectReference {
	return ObjectReference{
		ID:   a.ID,
		Type: ObjectTypeArtifact,
	}
}

// NameWithArticle returns the name of the artifact with an article.
func (a *Artifact) NameWithArticle() string {
	// First we extract the first word.
	firstWord := strings.Split(a.Name, " ")[0]

	// Add the article.
	return genlanguage.GetArticle(firstWord) + " " + a.Name
}

var (
	bookTitleConfig            = genbooks.NewSimpleTitleConfig(genbooks.BookVariantTitles)
	bookTitleConfigInstruction = genbooks.NewSimpleTitleConfig(genbooks.BookInstructionTitles)
)

// NewBook creates a new book.
func NewBook() *Artifact {
	// Set the seed for the text generator.
	id := nextArtifactID()
	artifactTextGen.Seed(int64(id))

	// Generate the title.
	title, err := artifactTextGen.GenerateFromConfig(nil, bookTitleConfig, nil)
	if err != nil {
		log.Println("error generating book title:", err)
	}

	return &Artifact{
		ID:   id,
		Name: "book: " + title.Text,
	}
}

// NewBookInstruction creates a new book with instructions.
func NewBookInstruction(name string) *Artifact {
	// Set the seed for the text generator.
	id := nextArtifactID()
	artifactTextGen.Seed(int64(id))

	// Generate the title.
	title, err := artifactTextGen.GenerateFromConfig([]genstory.TokenReplacement{{
		Token:       genbooks.TokenName,
		Replacement: name,
	}}, bookTitleConfigInstruction, nil)
	if err != nil {
		log.Println("error generating book title:", err)
	}

	return &Artifact{
		ID:   id,
		Name: title.Text,
	}
}

var (
	treasureNameConfig = &genstory.TextConfig{
		TokenPools: map[string][]string{
			"[ADJECTIVE]": {
				"shiny",
				"old",
				"mysterious",
				"strange",
				"ancient",
				"magical",
				"glowing",
				"dull",
				"rusty",
			},
			"[OBJECT]": {
				"box",
				"doll",
				"coin",
				"amulet",
				"necklace",
				"ring",
				"crown",
				"statuette",
				"mirror",
				"book",
				"scroll",
				"potion",
				"key",
				"diary",
				"letter",
				"drawing",
				"shard",
				"eye",
				"locket",
				"device",
				"helmet",
			},
			"[FEATURE]": {
				"engraving",
				"marking",
				"inscription",
				"symbol",
				"rune",
				"carving",
				"etching",
			},
			"[FEATURE_ADJECTIVE]": {
				"strange",
				"ancient",
				"mysterious",
				"magical",
				"glowing",
			},
		},
		TokenIsMandatory: map[string]bool{},
		Tokens: []string{
			"[ADJECTIVE]",
			"[OBJECT]",
			"[FEATURE]",
			"[FEATURE_ADJECTIVE]",
		},
		Templates: []string{
			"[ADJECTIVE] [OBJECT]",
			"[OBJECT]",
			"[ADJECTIVE] [OBJECT] with [FEATURE_ADJECTIVE:a] [FEATURE]",
			"[ADJECTIVE] [OBJECT] with [FEATURE:a]",
		},
		Modifiers: map[string]func(string) string{},
	}

	treasureNameRules = &genstory.Rules{
		Expansions: map[string][]string{
			"object": {
				"box",
				"doll",
				"coin",
				"amulet",
				"necklace",
				"ring",
				"crown",
				"statuette",
				"mirror",
				"book of [writing]",
				"scroll",
				"bundle of letters on [writing]",
				"book",
				"potion",
				"key",
				"diary",
				"letter",
				"drawing",
				"shard",
				"eye",
				"locket",
				"device",
				"helmet",
			},
			"writing": {
				"spells",
				"runes",
				"poems",
				"poetry",
				"fables",
				"fairy tales",
				"legends",
				"prophecies",
				"ballads",
				"beasts",
				"monsters",
				"dragons",
				"creatures",
			},
			"adjective": {
				"shiny",
				"old",
				"mysterious",
				"strange",
				"ancient",
				"magical",
				"glowing",
				"dull",
				"rusty",
			},
			"feature": {
				"[feature_adjective:a] [feature_type] [feature_location]",
				"[feature_adjective:a] [feature_type]",
				"[feature_type:a]",
			},
			"feature_type": {
				"engraving",
				"marking",
				"inscription",
				"symbol",
				"rune",
				"carving of [drawing_adjective:a] [drawing_subject]",
				"etching of [drawing_adjective:a] [drawing_subject]",
				"drawing of [drawing_adjective:a] [drawing_subject]",
			},
			"drawing_adjective": {
				"beautiful",
				"strange",
				"ancient",
				"mysterious",
				"magical",
				"glowing",
				"young",
				"old",
				"ugly",
				"misshapen",
			},
			"drawing_subject": {
				"woman",
				"girl",
				"man",
				"lady",
				"king",
				"queen",
				"prince",
				"princess",
				"flower",
				"tree",
				"animal",
				"monster",
				"dragon",
				"elf",
				"dwarf",
				"orc",
				"troll",
				"gnome",
				"halfling",
				"wizard",
				"witch",
				"mage",
				"warrior",
			},
			"feature_adjective": {
				"strange",
				"ancient",
				"mysterious",
				"magical",
				"glowing",
				"pulsing",
				"shimmering",
				"dark",
				"light",
				"bright",
				"faint",
			},
			"feature_location": {
				"on the back",
				"on the front",
				"on the side",
				"allong the edge",
				"on the top",
				"on the bottom",
			},
			"object_phrase": {
				"[adjective] [object] with [feature]",
				"[adjective] [object]",
				"[object]",
			},
		},
		Start: "[object_phrase]",
	}
)

// NewTreasure creates a new treasure.
func NewTreasure() *Artifact {
	useConfig := false
	id := nextArtifactID()
	// Dev notes:
	// - "a small, shiny box"
	// - "a mysterious doll"
	// - "an ancient coin with strange markings"
	var artifactName string
	if useConfig {
		trRes, err := artifactTextGen.GenerateFromConfig(nil, treasureNameConfig, nil)
		if err != nil {
			log.Println("error generating treasure:", err)
		}
		artifactName = trRes.Text
	} else {
		// Now we do the same with grammar rules.
		res := treasureNameRules.NewStory(int64(id))
		trRes, err := res.Expand()
		if err != nil {
			log.Println("error expanding story:", err)
		}
		artifactName = trRes
	}

	// Construct a story.
	// - ... moved a stone and found a small, shiny box.
	// - ... discovered a mysterious doll hidden in a hollow stump.
	// - ... found an ancient coin with strange markings.
	art := &Artifact{
		ID:   id,
		Name: artifactName,
	}

	// Check if it is cursed.
	const (
		outcomeCursed  = 0
		outcomeBlessed = 1
	)

	switch rand.Intn(10) {
	case outcomeCursed:
		art.Name += " (cursed)"

		// Pick a random curse.
		curses := []*Condition{
			curseLazinessAndCarelessness,
			curseCrueltyAndDeception,
			curseAmbitionAndCruelty,
			curseParanoiaCrueltyAndDeception,
			curseOfCursedFamily,
		}
		art.Condition = curses[rand.Intn(len(curses))]
	case outcomeBlessed:
		art.Name += " (blessed)"

		// Pick a blessing.
		art.Condition = blessingOfTheBrave
	}
	return art
}
