package civ2

import (
	"fmt"
	"math/rand"
	"sort"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/genworldvoronoi/civ"
	"github.com/Flokey82/go_gens/gengovernment"
	"github.com/Flokey82/go_gens/genlanguage"
)

type FactionType int

const (
	FactionTypeReligious FactionType = iota
	FactionTypeMartial
	FactionTypeCivil
	FactionTypeCriminal
	FactionTypeMerchant
	FactionTypeMax
)

// String returns the string representation of the faction type.
func (f FactionType) String() string {
	switch f {
	case FactionTypeReligious:
		return "Religious"
	case FactionTypeMartial:
		return "Martial"
	case FactionTypeCivil:
		return "Civil"
	case FactionTypeCriminal:
		return "Criminal"
	case FactionTypeMerchant:
		return "Merchant"
	}
	return "Unknown"
}

type Leadership struct {
	*gengovernment.Leadership
	Leader *Person
}

func newLeadership(f gengovernment.LeadershipForm) *Leadership {
	return &Leadership{
		Leadership: f.Generate(),
	}
}

func (l *Leadership) compare(other *Leadership) float64 {
	if l == other {
		return 1.0
	}
	if l == nil || other == nil {
		return -1.0
	}

	compatibility := 0.0
	if l.Form == other.Form {
		compatibility += 1.0
	} else {
		compatibility -= 1.0
	}

	if l.Succession == other.Succession {
		compatibility += 1.0
	} else {
		compatibility -= 1.0
	}

	if l.GenderRestriction == other.GenderRestriction {
		compatibility += 1.0
	} else {
		compatibility -= 1.0
	}

	if l.Leader != nil && other.Leader != nil {
		compatibility += l.Leader.compare(other.Leader)
		
		opA := l.Leader.Opinions.GetOpinion(other.Leader)
		opB := other.Leader.Opinions.GetOpinion(l.Leader)
		compatibility += (opA + opB) / 2
	}

	return compatibility / 5
}

func (l *Leadership) LeaderPreferredForm() gengovernment.LeadershipForm {
	if l.Leader == nil {
		return l.Form
	}
	traits := l.Leader.Traits
	if traits.HasTrait(geneticshuman.TraitAmbitious) {
		if traits.HasTrait(geneticshuman.TraitCruel) {
			return gengovernment.LeadershipFormDictatorship
		}
		if traits.HasTrait(geneticshuman.TraitKind) {
			return gengovernment.LeadershipFormDemocracy
		}
		return gengovernment.LeadershipFormMonarchy
	}
	if traits.HasTrait(geneticshuman.TraitHonest) || traits.HasTrait(geneticshuman.TraitTrusting) {
		return gengovernment.LeadershipFormDemocracy
	}
	if traits.HasTrait(geneticshuman.TraitAggressive) || traits.HasTrait(geneticshuman.TraitCruel) || traits.HasTrait(geneticshuman.TraitBrave) {
		return gengovernment.LeadershipFormDictatorship
	}
	if traits.HasTrait(geneticshuman.TraitCalm) || traits.HasTrait(geneticshuman.TraitCowardly) || traits.HasTrait(geneticshuman.TraitContent) {
		return gengovernment.LeadershipFormCouncil
	}
	return l.Form
}

func (l *Leadership) GetTitleForPerson(p *Person) string {
	if g := p.Gender(); g == GenderMale {
		return l.GetTitle(gengovernment.GenderMale)
	} else if g == GenderFemale {
		return l.GetTitle(gengovernment.GenderFemale)
	}
	return ""
}

func (l *Leadership) String() string {
	if l.Leader == nil {
		return "None"
	}
	title := l.GetTitleForPerson(l.Leader)
	return fmt.Sprintf("%s %s (%s)", title, l.Leader.Name(), l.Form.String())
}

func (l *Leadership) ChangeTitle(h *civ.History) {
	if l.Leader != nil {
		oldTitle := l.GetTitleForPerson(l.Leader)
		l.Title = l.Form.Title(l.LeaderPreferredTags()...)
		newTitle := l.GetTitleForPerson(l.Leader)

		l.Leader.Title = newTitle

		h.AddEvent(HistoryEventLeadership, fmt.Sprintf("%s changed title from %s to %s.", l.Leader.Name(), oldTitle, newTitle), l.Leader.Ref())
	} else {
		l.Title = l.Form.Title(l.LeaderPreferredTags()...)
	}
}

func (l *Leadership) LeaderPreferredTags() []gengovernment.TitleTag {
	var preferred []gengovernment.TitleTag
	if l.Leader != nil {
		if l.Leader.Traits.HasTrait(geneticshuman.TraitAmbitious) {
			preferred = append(preferred, gengovernment.TitleTagPower, gengovernment.TitleTagGrandeur)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitCruel) {
			preferred = append(preferred, gengovernment.TitleTagCruelty, gengovernment.TitleTagPower)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitKind) {
			preferred = append(preferred, gengovernment.TitleTagWisdom, gengovernment.TitleTagService)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitAggressive) {
			preferred = append(preferred, gengovernment.TitleTagMilitary, gengovernment.TitleTagPower)
		}
	}
	return preferred
}

func (c *City) getPreferredLeadershipForm() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormMonarchy
}

func (c *City) getPossibleLeadershipForms() []gengovernment.LeadershipForm {
	return []gengovernment.LeadershipForm{gengovernment.LeadershipFormMonarchy, gengovernment.LeadershipFormDemocracy, gengovernment.LeadershipFormCouncil}
}

func (c *City) findNaturalProgression() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormMonarchy
}

func (c *City) findCoupProgression() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormDictatorship
}

func (c *City) getLanguage() *genlanguage.Language {
	return c.Culture.Language
}

func (m *Civ) getFactionActions(f *Faction) []*FactionAction {
	return []*FactionAction{{
		probability: func(fac *Faction) float64 { return 0.05 },
		requires: func(fac *Faction) bool {
			return fac.Popularity < 0.5
		},
		consequences: func(e peopleThing, m *Civ, fac *Faction) {
			if fac.Leader == nil {
				return
			}
			traits := fac.Leader.Traits
			switch {
			case traits.HasTrait(geneticshuman.TraitKind):
				// Charity or public event.
				fac.Popularity.Add(0.2)
				m.History.AddEvent(HistoryEventFaction, fmt.Sprintf("%s held a public event to increase popularity.", fac.Name), fac.Ref())
			case traits.HasTrait(geneticshuman.TraitCruel):
				// Intimidation.
				fac.Popularity.Add(0.1)
				m.History.AddEvent(HistoryEventFaction, fmt.Sprintf("%s performed a display of power to intimidate the populace.", fac.Name), fac.Ref())
			case traits.HasTrait(geneticshuman.TraitDeceptive):
				// Manipulation.
				fac.Popularity.Add(0.15)
				m.History.AddEvent(HistoryEventFaction, fmt.Sprintf("%s spread rumors to shift public opinion.", fac.Name), fac.Ref())
			}
		},
	}}
}

func (c *City) getFactionActions(m *Civ, f *Faction) []*FactionAction {
	return m.getFactionActions(f)
}

func (c *City) String() string {
	return fmt.Sprintf("City %s (%d)", c.Name, c.ID)
}

func (s *Settlement) getPreferredLeadershipForm() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormChiefdom
}

func (s *Settlement) getPossibleLeadershipForms() []gengovernment.LeadershipForm {
	return []gengovernment.LeadershipForm{gengovernment.LeadershipFormChiefdom}
}

func (s *Settlement) findNaturalProgression() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormChiefdom
}

func (s *Settlement) findCoupProgression() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormDictatorship
}

func (s *Settlement) getLanguage() *genlanguage.Language {
	return s.Culture.Language
}

func (s *Settlement) getFactionActions(m *Civ, f *Faction) []*FactionAction {
	return m.getFactionActions(f)
}

func (s *Settlement) String() string {
	return fmt.Sprintf("Settlement %s (%d)", s.Name, s.ID)
}

func (t *Tribe) getPreferredLeadershipForm() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormChiefdom
}

func (t *Tribe) getPossibleLeadershipForms() []gengovernment.LeadershipForm {
	return []gengovernment.LeadershipForm{gengovernment.LeadershipFormChiefdom}
}

func (t *Tribe) findNaturalProgression() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormChiefdom
}

func (t *Tribe) findCoupProgression() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormDictatorship
}

func (t *Tribe) getLanguage() *genlanguage.Language {
	return t.Culture.Language
}

func (t *Tribe) getFactionActions(m *Civ, f *Faction) []*FactionAction {
	return m.getFactionActions(f)
}

func (t *Tribe) String() string {
	return t.Name()
}

func (cs *CityState) getPreferredLeadershipForm() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormCouncil
}

func (cs *CityState) getPossibleLeadershipForms() []gengovernment.LeadershipForm {
	return []gengovernment.LeadershipForm{gengovernment.LeadershipFormCouncil, gengovernment.LeadershipFormDemocracy}
}

func (cs *CityState) findNaturalProgression() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormCouncil
}

func (cs *CityState) findCoupProgression() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormDictatorship
}

func (cs *CityState) getLanguage() *genlanguage.Language {
	return cs.Culture.Language
}

func (cs *CityState) getFactionActions(m *Civ, f *Faction) []*FactionAction {
	return m.getFactionActions(f)
}

func (e *Empire) getPreferredLeadershipForm() gengovernment.LeadershipForm {
	if e.Capital.Population < 1000 {
		return gengovernment.LeadershipFormMonarchy
	}
	return gengovernment.LeadershipFormDictatorship
}

func (e *Empire) getPossibleLeadershipForms() []gengovernment.LeadershipForm {
	return []gengovernment.LeadershipForm{
		gengovernment.LeadershipFormMonarchy,
		gengovernment.LeadershipFormDictatorship,
		gengovernment.LeadershipFormCouncil,
		gengovernment.LeadershipFormDemocracy,
	}
}

func (e *Empire) findNaturalProgression() gengovernment.LeadershipForm {
	if e.Capital.Population < 1000 {
		return gengovernment.LeadershipFormMonarchy
	}
	return gengovernment.LeadershipFormDictatorship
}

func (e *Empire) findCoupProgression() gengovernment.LeadershipForm {
	return gengovernment.LeadershipFormDictatorship
}

func (e *Empire) getLanguage() *genlanguage.Language {
	return e.Culture.Language
}

func (e *Empire) getFactionActions(m *Civ, f *Faction) []*FactionAction {
	return m.getFactionActions(f)
}


func (l *Leadership) ChooseNewLeader(t peopleThing, f *Faction, m *Civ) *Person {
	var candidate *Person
	pickGender := func() geneticshuman.Gender {
		if l.GenderRestriction == gengovernment.GenderRestrictionMale {
			return geneticshuman.GenderMale
		}
		if l.GenderRestriction == gengovernment.GenderRestrictionFemale {
			return geneticshuman.GenderFemale
		}
		return randGender()
	}
	if l.Leader != nil && (l.Succession == gengovernment.SuccessionTypeInherited || rand.Intn(100) < 50) && l.Leader.Age > ageOfAdulthood {
		children := l.Leader.Children
		sort.Slice(children, func(i, j int) bool {
			return children[i].Age > children[j].Age
		})
		for _, c := range children {
			if c.Title != "" || c.Dead() {
				continue
			}
			if l.GenderRestriction != gengovernment.GenderRestrictionNone {
				g := c.Gender()
				if l.GenderRestriction == gengovernment.GenderRestrictionMale && g != geneticshuman.GenderMale ||
					l.GenderRestriction == gengovernment.GenderRestrictionFemale && g != geneticshuman.GenderFemale {
					continue
				}
			}
			candidate = c
			break
		}
		if candidate == nil {
			candidate = t.NewRandomChild(m, pickGender(), l.Leader)
		}
	} else {
		gender := pickGender()
		people := t.GetPeople()
		sort.Slice(people, func(i, j int) bool {
			return people[i].Popularity > people[j].Popularity
		})
		for _, p := range people {
			if p == l.Leader || p.Dead() || p.Age < ageOfAdulthood || p.Gender() != gender || p.Title != "" {
				continue
			}
			candidate = p
			break
		}
		if candidate == nil {
			candidate = t.NewRandomPerson(m, gender)
		}
	}
	m.History.AddEvent(HistoryEventLeadership, fmt.Sprintf("%s was chosen as the new leader of %s.", candidate.Name(), f.Name), f.Ref())
	return l.ReplaceLeader(candidate, t, f, m.History)
}

func (l *Leadership) ReplaceLeader(newLeader *Person, t peopleThing, f *Faction, h *civ.History) *Person {
	oldLeader := l.Leader
	if oldLeader != nil {
		oldLeader.Title += " (former)"
	}
	newLeader.Title = l.GetTitleForPerson(newLeader)
	l.Leader = newLeader

	if oldLeader != nil {
		h.AddEvent(HistoryEventLeadership, fmt.Sprintf("%s replaced %s as the leader of %s.", newLeader.Name(), oldLeader.Name(), f.Name), t.Ref())
	}
	return oldLeader
}

// NumLeaders returns the number of leaders in the leadership.
func (l *Leadership) NumLeaders() int {
	if l.Leader == nil {
		return 0
	}
	return 1
}

// Leaders returns the leaders in the leadership.
func (l *Leadership) Leaders() []*Person {
	if l.Leader == nil {
		return nil
	}
	return []*Person{l.Leader}
}

type Faction struct {
	ID         int
	Name       string
	Type       FactionType
	lang       *genlanguage.Language
	Popularity ClampedVal
	*Leadership
}

func (f *Faction) ChangeType(newType FactionType, leadForm gengovernment.LeadershipForm, h *civ.History) FactionType {
	oldType := f.Type
	if oldType != newType {
		f.Type = newType
		h.AddEvent(HistoryEventFaction, fmt.Sprintf("Faction %s changed type from %s to %s.", f.Name, oldType.String(), newType.String()), f.Ref())
	}
	oldForm := f.Leadership.Form
	if oldForm != leadForm {
		f.Leadership.ChangeForm(leadForm)
		h.AddEvent(HistoryEventFaction, fmt.Sprintf("Faction %s changed leadership form from %s to %s.", f.Name, oldForm.String(), leadForm.String()), f.Ref())
	}
	return oldType
}

func (f *Faction) ExecutePerson(p *Person, reason string, m *Civ) {
	m.killPerson(p, reason)
	m.History.AddEvent(HistoryEventFaction, fmt.Sprintf("%s %s of faction %s was executed for %s.", f.GetTitleForPerson(p), p.Name(), f.Name, reason), p.Ref())
}

func (m *Civ) genFaction(t peopleThing, f gengovernment.LeadershipForm, leader *Person) *Faction {
	lang := t.getLanguage()
	var name string
	if rand.Float64() < 0.5 {
		name = "The "
	}
	prefixes := []string{"Brothers", "Sisters", "Warriors", "Mystics", "Hunters", "Fishers", "Farmers", "Craftsmen", "Artisans"}
	name += prefixes[rand.Intn(len(prefixes))]

	if rand.Float64() < 0.5 {
		name += " of "
		if rand.Float64() < 0.5 {
			suffixes := []string{"the Sun", "the Moon", "the Stars", "the Earth", "the Sea", "the Mountains", "the Forest", "the River", "the Lake", "the Ocean"}
			name += suffixes[rand.Intn(len(suffixes))]
		} else {
			name += lang.MakeName()
		}
	}
	fac := &Faction{
		ID:         m.getNextFactionID(),
		Name:       name,
		Leadership: newLeadership(f),
		Popularity: 1.0,
		lang:       lang,
		Type:       FactionType(rand.Intn(int(FactionTypeMax))),
	}
	if leader == nil {
		fac.ChooseNewLeader(t, fac, m)
	} else {
		fac.ReplaceLeader(leader, t, fac, m.History)
	}
	fac.Leadership.ChangeTitle(m.History)

	return fac
}

func (f *Faction) Ref() civ.ObjectReference {
	return civ.ObjectReference{ID: f.ID, Type: civ.ObjectTypeFaction}
}

func (f *Faction) String() string {
	return fmt.Sprintf("%s (%s - %.2f)", f.Name, f.Leadership.String(), f.Popularity)
}

type FactionAction struct {
	probability  func(fac *Faction) float64
	requires     func(fac *Faction) bool
	consequences func(e peopleThing, m *Civ, fac *Faction)
}

func (f *FactionAction) Execute(e peopleThing, m *Civ, fac *Faction) {
	if f.consequences != nil {
		f.consequences(e, m, fac)
	}
}

type GoverningPeople struct {
	Leadership      *Faction
	Factions        []*Faction
	Satisfaction    ClampedVal
	SatisfactionAvg *RunningAverageLimit
	Gold            int
}

func newGoverningPeople() *GoverningPeople {
	return &GoverningPeople{
		Satisfaction:    1.0,
		SatisfactionAvg: NewRunningAverageLimit(100),
	}
}
