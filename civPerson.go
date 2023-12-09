package genworldvoronoi

import (
	"fmt"
	"math/rand"

	"github.com/Flokey82/genetics"
	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/go_gens/gameconstants"
	"github.com/Flokey82/go_gens/genlanguage"
)

func (m *Civ) getNextPersonID() int {
	m.nextPersonID++
	return m.nextPersonID
}

// tickPerson advances the person by nDays and returns any new born child.å
// TODO: Twins, triplets, etc.
func (m *Civ) tickPerson(p *Person, nDays int, cf func(int) *Culture) *Person {
	if !m.doesPersonExist(p) {
		return nil
	}
	// Calculate age.
	m.tickPersonAge(p, nDays)

	// Advance pregnancy.
	child := m.tickPersonPregnancy(p, nDays, cf)

	// Check if person dies of natural causes.
	m.tickPersonDeath(p, nDays)

	return child
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

// tickPersonPregnancy advances the pregnancy of a person.
// If the pregnancy is successful, a new person is born.
func (m *Civ) tickPersonPregnancy(p *Person, nDays int, cf func(int) *Culture) *Person {
	// Check if the person is pregnant.
	if p.Prengancy != nil {
		return m.advancePersonPregnancy(p, nDays, cf)
	}

	// Check if the person can get pregnant.
	if p.isOfChildbearingAge() && p.canBePregnant() {
		// Approximately once every 5 years if no children.
		// TODO: Figure out proper chance of birth.
		chance := 5 * 365
		if p.Age > 40 {
			// Over 40, it becomes more and more unlikely.
			// TODO: Genetic variance?
			chance *= (p.Age - 40)
		}

		// The more children, the less likely it becomes
		// that more children are on the way.
		//
		// NOTE: Not because of biological reasons, but
		// who wants more children after having some.
		chance *= len(p.Children) + 1
		if rand.Intn(chance) < nDays {
			p.newPersonPregnancy(m.getNextPersonID(), p.Spouse)
		}
	}
	return nil
}

func (m *Civ) tickPersonDeath(p *Person, nDays int) {
	// Check if person dies of natural causes.
	if gameconstants.DiesAtAgeWithinNDays(p.Age, nDays) {
		// If the person just gave birth, we note that the person
		// died during childbirth.
		m.killPerson(p, "") // Random cause?
	}
}

func (m *Civ) killPerson(p *Person, reason string) {
	p.Death.Day = int(m.History.GetDayOfYear())
	p.Death.Year = int(m.History.GetYear())
	p.Death.Region = p.Region

	// If they have a spouse, unset their spouse.
	if p.Spouse != nil {
		p.Spouse.Spouse = nil
	}

	// :(
	if p.Prengancy != nil {
		m.killPerson(p.Prengancy, reason)
	}

	var deathStr string
	if reason == "" {
		deathStr = fmt.Sprintf("%s died", p.Name())
	} else {
		deathStr = fmt.Sprintf("%s died due to %s", p.Name(), reason)
	}
	m.AddEvent("Death", deathStr, p.Ref())

	/*
		var str string
		if p.Spouse != nil {
			str += "[spouse]"
		}
		if len(p.Children) > 0 {
			str += fmt.Sprintf("[%d children]", len(p.Children))
		}
		log.Println("killed", p.Name(), "at", p.Death.Region, "aged", p.Age, str)
	*/
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
	ID      int            // ID of the person
	Region  int            // Location of the person
	Genes   genetics.Genes // Genes.
	City    *City          // City of the person
	Culture *Culture       // Culture of the person

	// Todo: Allow different naming conventions.
	FirstName string
	LastName  string
	NickName  string

	// Birth, death...
	// TODO: Add death cause.
	Age   int // Age of the person.
	Birth LifeEvent
	Death LifeEvent

	// Pregnancy
	PregnancyCounter int     // Days of pregnancy
	Prengancy        *Person // baby (TODO: twins, triplets, etc.)

	// Family (TODO: Distinguish between known and unknown family members.)
	// Maybe use a map of relations to people?
	Mother   *Person
	Father   *Person
	Spouse   *Person   // TODO: keep track of spouses that might have perished?
	Children []*Person // TODO: Split into known and unknown children.
}

func (m *Civ) newRandomPersonAt(r int, culture *Culture) *Person {
	// Random genes / gender.
	genes := genetics.NewRandom()
	geneticshuman.SetGender(&genes, randGender())

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
	if poolSize := lang.GetLastNamePoolSize(); poolSize > 300 && rand.Intn(poolSize) > 10 {
		lastName = lang.GetLastName()
	}
	if lastName == "" {
		lastName = lang.MakeLastName()
	}

	// TODO: Calculate age.
	p := &Person{
		ID:        m.getNextPersonID(),
		Culture:   culture,
		Genes:     genes,
		FirstName: firstName,
		LastName:  lastName,
		Birth: LifeEvent{
			Year:   int(m.History.GetYear()) - ageOfAdulthood + rand.Intn(2*ageOfAdulthood),
			Day:    rand.Intn(365),
			Region: r, // TODO: Pick a birth region that makes sense.
		},
	}

	// Update location.
	m.updatePersonLocation(p, r)

	// TODO: Random spouse, children, etc.?
	m.People = append(m.People, p)
	return p
}

// Name returns the name of the person.
func (p *Person) Name() string {
	if p.NickName != "" {
		return fmt.Sprintf("%s %q %s", p.FirstName, p.NickName, p.LastName)
	}
	return p.FirstName + " " + p.LastName
}

// Ref returns the object reference of the person.
func (p *Person) Ref() ObjectReference {
	return ObjectReference{
		ID:   p.ID,
		Type: ObjectTypePerson,
	}
}

// String returns the string representation of the person.
func (p *Person) String() string {
	return fmt.Sprintf("%s \n%s", p.Name(), geneticshuman.String(p.Genes))
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

	// We need to set the name after birth, because the parents might not know the gender of the baby
	// until birth. (If there's magic, only wealthy people would be able to determine the gender before)
	child := &Person{
		ID:     id,
		Genes:  genes,
		Mother: p,
		Father: father,
	}

	p.PregnancyCounter = pregnancyDays
	p.Prengancy = child
	return child
}

// advancePersonPregnancy advances the pregnancy of the person.
// TODO: Add twins, triplets, etc.
func (m *Civ) advancePersonPregnancy(p *Person, nDays int, cf func(int) *Culture) *Person {
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
