package civ2

import (
	"fmt"
	"math"
	"math/rand"

	"github.com/Flokey82/genetics"
	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/genworldvoronoi/civ"
)

type Gender = geneticshuman.Gender

const (
	GenderMale   = geneticshuman.GenderMale
	GenderFemale = geneticshuman.GenderFemale
)

// randGender returns a random gender.
func randGender() Gender {
	if rand.Intn(2) == 0 {
		return GenderFemale
	}
	return GenderMale
}

type ClampedVal float64

func (c *ClampedVal) Add(v float64) {
	*c = ClampedVal(math.Max(0, math.Min(1, float64(*c)+v)))
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
type Person struct {
	ID          int                      // ID of the person
	Region      int                      // Location of the person
	Genes       genetics.Genes           // Genes.
	Personality geneticshuman.FiveFactor // Personality of the person
	Traits      geneticshuman.Trait      // Traits of the person
	City        *City                    // City of the person
	Culture     *Culture                 // Culture of the person
	Opinions    *Opinions                // Opinions of the person
	Conditions  *Conditions              // Conditions of the person
	Popularity  ClampedVal               // Popularity of the person (reputation, 0.0-1.0)
	Karma       ClampedVal               // Karma of the person (good/bad deeds, 0.0-1.0)

	// Todo: Allow different naming conventions.
	FirstName string
	LastName  string
	NickName  string
	Title     string // Title of the person.

	// Birth, death...
	Age   int // Age of the person.
	Birth LifeEvent
	Death LifeEvent

	// Pregnancy
	PregnancyCounter int     // Days of pregnancy
	Prengancy        *Person // baby

	// Gold.
	Gold float64

	// Family
	Mother   *Person
	Father   *Person
	Spouse   *Person
	Children []*Person

	// Artifacts
	Artifacts []*Artifact
}

func (p *Person) GetID() int {
	return p.ID
}

// Ref returns the object reference of the person.
func (p *Person) Ref() civ.ObjectReference {
	return civ.ObjectReference{
		ID:   p.ID,
		Type: civ.ObjectTypePerson,
	}
}

// String returns a string representation of the person.
func (p *Person) String() string {
	name := p.Name()
	if p.Title != "" {
		name = p.Title + " " + name
	}
	gender := "°"
	if p.Gender() == GenderFemale {
		gender = "♀"
	} else if p.Gender() == GenderMale {
		gender = "♂"
	}
	isDead := " "
	if p.Dead() {
		isDead = "†"
	}
	str := fmt.Sprintf("%s (%s%s%d) (P:%.1f, K:%.1f, G:%.1f, A:%d)", name, gender, isDead, p.Age, p.Popularity, p.Karma, p.Gold, len(p.Artifacts))
	if p.Traits != 0 {
		str += " [" + p.Traits.String() + "]"
	}
	return fmt.Sprintf("(%d) %s", p.ID, str)
}

func (p *Person) addArtifact(m *Civ, a *Artifact) {
	p.Artifacts = append(p.Artifacts, a)
	// Apply any condition (blessing, curse, etc.) from the artifact.
	if a.Condition != nil {
		if p.Conditions == nil {
			p.Conditions = NewConditions()
		}
		p.Conditions.Add(a.Condition, m, p)
	}
}

func (p *Person) removeArtifact(m *Civ, a *Artifact) {
	for i, art := range p.Artifacts {
		if art == a {
			p.Artifacts = append(p.Artifacts[:i], p.Artifacts[i+1:]...)
			// Remove any condition (blessing, curse, etc.) from the artifact.
			if a.Condition != nil && p.Conditions != nil {
				p.Conditions.Remove(a.Condition, true, m, p)
			}
			break
		}
	}
}

func (p *Person) transferArtifact(m *Civ, a *Artifact, to *Person) {
	p.removeArtifact(m, a)
	to.addArtifact(m, a)
}

func (p *Person) compare(other *Person) float64 {
	if p == nil || other == nil {
		return -1.0
	}
	if p == other {
		return 1.0
	}
	// TODO: Weight by genetics and social opinions
	return 0.0
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

// Dead returns true if the person is dead.
func (p *Person) Dead() bool {
	return p.Death.IsSet()
}

// Gender returns the gender of the person.
func (p *Person) Gender() Gender {
	return geneticshuman.GetGender(&p.Genes)
}

const (
	ageOfAdulthood     = 18
	ageEndChildbearing = 45
	pregnancyDays      = 280 // for humans
)

func (m *Civ) newRandomPersonAt(r int, culture *Culture, gender Gender, parent *Person) *Person {
	// Random genes / gender.
	var genes genetics.Genes
	if parent != nil {
		genes = genetics.Mix(parent.Genes, genetics.NewRandom(), 1)
	} else {
		genes = genetics.NewRandom()
	}
	geneticshuman.SetGender(&genes, gender)

	lang := culture.Language

	// If the first/last name pool is large enough, we should
	// start reusing names because generating new names is expensive.
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
		Region:      r,
		Birth: LifeEvent{
			Year:   int(m.Geo.Calendar.GetYear()) - age,
			Day:    rand.Intn(365),
			Region: r,
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

	m.People = append(m.People, p)
	return p
}

func (p *Person) isOfChildbearingAge() bool {
	return p.Age >= ageOfAdulthood && p.Age < ageEndChildbearing
}

func (p *Person) isEligibleSingle() bool {
	return p.Age > ageOfAdulthood && p.Spouse == nil
}

func (p *Person) canBePregnant() bool {
	return p.Gender() == GenderFemale && p.Spouse != nil && p.Prengancy == nil
}

func (p *Person) newPersonPregnancy(id int, father *Person) *Person {
	var genes genetics.Genes
	if father != nil {
		genes = genetics.Mix(p.Genes, father.Genes, 1)
	} else {
		genes = genetics.Mix(p.Genes, genetics.NewRandom(), 1)
	}
	geneticshuman.SetGender(&genes, randGender())

	fiveFactor := geneticshuman.GetFiveFactor(&genes)
	traits := geneticshuman.GetTraits(fiveFactor)

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

func isRelated(a, b *Person) bool {
	if a == b.Father || a == b.Mother || b == a.Father || b == a.Mother {
		return true
	}
	if (a.Father == nil && a.Mother == nil) || (b.Father == nil && b.Mother == nil) {
		return false
	}
	return a.Mother == b.Mother || a.Father == b.Father
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
