package genworldvoronoi

import (
	"fmt"
	"log"
	"math/rand"
	"sort"

	"github.com/Flokey82/genetics"
	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/go_gens/gameconstants"
	"github.com/Flokey82/go_gens/utils"
)

func (m *Civ) tickPeople() {
	// TODO: Run some cleanup of dead people.
	for _, p := range m.People {
		// Check if we are still alive / alive yet.
		if p.Death.IsSet() || !p.Birth.IsSet() || p.Birth.Year > int(m.History.GetYear()) {
			continue
		}
		m.tickPerson(p)
	}

	// TODO: Pair up single people.
	// This should be done per location.
}

func (m *Civ) matchMaker(people []*Person) {
	// Get eligible singles (not dead, not married, and TODO: already alive).
	var single []*Person
	for _, p := range people {
		if p.isEligibleSingle() && !p.Death.IsSet() {
			single = append(single, p)
		}
	}

	// Sort by age, so similar age people are more likely to be paired up quicker.
	sort.Slice(single, func(a, b int) bool {
		return single[a].Age > single[b].Age
	})

	// Pair up singles.
	for i, p := range single {
		if !p.isEligibleSingle() {
			continue // Not single anymore.
		}
		for j, pc := range single {
			if !pc.isEligibleSingle() || p.Region != pc.Region {
				continue // Not single anymore or not in same region.
			}

			// TODO: Allow same sex couples (which can adopt children/orphans).
			if i == j || p.Gender == pc.Gender || isRelated(p, pc) {
				continue
			}

			// At most 33% age difference.
			if utils.Abs(p.Age-pc.Age) > utils.Min(p.Age, pc.Age)/3 {
				continue
			}
			p.Spouse = pc
			pc.Spouse = p

			// Update family name.
			// TODO: This is not optimal... There should be a better way to do this.
			if p.Gender == GenderFemale {
				p.LastName = pc.LastName
			} else {
				pc.LastName = p.LastName
			}
			log.Println(p.String(), "and", pc.String(), "are in love")
			break
		}
	}
}

func (m *Civ) updatePersonLocation(p *Person, r int) {
	// Update location.
	// NOTE: We should differentiate between people who live in the city and
	// people who work in or visit the city.
	p.Region = r
	p.City = m.GetCity(r)
	// TODO: Add person to city population.
}

const (
	ageOfAdulthood     = 16
	ageEndChildbearing = 45
)

func (m *Civ) tickPerson(p *Person) {
	// Calculate age.
	if p.Birth.IsSet() {
		p.Age = int(m.History.GetYear()) - p.Birth.Year
		if m.History.GetDayOfYear() < p.Birth.Day {
			p.Age--
		}
	}

	if p.Gender == GenderFemale && p.Age >= ageOfAdulthood && p.Age < ageEndChildbearing {
		if p.PregnancyCounter > 0 {
			// Person is pregnant.
			m.advancePersonPregnancy(p)
		} else if p.Spouse != nil {
			// There is a slight chance that the person gets pregnant.
			// TODO: This should maybe be triggered by the city birth rate?
			if rand.Intn(365*4) <= 1 {
				m.newPersonPregnancy(p, p.Spouse)
			}
		}
	}

	// Check if person is dead.
	if gameconstants.DiesAtAge(p.Age) {
		// If the person just gave birth, we note that the person
		// died during childbirth.
		p.Death.Year = int(m.History.GetYear())
		p.Death.Day = m.History.GetDayOfYear()

		// Remove as spouse (if any).
		if p.Spouse != nil {
			p.Spouse.Spouse = nil
		}
	}
}

type LifeEvent struct {
	Year   int
	Day    int
	Region int
}

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
	City    *City          // City of the person
	Genes   genetics.Genes // Genes.
	Gender  Gender         // Gender.
	Age     int            // Age of the person.
	Culture *Culture

	// Todo: Allow different naming conventions.
	FirstName string
	LastName  string
	NickName  string

	// Birth, death...
	// TODO: Add death cause.
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

// isElegibleSingle returns true if the person is old enough and single.
func (p *Person) isEligibleSingle() bool {
	return p.Age > ageOfAdulthood && p.Spouse == nil // Old enough and single.
}

// canBePregnant returns true if the person is old enough and not pregnant.
func (p *Person) canBePregnant() bool {
	// Female, has a spouse (implies old enough), and is currently not pregnant.
	// TODO: Set randomized upper age limit.
	return p.Gender == GenderFemale && p.Spouse != nil && p.Prengancy == nil
}

// Name returns the name of the person.
func (p *Person) Name() string {
	var name string
	if p.NickName != "" {
		name = fmt.Sprintf("%s %q", p.FirstName, p.NickName)
	} else {
		name = p.FirstName
	}
	if p.LastName != "" {
		name += " " + p.LastName
	}
	return name
}

// String returns the string representation of the person.
func (p *Person) String() string {
	return fmt.Sprintf("%s \n%s", p.Name(), geneticshuman.String(p.Genes))
}

const pregnancyDays = 280 // for humans

func (m *Civ) newPersonPregnancy(mother, father *Person) *Person {
	var mGenes, fGenes genetics.Genes
	if mother != nil {
		mGenes = mother.Genes
	} else {
		mGenes = genetics.NewRandom()
	}

	if father != nil {
		fGenes = father.Genes
	} else {
		fGenes = genetics.NewRandom()
	}

	gender := randGender()
	genes := genetics.Mix(mGenes, fGenes, 1)

	// We need to set the name after birth, because the parents might not know the gender of the baby
	// until birth. (If there's magic, only wealthy people would be able to determine the gender before)
	p := &Person{
		ID:     m.getNextPersonID(),
		Gender: gender,
		Genes:  fixGenes(gender, genes), // Fix genes wrt. gender (the genetic mix doesn't limit gender varaition)
		Mother: mother,
		Father: father,
		Birth: LifeEvent{
			Region: -1, // Unset until birth.
		},
	}

	mother.PregnancyCounter = pregnancyDays
	mother.Prengancy = p
	return p
}

func (m *Civ) advancePersonPregnancy(p *Person) {
	if p.PregnancyCounter == 0 {
		return
	}

	// Reduce pregnancy counter.
	p.PregnancyCounter--
	if p.PregnancyCounter > 0 {
		return
	}

	// Birth
	child := p.Prengancy
	p.Prengancy = nil
	p.PregnancyCounter = 0

	// Add child to family and name it.
	// TODO: Use naming convention of culture to determine
	// if mother or father name the child.
	if child.Mother != nil {
		child.Mother.Children = append(child.Mother.Children, child)
		child.FirstName = child.Mother.Culture.Language.MakeFirstName()
	} else if child.Father != nil {
		child.FirstName = child.Father.Culture.Language.MakeFirstName()
	}

	if child.Father != nil {
		child.Father.Children = append(child.Father.Children, child)
		child.LastName = child.Father.LastName
	} else if child.Mother != nil {
		child.LastName = child.Mother.LastName
	}

	// Set birth date.
	child.Birth.Year = int(m.History.GetYear())
	child.Birth.Day = m.History.GetDayOfYear()
	child.Birth.Region = p.Region

	// Update location.
	m.updatePersonLocation(child, p.Region)

	// Add child to world.
	m.People = append(m.People, child)
}

func (m *Civ) newRandomPersonAt(r int) *Person {
	// Random gender.
	gender := randGender()
	// Random genes.
	genes := genetics.NewRandom()

	// Get the culture at the region.
	culture := m.GetCulture(r)
	lang := culture.Language

	// Create the person.
	// TODO: Calculate age.
	p := &Person{
		ID:        m.getNextPersonID(),
		Culture:   culture,
		Gender:    gender,
		Genes:     fixGenes(gender, genes),
		FirstName: lang.MakeFirstName(),
		LastName:  lang.MakeLastName(),
		Birth: LifeEvent{
			Year:   int(m.History.GetYear()) - ageOfAdulthood + rand.Intn(2*ageOfAdulthood),
			Day:    rand.Intn(365),
			Region: -1, // TODO: Pick a birth region that makes sense.
		},
	}

	// Update location.
	m.updatePersonLocation(p, r)

	// TODO: Random spouse, children, etc.?
	m.People = append(m.People, p)
	return p
}

func (m *Civ) getNextPersonID() int {
	m.nextPersonID++
	return m.nextPersonID
}

// Gender represents a gender.
type Gender int

const (
	GenderFemale Gender = iota
	GenderMale
)

// randGender returns a random gender.
func randGender() Gender {
	return Gender(rand.Intn(2))
}

// String returns the string representation of the gender.
func (g Gender) String() string {
	switch g {
	case GenderFemale:
		return "F"
	case GenderMale:
		return "M"
	default:
		return "X"
	}
}

// fixGenes makes sure that the gender is set properly in the genes.
// NOTE: This needs to be done due to the genetics package being a bit weird.
func fixGenes(gender Gender, genes genetics.Genes) genetics.Genes {
	switch gender {
	case GenderFemale:
		geneticshuman.SetGender(&genes, geneticshuman.GenderFemale)
	case GenderMale:
		geneticshuman.SetGender(&genes, geneticshuman.GenderMale)
	default:
		geneticshuman.SetGender(&genes, 0)
	}
	return genes
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
