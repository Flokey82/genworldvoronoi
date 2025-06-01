package civ

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/Flokey82/genetics"
	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/go_gens/gameconstants"
	"github.com/Flokey82/go_gens/genlanguage"
)

func (m *Civ) getNextPersonID() int {
	m.nextPersonID++
	return m.nextPersonID
}

func (m *Civ) getPeopleAt(r int) []*Person {
	return m.PeopleToRegion.GetValues()[r]
}

func (m *Civ) getPeopleAtRegionsNoCache(peopleToRegion [][]*Person) [][]*Person {
	if len(peopleToRegion) != m.NumRegions {
		peopleToRegion = make([][]*Person, m.NumRegions)
	}
	for i := range peopleToRegion {
		// TODO: Find a better way to scale up the capacity of each slice.
		if len(peopleToRegion[i]) == 0 {
			continue
		}

		// If we have only 20% free space, we scale up.
		if cap(peopleToRegion[i]) < len(peopleToRegion[i])*5/4 {
			// The capacity is too small, we need to scale up.
			peopleToRegion[i] = make([]*Person, 0, cap(peopleToRegion[i])*2)
		} else {
			// Trim the slice.
			peopleToRegion[i] = peopleToRegion[i][:0]
		}
	}
	for _, p := range m.People {
		// Check if the slice is uninitialized.
		if cap(peopleToRegion[p.Region]) == 0 {
			peopleToRegion[p.Region] = make([]*Person, 0, 1024)
		}
		peopleToRegion[p.Region] = append(peopleToRegion[p.Region], p)
	}
	return peopleToRegion
}

// tickPerson advances the person by nDays and returns any new born children.
// TODO: Twins, triplets, etc.
func (m *Civ) tickPerson(p *Person, nDays int, cf func(int) *Culture) *Person {
	if !m.doesPersonExist(p) {
		return nil
	}
	// Calculate age.
	m.tickPersonAge(p, nDays)

	// Advance pregnancy.
	var child *Person
	if p.Prengancy != nil {
		child = m.tickPersonPregnancy(p, nDays, cf)
	}

	// Check if person dies of natural causes.
	m.tickPersonDeath(p, nDays)

	// If the person is dead, we don't need to do anything else.
	if p.Death.IsSet() {
		return nil
	}

	// Check existing relationships.
	{
		// Check who we love or hate in our family.
		var hated, loved []relation
		for _, c := range pickRelatives(p) {
			// Check if we hate the person.
			if p.Opinions.GetOpinion(c.p) < -0.5 {
				hated = append(hated, c)
			} else if p.Opinions.GetOpinion(c.p) > 0.5 {
				loved = append(loved, c)
			}
		}

		// Add an entry to the history.
		if len(hated) > 0 {
			var str string
			for _, h := range hated {
				str += fmt.Sprintf("%s (%s %.2f), ", h.p.String(), h.relationOfPerson, p.Opinions.GetOpinion(h.p))
			}
			str = str[:len(str)-2]
			m.AddEvent("Hate", fmt.Sprintf("%s hates %s", p.String(), str), p.Ref())
		}
		if len(loved) > 0 {
			var str string
			for _, h := range loved {
				str += fmt.Sprintf("%s (%s %.2f), ", h.p.String(), h.relationOfPerson, p.Opinions.GetOpinion(h.p))
			}
			str = str[:len(str)-2]
			m.AddEvent("Love", fmt.Sprintf("%s loves %s", p.String(), str), p.Ref())
		}

		// Check if we have a nemesis.
		nemesis := p.Opinions.GetNemesis()
		if nemesis != nil {
			m.AddEvent("Nemesis", fmt.Sprintf("%s has a nemesis: %s (%.2f)", p.String(), nemesis.String(), p.Opinions.GetOpinion(nemesis)), p.Ref())
		}
	}

	m.tickFamily(p, nDays)
	return child
}

func (m *Civ) tickFamily(p *Person, nDays int) {
	/*
		// TODO: Select family actions depending on the number of days.
		// If we tick a whole year, we might pick from more impactful actions, etc.
		const (
			emoLove    = "love"
			emoHate    = "hate"
			emoLike    = "like"
			emoDislike = "dislike"
		)

		getEmotion := func(op float64) string {
			if op < -0.5 {
				return emoHate
			} else if op < 0.0 {
				return emoDislike
			} else if op < 0.5 {
				return emoLike
			}
			return emoLove
		}

		getCareThreshold := func(p *Person) float64 {
			careThreshold := 0.0
			if p.Traits.HasTrait(geneticshuman.TraitKind) {
				careThreshold -= 0.2
			}
			if p.Traits.HasTrait(geneticshuman.TraitCruel) {
				careThreshold += 0.5
			}
			return careThreshold
		}

		doesCare := func(op float64, p *Person) bool {
			return op > getCareThreshold(p)
		}

		pickAction := func(otherPerson *Person, actions []string, relation string) {
			if otherPerson.isDead() {
				// TODO: We might visit the grave, pray for them, etc.
				return
			}

			// Check how much we like the other person.
			op := p.Opinions.GetOpinion(otherPerson)

			// Calculate the threshold over which we actually care about the opinion of the other person.
			doWeCare := doesCare(op, p)

			// If we care about the opinion of the other person, we might do something about it.
			if doWeCare {
				// log.Printf("!=! %s %s %s %s (%.2f) and cares", p.String(), opEmo, relation, otherPerson.String(), op)
				action := actions[rand.Intn(len(actions))]

				// Get the emotion that we have of the other person.
				opEmo := getEmotion(op)

				// Check if the other person cares about us, otherwise the action might backfire.
				// But there is a chance that the action might still be successful.
				if otherPersonOp := otherPerson.Opinions.GetOpinion(p); doesCare(otherPersonOp, otherPerson) || rand.Intn(100) > 90 {
					ev := m.AddEvent("Action", fmt.Sprintf("%s %s for their %s %s which they %s", p.String(), action, relation, otherPerson.String(), opEmo), p.Ref())
					p.Opinions.AddOpinion(otherPerson, 0.1, ev)
					otherPerson.Opinions.AddOpinion(p, 0.1, ev)
				} else {
					ev := m.AddEvent("Action", fmt.Sprintf("%s %s for their %s %s which they %s, but they didn't care for it", p.String(), action, relation, otherPerson.String(), opEmo), p.Ref())
					p.Opinions.AddOpinion(otherPerson, -0.1, ev)
					otherPerson.Opinions.AddOpinion(p, -0.1, ev)
				}
			} else {
				// log.Printf("!=! %s %s %s %s (%.2f) but doesn't care", p.String(), opEmo, relation, otherPerson.String(), op)

				// TODO: Pick from actions that don't require caring.
				// This might be even hostile actions, but it might also just be ignoring the other person.
				// If the other person cares about us, they might be hurt by us ignoring them.
			}
		}

		// Check if we are in the same region as our family.

		// First, handle the spouse (if we have one).
		if p.Spouse != nil {
			pickAction(p.Spouse, []string{"kisses", "hugs", "caresses", "compliments", "flirts with"}, "spouse")
		}

		// Check if we have children.
		for _, c := range p.Children {
			pickAction(c, []string{"hugs", "caresses", "compliments", "praises", "plays with"}, "child")
		}

		// Check if we have parents.
		if p.Mother != nil {
			pickAction(p.Mother, []string{"hugs", "caresses", "compliments", "praises", "thanks"}, "mother")
		}
		if p.Father != nil {
			pickAction(p.Father, []string{"hugs", "caresses", "compliments", "praises", "thanks"}, "father")
		}

		// TODO: Parents, children, siblings, etc.
	*/
}

const (
	ageOfAdulthood     = 18
	ageEndChildbearing = 45
)

func (m *Civ) tickPersonAge(p *Person, nDays int) {
	// Calculate current age.
	if m.History.GetDayOfYear() < p.Birth.Day {
		p.Age = int(m.History.GetYear()) - p.Birth.Year - 1
	} else {
		p.Age = int(m.History.GetYear()) - p.Birth.Year
	}
}

func (m *Civ) tickPersonDeath(p *Person, nDays int) {
	// Check if person dies of natural causes.
	if !gameconstants.DiesAtAgeWithinNDays(p.Age, nDays) || p.Death.IsSet() {
		return
	}
	// If the person just gave birth, we note that the person
	// died during childbirth.
	options := []string{
		"illness",
		"an accident",
		"a mysterious cause",
		"a heart attack",
		"a stroke",
		"a fall",
		"a lightning strike",
		"a snake bite",
		"a wild animal attack",
		"a drowning",
		"a fire",
		"a poisoning",
	}

	// Depending on the personality, the person might die of different causes.
	// For example, a person with low agreeableness might die in a duel, while
	// a person with low conscientiousness might die in an accident.

	if p.Age > 14 {
		options = append(options, "old age")

		// High aggression might lead to death in a duel, fight, or challenging a wild animal.
		if p.Traits.HasTrait(geneticshuman.TraitAggressive) {
			options = append(options, "a duel", "a fight", "challenging a wild animal")
			options = append(options, "a heart attack", "a stroke")
		}

		// Carelessness might lead to death in an accident, eating spoiled food, or a fall.
		if p.Traits.HasTrait(geneticshuman.TraitCareless) {
			options = append(options, "an accident", "eating spoiled food", "a fall")
		}

		// Carelessness might lead to death during new experiences.
		// - Poisioning from trying new food.
		// - Dying duing extreme sports.
		// - Drowning while swimming.
		if p.Traits.HasTrait(geneticshuman.TraitCareless) {
			options = append(options, "a mysterious cause", "a poisoning")
		}

		// Trusting might lead to death by being deceived, robbed, or poisoned.
		if p.Traits.HasTrait(geneticshuman.TraitTrusting) {
			options = append(options, "being deceived", "being robbed", "being poisoned")
		}
	}
	m.killPerson(p, options[rand.Intn(len(options))]) // Random cause?
}

// handleInheritance handles the inheritance of a person.
func (m *Civ) handleInheritance(p *Person) {
	// NOTE: This should depend on where and how the person died.
	// If the location is unreachable or unknown, all items in the inventory
	// might be lost or randomly found by others.
	// If the person died at home and there are no inheritors, the items
	// can be found at the location.

	// Sort relatives by opinion.
	// TODO:
	// - Add friends as well.
	// - Add a will.
	// - Add greedy people who might try to steal the inheritance.
	// - The rules of inheritance should depend on the culture.
	relatives := pickRelatives(p)
	sort.Slice(relatives, func(i, j int) bool {
		return p.Opinions.GetOpinion(relatives[i].p) > p.Opinions.GetOpinion(relatives[j].p)
	})

	// Transfer artifacts and gold to the most loved people.
	if len(relatives) == 0 || (len(relatives) == 1 && relatives[0].p == p) {
		// No relatives, so we should hide the artifacts and gold for others to find.
		// Any home of the person should become abandoned and if unmainatined
		// fall to ruin, where the artifacts might be found.
		return
	}
}

func (m *Civ) killPerson(p *Person, reason string) *Event {
	p.Death.Day = int(m.History.GetDayOfYear())
	p.Death.Year = int(m.History.GetYear())
	p.Death.Region = p.Region

	// If they have a spouse, unset their spouse.
	if p.Spouse != nil {
		p.Spouse.Spouse = nil
	}

	// :(
	if p.Prengancy != nil {
		m.killPerson(p.Prengancy, fmt.Sprintf("mother %s dying due to %s", p.Name(), reason))
	}

	// Transfer inheritance.
	m.handleInheritance(p)

	var deathStr string
	name := p.Name()
	if name == "" {
		if p.Birth.IsSet() {
			name = "unknown person"
		} else if p.Gender() == geneticshuman.GenderFemale {
			name = "unborn girl"
		} else if p.Gender() == geneticshuman.GenderMale {
			name = "unborn boy"
		} else {
			name = "unborn child"
		}
	}
	if reason == "" {
		deathStr = fmt.Sprintf("%s died at age %d", name, p.Age)
	} else {
		deathStr = fmt.Sprintf("%s died at age %d due to %s", name, p.Age, reason)
	}
	return m.AddEvent("Death", deathStr, p.Ref())
}

func (m *Civ) doesPersonExist(p *Person) bool {
	return !p.Death.IsSet() && p.Birth.IsSet() && p.Birth.Year < int(m.History.GetYear())
}

func (m *Civ) updatePersonLocation(p *Person, r int) {
	// Update location.
	// NOTE: We should differentiate between people who live in the city and
	// people who work in or visit the city.
	p.Region = r
	// p.City = m.GetCity(r)
	// TODO: Add person to city population?
}

// LifeEvent represents a date and place in the world.
type LifeEvent struct {
	Year   int
	Day    int
	Region int
}

// IsSet returns true if the life event is set.
func (l LifeEvent) IsSet() bool {
	return l.Year != 0 || l.Day != 0 || l.Region != 0
}

// Person represents a person in the world.
// TODO: Improve efficiency of this struct.
//   - We could drop age, and use day-ticks for birth and death instead.
//   - Also, we can get the gender directly from the genes.
//   - We might be able to drop the pregnancy counter and use the birth life event
//     of the child as a counter.
//   - We can use use the Region for the location and derive the city from that.
//   - A lot of this stuff is identical to simvillage_simple, so we could probably
//     merge the person logic somehow, or move it to a separate package.
type Person struct {
	ID          int                      // ID of the person
	Region      int                      // Location of the person
	Genes       genetics.Genes           // Genes.
	Personality geneticshuman.FiveFactor // Personality of the person
	Traits      geneticshuman.Trait      // Traits of the person
	City        *City                    // City of the person
	Culture     *Culture                 // Culture of the person
	Opinions    *Opinions                // Opinions of the person
	Popularity  ClampedVal               // Popularity of the person (reputation, 0.0-1.0)
	Karma       ClampedVal               // Karma of the person (good/bad deeds, 0.0-1.0)
	// Heroism     ClampedVal               // Heroism of the person (0.0-1.0)
	// Villainy    ClampedVal               // Villainy of the person (0.0-1.0)

	// Todo: Allow different naming conventions.
	FirstName string
	LastName  string
	NickName  string
	Title     string // Title of the person. TODO: Improve this.

	// Birth, death...
	// TODO: Add death cause.
	Age   int // Age of the person.
	Birth LifeEvent
	Death LifeEvent

	// Pregnancy
	PregnancyCounter int     // Days of pregnancy
	Prengancy        *Person // baby (TODO: twins, triplets, etc.)

	// Gold.
	Gold float64

	// Family (TODO: Distinguish between known and unknown family members.)
	// Maybe use a map of relations to people?
	Mother   *Person
	Father   *Person
	Spouse   *Person   // TODO: keep track of spouses that might have perished?
	Children []*Person // TODO: Split into known and unknown children.
}

func (m *Civ) newRandomPersonAt(r int, culture *Culture, gender geneticshuman.Gender, parent *Person) *Person {
	if parent != nil && parent.Age < ageOfAdulthood {
		panic("Parent is too young to have children.")
	}
	// Random genes / gender.
	var genes genetics.Genes
	if parent != nil {
		genes = genetics.Mix(parent.Genes, genetics.NewRandom(), 1)
	} else {
		genes = genetics.NewRandom()
	}
	geneticshuman.SetGender(&genes, gender)

	lang := culture.Language

	// Create the person.

	// If the first/last name pool is large enough, we should
	// start reusing names because generating new names is expensive.
	//
	// TODO: With increasing pool size, we should increase the chance
	// of reusing names.
	var firstName string
	if poolSize := lang.GetFirstNamePoolSize(); poolSize > 100 && rand.Intn(poolSize) > 10 {
		firstName = lang.GetFirstName()
	}
	if firstName == "" {
		firstName = lang.MakeFirstName()
	}

	// Same for last names.
	var lastName string
	if parent != nil {
		lastName = parent.LastName
	} else if poolSize := lang.GetLastNamePoolSize(); poolSize > 300 && rand.Intn(poolSize) > 10 {
		lastName = lang.GetLastName()
	}
	if lastName == "" {
		lastName = lang.MakeLastName()
	}

	// Infer personality and traits from genes.
	fiveFactor := geneticshuman.GetFiveFactor(&genes)
	traits := geneticshuman.GetTraits(fiveFactor)

	// Pick an age.
	var age int
	if parent != nil {
		age = max(rand.Intn(parent.Age-16), parent.Age/2) // Pick a child age.
	} else {
		age = ageOfAdulthood + rand.Intn(2*ageOfAdulthood) // Pick an adult age.
	}

	p := &Person{
		ID:          m.getNextPersonID(),
		Culture:     culture,
		Opinions:    NewOpinions(),
		Genes:       genes,
		FirstName:   firstName,
		LastName:    lastName,
		Personality: fiveFactor,
		Traits:      traits,
		Age:         age,
		Birth: LifeEvent{
			Year:   int(m.History.GetYear()) - age,
			Day:    rand.Intn(365),
			Region: r, // TODO: Pick a birth region that makes sense.
		},
	}

	// Assign as child to parent.
	if parent != nil {
		if parent.Gender() == GenderFemale {
			p.Mother = parent
		} else {
			p.Father = parent
		}
		parent.Children = append(parent.Children, p)
	}

	// Update location.
	m.updatePersonLocation(p, r)

	// TODO: Random spouse, children, etc.?
	m.People = append(m.People, p)
	return p
}

// Ref returns the object reference of the person.
func (p *Person) Ref() ObjectReference {
	return ObjectReference{
		ID:   p.ID,
		Type: ObjectTypePerson,
	}
}

// String returns a string representation of the leader.
func (p *Person) String() string {
	name := p.Name()
	if p.Title != "" {
		name = p.Title + " " + name
	}
	gender := "°"
	if p.Gender() == geneticshuman.GenderFemale {
		gender = "♀"
	} else if p.Gender() == geneticshuman.GenderMale {
		gender = "♂"
	}
	isDead := " "
	if p.Dead() {
		isDead = "†"
	}
	str := fmt.Sprintf("%s (%s%s%d) (P:%.1f, K:%.1f, G:%.1f)", name, gender, isDead, p.Age, p.Popularity, p.Karma, p.Gold)
	if p.Traits != 0 {
		str += " [" + p.Traits.String() + "]"
	}
	return fmt.Sprintf("(%d) %s", p.ID, str)
}

// compare returns the similarity between two people.
func (p *Person) compare(other *Person) float64 {
	if p == other {
		return 1.0
	}
	if p == nil || other == nil {
		return -1.0
	}
	cultureValue := p.Culture.compare(other.Culture)
	languageValue := compareLanguage(p.Culture.Language, other.Culture.Language)
	// religionValue := p.Religion.compare(other.Religion)
	traitValue := p.Traits.Compare(other.Traits)

	return (cultureValue + languageValue + traitValue) / 3
}

// Name returns the name of the person.
func (p *Person) Name() string {
	if p.FirstName == "" && p.LastName == "" {
		if p.NickName != "" {
			return p.NickName
		}
		return ""
	}
	if p.NickName != "" {
		return fmt.Sprintf("%s %q %s", p.FirstName, p.NickName, p.LastName)
	}
	return p.FirstName + " " + p.LastName
}

// StringGenes returns the string representation of the person.
func (p *Person) StringGenes() string {
	return geneticshuman.String(p.Genes)
}

// Dead returns true if the person is dead.
func (p *Person) Dead() bool {
	return p.Death.IsSet()
}

// Gender returns the gender of the person.
func (p *Person) Gender() geneticshuman.Gender {
	return geneticshuman.GetGender(&p.Genes)
}

func (p *Person) isOfChildbearingAge() bool {
	return p.Age >= ageOfAdulthood && p.Age < ageEndChildbearing
}

// isElegibleSingle returns true if the person is old enough to look for a partner and single.
func (p *Person) isEligibleSingle() bool {
	return p.Age > ageOfAdulthood && p.Spouse == nil // Old enough and single.
}

// canBePregnant returns true if the person is old enough and not pregnant.
func (p *Person) canBePregnant() bool {
	// Female, has a spouse (implies old enough), and is currently not pregnant.
	// TODO: Set randomized upper age limit.
	return p.Gender() == GenderFemale && p.Spouse != nil && p.Prengancy == nil
}

const pregnancyDays = 280 // for humans

func (p *Person) newPersonPregnancy(id int, father *Person) *Person {
	// Mix genes.
	var genes genetics.Genes
	if father != nil {
		genes = genetics.Mix(p.Genes, father.Genes, 1)
	} else {
		genes = genetics.Mix(p.Genes, genetics.NewRandom(), 1)
	}

	// Fix genes wrt. gender (the genetic mix doesn't limit gender varaition)
	geneticshuman.SetGender(&genes, randGender())

	fiveFactor := geneticshuman.GetFiveFactor(&genes)
	traits := geneticshuman.GetTraits(fiveFactor)

	// We need to set the name after birth, because the parents might not know the gender of the baby
	// until birth. (If there's magic, only wealthy people would be able to determine the gender before)
	child := &Person{
		ID:          id,
		Genes:       genes,
		Mother:      p,
		Father:      father,
		Personality: fiveFactor,
		Traits:      traits,
		Opinions:    NewOpinions(),
	}

	p.PregnancyCounter = pregnancyDays
	p.Prengancy = child
	return child
}

// tickPersonPregnancy advances the pregnancy of the person.
// TODO: Add twins, triplets, etc.
func (m *Civ) tickPersonPregnancy(p *Person, nDays int, cf func(int) *Culture) *Person {
	if p.Prengancy == nil {
		return nil
	}

	// Reduce pregnancy counter.
	p.PregnancyCounter -= nDays
	if p.PregnancyCounter > 0 {
		return nil
	}

	// Birth!
	wasBornNDaysAgo := -p.PregnancyCounter
	child := p.Prengancy

	// Reset pregnancy.
	p.Prengancy = nil
	p.PregnancyCounter = 0

	// Add child to family and name it.
	// We use spouse since this is the acting father.
	// TODO: Use naming convention of culture to determine if mother or father name the child.
	var lang *genlanguage.Language
	if p.Spouse != nil && rand.Intn(100) < 50 {
		lang = p.Spouse.Culture.Language
	} else {
		lang = p.Culture.Language
	}

	// There is a random chance we generate a new name, but the larger the pool
	// the less likely we are to generate a new name.
	var firstName string
	if poolSize := lang.GetFirstNamePoolSize(); poolSize > 100 && rand.Intn(poolSize) > 10 {
		firstName = lang.GetFirstName()
	}
	if firstName == "" {
		firstName = lang.MakeFirstName()
	}
	child.FirstName = firstName

	// Add child to the children of the mother.
	p.Children = append(p.Children, child)

	// Add child to the children of the "father" uhm.. spouse.
	if p.Spouse != nil {
		p.Spouse.Children = append(p.Spouse.Children, child)
	} else if p.Father != nil {
		// TODO: What if spouse != father?
		p.Father.Children = append(p.Father.Children, child)
	}

	// Use the mother's last name.
	child.LastName = p.LastName

	// Set birth date.
	child.Birth.Region = p.Region
	child.Birth.Year = int(m.History.GetYear())
	child.Birth.Day = m.History.GetDayOfYear() - wasBornNDaysAgo
	if child.Birth.Day < 0 {
		child.Birth.Year--
		child.Birth.Day += 365
		// Age the baby for the number of days it was born ago.
		m.tickPerson(child, wasBornNDaysAgo, cf)
	}

	// Set city.
	child.City = p.City
	if child.City != nil {
		child.City.People = append(child.City.People, child)
		child.Culture = child.City.Culture
	}

	// Set culture.
	// NOTE: Should this be the culture of the mother or father?
	// If mother and father are from different cultures, which one should it be?
	// If the child is born in a different region, should the culture change?
	//
	// I think it'd be great to randomly determine which culture the child
	// should have. This would ba an interesting source of conflict and story.
	if child.Culture == nil {
		child.Culture = cf(p.Region)
	}

	// Update location.
	m.updatePersonLocation(child, p.Region)

	// Add child to world.
	m.People = append(m.People, child)

	// log.Println("New person born:", child.Name())
	return child
}

// isDead returns true if the person is dead.
func (p *Person) isDead() bool {
	return p.Death.IsSet()
}

var (
	GenderFemale = geneticshuman.GenderFemale
	GenderMale   = geneticshuman.GenderMale
)

// randGender returns a random gender.
func randGender() geneticshuman.Gender {
	if rand.Intn(2) == 0 {
		return GenderFemale
	}
	return GenderMale
}

// isRelated returns true if a and b are related (up to first degree).
func isRelated(a, b *Person) bool {
	// Check if there is a parent/child relationship.
	if a == b.Father || a == b.Mother || b == a.Father || b == a.Mother {
		return true
	}

	// If either (or both) of the parents are nil, we assume that they are not related.
	if (a.Father == nil && a.Mother == nil) || (b.Father == nil && b.Mother == nil) {
		return false
	}

	// Check if there is a (half-) sibling relationship.
	return a.Mother == b.Mother || a.Father == b.Father
}

// calcRelationshipDistance calculates the distance between two people in the family tree.
func calcRelationshipDistance(a, b *Person) int {
	if a == b {
		return 0
	}

	// We just expand the relationships until we find the other person.
	queue := NewPersonQueue()
	seen := make(map[*Person]bool)
	distance := make(map[*Person]int)
	seen[a] = true
	distance[a] = 0
	queue.PushBack(a)

	// Expand the queue.
	for queue.Len() > 0 {
		p := queue.PopFront()
		if p == b {
			return distance[p]
		}

		// Add children.
		if p.Mother != nil && !seen[p.Mother] {
			seen[p.Mother] = true
			distance[p.Mother] = distance[p] + 1
			queue.PushBack(p.Mother)
		}
		if p.Father != nil && !seen[p.Father] {
			seen[p.Father] = true
			distance[p.Father] = distance[p] + 1
			queue.PushBack(p.Father)
		}

		// Add children.
		for _, c := range p.Children {
			if !seen[c] {
				seen[c] = true
				distance[c] = distance[p] + 1
				queue.PushBack(c)
			}
		}
	}

	return -1
}

type personNode struct {
	p    *Person
	next *personNode
	prev *personNode
}

func NewPersonQueue() *PersonQueue {
	return &PersonQueue{}
}

// PersonQueue is a simple FIFO queue based on a doubly linked list.
type PersonQueue struct {
	head, tail *personNode
	len        int
}

// Len returns the number of elements in the queue.
func (q *PersonQueue) Len() int {
	return q.len
}

// PushBack adds a new element to the back of the queue.
func (q *PersonQueue) PushBack(p *Person) {
	node := &personNode{p: p}
	if q.tail == nil {
		q.head = node
		q.tail = node
	} else {
		q.tail.next = node
		node.prev = q.tail
		q.tail = node
	}
	q.len++
}

// PopFront removes and returns the element at the front of the queue.
func (q *PersonQueue) PopFront() *Person {
	if q.head == nil {
		return nil
	}
	node := q.head
	q.head = node.next
	if q.head == nil {
		q.tail = nil
	} else {
		q.head.prev = nil
	}
	q.len--
	return node.p
}
