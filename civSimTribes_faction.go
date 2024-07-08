package genworldvoronoi

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"

	"github.com/Flokey82/genetics/geneticshuman"
	"github.com/Flokey82/go_gens/gengovernment"
	"github.com/Flokey82/go_gens/genlanguage"
)

var leaderID int

func getLeaderID() int {
	leaderID++
	return leaderID
}

var factionID int

func getFactionID() int {
	factionID++
	return factionID
}

func (s *simState) handleLeadership(t *Tribe) {
	m := s.m

	// Update the factions.
	// This will trigger election of new leaders, etc.
	t.Leadership.Tick(t, m)
	for _, f := range t.Factions {
		f.Tick(t, m)
	}

	// Update existing factions.
	// - If a faction has a popularity of 0, it should be removed.
	// - Either exile, kill, or merge with another faction.
	// Factions should have independent actions, etc.
	// - Factions with high popularity unlock new actions, etc.
	// - Available action depends on sponsorship, popularity, etc.
	// - We will need to keep track of notable sponsors and members.
	// Example actions:
	// - Charitable actions (help the poor, etc.)
	// - Userper actions (try to take over the tribe)
	// - Religious actions (try to convert the tribe to a new religion)
	// - Intrigue actions (try to kill the leadership, etc.)

	// The lower the satisfaction, the higher the chance that a faction will form.
	if t.Satisfaction < 0.75 && rand.Float64() > float64(t.Satisfaction) {
		log.Printf("Tribe %d is unhappy. A new faction might form.", t.ID)
		if len(t.Factions) == 0 || rand.Float64() < 0.1/float64(len(t.Factions)) {
			// TODO: Pick a LeadershipForm opposed to the current leadership.
			// - If the leadership is a monarchy, the faction should be a republic, etc.
			// - If the leadership is a council, the faction should be a dictatorship, etc.
			form := t.findCoupProgression()

			// Pick a person that suits the form.
			// We would pick a person that is ambitious, etc.
			// Maybe just the polar opposite of the current leadership?
			origLeader := t.Leadership.Leader // TODO: Check if this is nil.
			mostDifferent := 0.0
			var leader *Person
			for _, p := range t.People {
				if p.Dead() || p.Age < ageOfAdulthood || p.Title != "" {
					continue
				}
				if p == origLeader {
					continue
				}
				diff := p.compare(origLeader)
				if diff > mostDifferent {
					mostDifferent = diff
					leader = p
				}
				// TODO: Also take opinion and reputation into account.
			}

			f := genFaction(t, m, form, leader)
			t.Factions = append(t.Factions, f)

			// Add a history entry.
			historyMsg := fmt.Sprintf("Tribe %d (at %.2f) has formed a new faction %s.", t.ID, float64(t.Satisfaction), f.String())
			m.History.AddEvent("Founding (Faction)", historyMsg, t.Ref())
		} else if len(t.Factions) > 0 && rand.Float64() > float64(t.Leadership.Popularity) {
			// There is a chance that a faction, more popular than the leadership, will try to take over.
			// Sort the factions by popularity.
			sort.Slice(t.Factions, func(i, j int) bool {
				return t.Factions[i].Popularity > t.Factions[j].Popularity
			})

			if topFaction := t.Factions[0]; topFaction.Popularity > t.Leadership.Popularity {
				executeTakeover(t, topFaction, s.m)
			}
		}
	}
}

func executeTakeover(t *Tribe, topFaction *Faction, m *Civ) {
	// The top faction will take over.
	// Depending on chance and popularity, it might kill the leadership, or simply replace it.
	//
	// TODO:
	// - Take note of this event.
	// - Change opinion of factions, etc.
	if oldLeadership := t.Leadership; rand.Float64() > float64(oldLeadership.Popularity) {
		executeCoup(t, topFaction, m)
	} else {
		executeTransition(t, topFaction, m)
	}
}

func executeCoup(t *Tribe, newLeadership *Faction, m *Civ) {
	oldLeadership := t.Leadership

	// Get the new type based on a coup.
	newForm := t.findCoupProgression()

	// The more unpopular the leadership, the higher the chance that leadership will be killed.
	if t.Population > oldLeadership.NumLeaders()+newLeadership.NumLeaders() {
		t.Population -= oldLeadership.NumLeaders()
	} else {
		log.Printf("%s the population is less than the number of leaders!", t.String())
	}
	t.Leadership = newLeadership
	t.Leadership.ChangeType(FactionTypeCivil, newForm, m.History)

	// Remove new leadership from the list of secondary factions.
	newFactions := make([]*Faction, 0, len(t.Factions)-1)
	for _, fac := range t.Factions {
		if fac == newLeadership {
			continue
		}
		newFactions = append(newFactions, fac)
	}
	t.Factions = newFactions

	// Add a history entry.
	historyMsg := fmt.Sprintf("Tribe %d's unpopular leadership (%s at %.2f) was lynched by faction %s at %.2f", t.ID, oldLeadership.Name, oldLeadership.Popularity, newLeadership.Name, newLeadership.Popularity)
	m.History.AddEvent("Uprising (Faction)", historyMsg, t.Ref())
}

func executeTransition(t *Tribe, newLeadership *Faction, m *Civ) {
	oldLeadership := t.Leadership
	// Get the new type based on a transition.
	newForm := t.findNaturalProgression()

	// The faction will simply replace the current leadership
	// and we swap the spot in the secondary factions.
	for i, tf := range t.Factions {
		if tf == newLeadership {
			t.Factions[i] = oldLeadership
			break
		}
	}
	t.Leadership = newLeadership

	newLeadership.ChangeType(FactionTypeCivil, newForm, m.History)
	oldLeadership.ChangeType(FactionTypeCivil, gengovernment.LeadershipFormChiefdom, m.History) // TODO: If this was a dictatorship, autocratic... milder version of ruling class

	// Add a history entry.
	historyMsg := fmt.Sprintf("Tribe %d's leadership (%s at %.2f) was replaced by faction %s at %.2f", t.ID, oldLeadership.Name, oldLeadership.Popularity, newLeadership.Name, newLeadership.Popularity)
	m.History.AddEvent("Election (Faction)", historyMsg, t.Ref())
}

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

// TODO: Leadership should be an interface.
// MakeDecision(action) should be abstract and how the leadership decides depends on what form it has
// - There could be a normal monarchy where only the king decides
// - There could be a sort of council, where a vote is taken
// - There could be a democracy, where the people vote
// The structure or hierarchy of the leadership can be anything from a single leader, to a council, to a democracy.
// The form of leadership comes with special traits and actions.
// There is also nuance in the actions depending on the culture.
// - Would it be accceptable to kill a leader to take over?
// - How would the people react to a coup?
//
// There should be the possibility to change the leadership type.
// ... a chiefdom could become a democracy, or a democracy could become a dictatorship.
//
// Every action might result in a reaction by the people.
// If an unpopular action is taken, the popularity of the leadership will decrease.
// If an action is extremely unpopular, the people might revolt.
// - There is a chance that the revolt will succeed or fail.
// - If the revolt succeeds, the leadership will be replaced.
// - If the revolt fails, the leadership will become more oppressive.
// - Success depends on the popularity of the leadership, if there is one or multiple factions involved,
//   and their popularity, etc. A critical mass of participants is needed for a successful revolt.
// - There are different types of revolts, from peaceful protests to violent uprisings.
// - Depending on the type of leadership, the type of revolt will have different chances of success.
// - A democracy might be more likely to succeed in a peaceful protest, while a dictatorship might be more likely to succeed in a violent uprising.
//
// Succession also depends on the type of leadership.
// - In a monarchy, the oldest child might inherit the throne (or other rules, like the youngest girl. etc.)
// - In a council with a lead, the council might elect a new leader from the council members.
// - In a council without a lead, the council might elect a new leader from the people.
// - In a democracy, the people might elect a new leader.
// - In a dictatorship, the dictator might appoint a successor, or if he doesn't,
//   the factions might fight for control. If it is an empire, it might simply fall apart.
// - In a chiefdom, the chief might appoint someone, it might be inherited,
//   a council might vote, and/or one might be chosen by merit, or the people might elect a new chief.
//   Like the bravest warrior, the best hunter, the most skilled craftsman, most successful trader, etc.
//
// There are democracies that still have a king or queen, so there should be the possibility to define
// the voting rights of each component of the leadership (e.g. the king, the council, the people).
// - Individuals have popularity, and decisions that they make will affect their popularity.
// - Figureheads are always affected by any decisions made by the leadership.
// - Individuals can propose actions, which they will choose depending on their traits.
// - Individuals that care about their reputation will try to make popular decisions
// - Individuals that are ambitious will try to make decisions that benefit them personally.
// - If someone cares about their popularity, but is also ambitious, they might try to
//   covertly make decisions that benefit them personally, while appearing to make popular decisions.
//
// There should be a requirement for some governing forms to develop, like a democracy requires a certain
// level of literacy (or education), a certain level of wealth, etc.

// For now we just implement a simple leadership structure.
type Leadership struct {
	*gengovernment.Leadership
	Leader *Person
}

func newLeadership(f gengovernment.LeadershipForm) *Leadership {
	return &Leadership{
		Leadership: f.Generate(),
	}
}

// GetTitleForPerson returns the title for the given person.
func (l *Leadership) GetTitleForPerson(p *Person) string {
	if g := p.Gender(); g == GenderMale {
		return l.GetTitle(gengovernment.GenderMale)
	} else if g == GenderFemale {
		return l.GetTitle(gengovernment.GenderFemale)
	}
	return ""
}

// String returns a string representation of the leadership.
func (l *Leadership) String() string {
	if l.Leader == nil {
		return "None"
	}
	title := l.GetTitleForPerson(l.Leader)
	return fmt.Sprintf("%s %s (%s)", title, l.Leader.Name(), l.Form.String())
}

// ChangeTitle changes the title of the leader.
func (l *Leadership) ChangeTitle(h *History) {
	if l.Leader != nil {
		oldTitle := l.GetTitleForPerson(l.Leader)
		l.Title = l.Form.Title(l.LeaderPreferredTags()...)
		newTitle := l.GetTitleForPerson(l.Leader)

		l.Leader.Title = newTitle

		// Add history event for title change.
		h.AddEvent("leadership", fmt.Sprintf("%s of faction %s changed title from %s to %s.", l.Leader.Name(), l.Leader.Name(), oldTitle, newTitle), l.Leader.Ref())
	} else {
		newTitle := l.Form.Title(l.LeaderPreferredTags()...)
		l.Title = newTitle
	}
}

// LeaderPreferredTags returns the preferred title tags for the leader.
func (l *Leadership) LeaderPreferredTags() []gengovernment.TitleTag {
	var preferred []gengovernment.TitleTag
	if l.Leader != nil {
		if l.Leader.Traits.HasTrait(geneticshuman.TraitAmbitious) {
			preferred = append(preferred, gengovernment.TitleTagPower, gengovernment.TitleTagGrandeur, gengovernment.TitleTagSensuality)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitDeceptive) {
			preferred = append(preferred, gengovernment.TitleTagMythical, gengovernment.TitleTagCruelty, gengovernment.TitleTagMagic, gengovernment.TitleTagDivine, gengovernment.TitleTagSensuality, gengovernment.TitleTagGrandeur)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitHonest) {
			preferred = append(preferred, gengovernment.TitleTagWisdom, gengovernment.TitleTagService)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitAggressive) {
			preferred = append(preferred, gengovernment.TitleTagMilitary, gengovernment.TitleTagCruelty, gengovernment.TitleTagPower)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitCalm) {
			preferred = append(preferred, gengovernment.TitleTagWisdom, gengovernment.TitleTagService)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitCowardly) {
			preferred = append(preferred, gengovernment.TitleTagOther)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitBrave) {
			preferred = append(preferred, gengovernment.TitleTagMilitary, gengovernment.TitleTagPower)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitContent) {
			preferred = append(preferred, gengovernment.TitleTagOther)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitCareless) {
			preferred = append(preferred, gengovernment.TitleTagGrandeur, gengovernment.TitleTagMagic)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitCareful) {
			preferred = append(preferred, gengovernment.TitleTagWisdom, gengovernment.TitleTagService)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitParanoid) {
			preferred = append(preferred, gengovernment.TitleTagDivine, gengovernment.TitleTagMagic)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitTrusting) {
			preferred = append(preferred, gengovernment.TitleTagWisdom, gengovernment.TitleTagService)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitCruel) {
			preferred = append(preferred, gengovernment.TitleTagCruelty, gengovernment.TitleTagPower)
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitKind) {
			preferred = append(preferred, gengovernment.TitleTagWisdom, gengovernment.TitleTagService)
		}
	}
	return preferred
}

// LeaderPreferredForm returns the preferred form for the leader.
func (l *Leadership) LeaderPreferredForm() gengovernment.LeadershipForm {
	// TODO THIS SHOULD ONLY TAKE IN ACCOUNT FORMS THAT ARE AVAILABLE TO THE TRIBE OR FACTION
	if l.Leader == nil {
		return l.Form
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitAmbitious) {
		if l.Leader.Traits.HasTrait(geneticshuman.TraitCruel) {
			return gengovernment.LeadershipFormDictatorship
		}
		if l.Leader.Traits.HasTrait(geneticshuman.TraitKind) {
			return gengovernment.LeadershipFormDemocracy
		}
		return gengovernment.LeadershipFormMonarchy
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitDeceptive) {
		// return gengovernment.LeadershipFormTheocracy
		return gengovernment.LeadershipFormDictatorship
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitHonest) {
		return gengovernment.LeadershipFormDemocracy
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitAggressive) {
		return gengovernment.LeadershipFormDictatorship
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitCalm) {
		return gengovernment.LeadershipFormCouncil
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitCowardly) {
		return gengovernment.LeadershipFormCouncil
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitBrave) {
		return gengovernment.LeadershipFormDictatorship
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitContent) {
		return gengovernment.LeadershipFormCouncil
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitCareless) {
		// return gengovernment.LeadershipFormTheocracy
		return gengovernment.LeadershipFormDictatorship
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitCareful) {
		return gengovernment.LeadershipFormCouncil
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitParanoid) {
		// return gengovernment.LeadershipFormTheocracy
		return gengovernment.LeadershipFormDictatorship
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitTrusting) {
		return gengovernment.LeadershipFormDemocracy
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitCruel) {
		return gengovernment.LeadershipFormDictatorship
	}
	if l.Leader.Traits.HasTrait(geneticshuman.TraitKind) {
		return gengovernment.LeadershipFormDemocracy
	}
	return l.Form
}

// ChooseNewLeader chooses a new leader for the leadership.
// TODO: This should allow for multiple leaders, a council, etc.
// So the leader that needs to be replaced should be specified.
func (l *Leadership) ChooseNewLeader(t *Tribe, f *Faction, m *Civ) *Person {
	var candidate *Person
	pickGender := func() geneticshuman.Gender {
		if l.GenderRestriction == gengovernment.GenderRestrictionMale {
			return geneticshuman.GenderMale
		}
		if l.GenderRestriction == gengovernment.GenderRestrictionFemale {
			return geneticshuman.GenderFemale
		}
		if rand.Intn(100) < 50 {
			return geneticshuman.GenderMale
		}
		return geneticshuman.GenderFemale
	}
	if l.Leader != nil && (l.Succession == gengovernment.SuccessionTypeInherited || rand.Intn(100) < 50) && l.Leader.Age > ageOfAdulthood {
		// Inherited or random chance.
		// Sort all children by age, start from the oldest and find one
		// with a matching gender
		children := l.Leader.Children
		sort.Slice(children, func(i, j int) bool {
			return children[i].Age < children[j].Age
		})
		for _, c := range children {
			// TODO: Warn of children that seem to already be leaders!
			if c.Title != "" {
				log.Printf("Child %s already has a title %q!", c.Name(), c.Title)
				continue
			}
			if c.isDead() {
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
		// Sort all people by popularity, start from the most popular and find someone
		// who is maybe ambitious, etc.
		people := t.People
		sort.Slice(people, func(i, j int) bool {
			return people[i].Popularity > people[j].Popularity
		})
		for _, p := range people {
			if p == l.Leader || p.isDead() || p.Age < ageOfAdulthood || p.Gender() != gender {
				continue
			}
			if p.Title != "" {
				log.Printf("Person %s already has a title %q!", p.Name(), p.Title)
				continue
			}
			candidate = p
			break
		}
		if candidate == nil {
			candidate = t.NewRandomPerson(m, gender)
		}
	}
	// Add history event.
	h := m.History
	h.AddEvent("leadership", fmt.Sprintf("%s of faction %s was chosen to be the new leader.", candidate.Name(), f.Name), f.Ref())
	return l.ReplaceLeader(candidate, t, f, h)
}

// ReplaceLeader replaces the leader of the faction with a new one and returns the old leader.
func (l *Leadership) ReplaceLeader(newLeader *Person, t *Tribe, f *Faction, h *History) *Person {
	if l.Leader == nil {
		l.Leader = newLeader
		return nil
	}
	oldLeader := l.Leader
	oldLeader.Title += " (former)"
	newLeader.Title = l.GetTitleForPerson(newLeader)
	l.Leader = newLeader
	// Add history event.
	h.AddEvent("leadership", fmt.Sprintf("%s %s of faction %s was replaced by %s.", l.GetTitleForPerson(oldLeader), oldLeader.Name(), f.Name, newLeader.Name()), t.Ref())
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

// Faction represents a faction within a tribe or a city state.
type Faction struct {
	ID   int
	Name string // Name of the faction.
	// TODO: Add faction properties.
	// Type (e.g. religious, military, etc.)
	Type FactionType
	// TODO: Add leadership properties.
	// Type (e.g. democratic, autocratic, etc.
	// Titles (e.g. king, council, etc.)
	// Leader(s) (e.g. council, king, etc.)
	*Leadership
	// Influence  ClampedVal // Influence or popularity will determine the power of the leadership.
	Popularity ClampedVal // Popularity will determine the happiness of the tribe.
	lang       *genlanguage.Language
}

// genFaction generates a new faction with the given name.
func genFaction(t *Tribe, m *Civ, f gengovernment.LeadershipForm, leader *Person) *Faction {
	lang := t.Language

	// TODO: Migrate the name generation to a dedicated function.
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
			// TODO: Like religion, allow place names, etc.
			name += lang.MakeName()
		}
	}
	fac := &Faction{
		ID:         getFactionID(),
		Name:       name,
		Leadership: newLeadership(f),
		// Influence:  1.0,
		Popularity: 1.0,
		lang:       lang,
		Type:       FactionType(rand.Intn(int(FactionTypeMax))),
	}
	// TODO: This should be done before generating the leadership.
	if leader == nil {
		fac.ChooseNewLeader(t, fac, m)
	} else {
		fac.ReplaceLeader(leader, t, fac, m.History)
	}
	fac.Leadership.ChangeTitle(m.History)

	return fac
}

// Ref returns a reference to the faction.
func (f *Faction) Ref() ObjectReference {
	return ObjectReference{ID: f.ID, Type: ObjectTypeFaction}
}

// String returns a string representation of the faction.
func (f *Faction) String() string {
	return fmt.Sprintf("%s (%s - %.2f)", f.Name, f.Leadership.String(), f.Popularity)
}

// ChangeType changes the type of the faction.
func (f *Faction) ChangeType(newType FactionType, leadForm gengovernment.LeadershipForm, h *History) FactionType {
	// TODO: Make this dependent on the tribe's state.
	// Depending on who is making this change, we might want to change the gender restriction.
	// f.Leadership.GenderRestriction = leadForm.getGenderRestriction()
	// Add history event.
	oldType := f.Type
	if oldType != newType {
		f.Type = newType
		h.AddEvent("faction", fmt.Sprintf("Faction %s changed type from %s to %s.", f.Name, oldType.String(), newType.String()), f.Ref())
	}
	oldForm := f.Leadership.Form
	if oldForm != leadForm {
		// If it is still nomadic, it should pick chiefdom, if it is settled, it should pick a monarchy, etc.
		f.Leadership.ChangeForm(leadForm)
		h.AddEvent("faction", fmt.Sprintf("Faction %s changed leadership form from %s to %s.", f.Name, oldForm.String(), leadForm.String()), f.Ref())
	}
	return oldType
}

// ExecutePerson executes a person (e.g. leader) of the faction.
func (f *Faction) ExecutePerson(p *Person, reason string, m *Civ) {
	h := m.History
	m.killPerson(p, reason)
	// TODO: Add history event.
	// TODO: Choose method of execution.
	// - This might be a preference of the faction.
	// - This might be chosen based on the severity of the infraction.
	// - This might be chosen based on the popularity of the leader.
	h.AddEvent("faction", fmt.Sprintf("%s %s of faction %s was executed for %s.", f.GetTitleForPerson(p), p.Name(), f.Name, reason), p.Ref())
}

// Tick the faction for one year.
func (f *Faction) Tick(t *Tribe, m *Civ) {
	h := m.History

	// Update the leadership.
	// TODO: This should allow for multiple leaders, a council, etc.
	leaders := f.Leaders()
	for _, leader := range leaders {
		if leader.Dead() {
			h.AddEvent("leadership", fmt.Sprintf("%s %s of faction %s died.", f.GetTitleForPerson(leader), leader.Name(), f.Name), f.Ref())
			f.Leadership.ChooseNewLeader(t, f, m) // This should specify which leader to replace.
		}
	}

	// TODO: Add action for faction.
	// - This might be an action to reduce the popularity of a rival faction or leader.
	// - This might be an action to increase the popularity of the faction.
	// Execute a random action.
	if rand.Float64() < 0.1 {
		action := f.pickAction(t, m)
		if action != nil {
			var flavor string
			if len(action.flavors) > 0 {
				flavor = action.flavors[rand.Intn(len(action.flavors))]
			}
			h.AddEvent("faction", action.String(f, flavor), f.Ref())
			action.Execute(f, flavor)
			f.Popularity.Add(action.changePopularity)
		}
	}
}

// FactionAction represents an action that a faction can take.
// TODO: Add consequences, etc.
type FactionAction struct {
	fmtStr           string
	flavors          []string
	changePopularity float64
	probability      float64
	requires         func(fac *Faction) bool
	consequences     func(fac *Faction, flavor string)
}

func (f *FactionAction) String(fac *Faction, flavor string) string {
	str := f.fmtStr
	str = strings.ReplaceAll(str, "[FACTION]", fac.Name)
	str = strings.ReplaceAll(str, "[LEADER]", fac.GetTitleForPerson(fac.Leader)+" "+fac.Leader.Name())
	if len(f.flavors) > 0 {
		str = strings.ReplaceAll(str, "[FLAVOR]", flavor)
	}
	return str
}

func (f *FactionAction) Execute(fac *Faction, flavor string) {
	if f.consequences != nil {
		f.consequences(fac, flavor)
	}
}

func (f *Faction) pickAction(t *Tribe, m *Civ) *FactionAction {
	h := m.History

	// Depending on the form of leadership, the actions will be different and decided by different people.
	// Right now, we just have a single leader in each type of leadership.

	// TODO: The actions should depend on the personality of the faction or faction leader.
	// TODO: Can consequences be chains of actions which are then narrated?
	// There is a tree or sequence of actions, which will be played out and the
	// resulting chain of events will be used to generate the narrative.
	// We can store the chain of events to store the whole sequence of events.
	actionsLeaderNegative := []*FactionAction{{
		fmtStr:           "[LEADER] of faction [FACTION] is discovered to be a [FLAVOR].",
		flavors:          []string{"spy", "traitor", "criminal", "fraud"},
		changePopularity: 0.7,
		probability:      0.1,
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitDeceptive) // && !leader.Traits.HasTrait(geneticshuman.TraitCareful)
		},
		consequences: func(fac *Faction, flavor string) {
			oldLeader := fac.ChooseNewLeader(t, fac, m)
			fac.ExecutePerson(oldLeader, "treason", m)
		},
	}, {
		fmtStr:           "[LEADER] of faction [FACTION] has been [FLAVOR].",
		flavors:          []string{"stealing", "lying", "abusing power"},
		changePopularity: 0.7,
		probability:      0.1,
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitDeceptive) // && !leader.Traits.HasTrait(geneticshuman.TraitCareful)
		},
		consequences: func(fac *Faction, flavor string) {
			leader := fac.Leader
			detectionChance := 0.5
			if leader.Traits.HasTrait(geneticshuman.TraitCareful) {
				detectionChance = 0.2
			} else if leader.Traits.HasTrait(geneticshuman.TraitCareless) {
				detectionChance = 0.8
			}

			// Check if the scandal is discovered.
			if rand.Float64() < detectionChance {
				// The leader is discovered and will be replaced.
				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has been caught %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Scandal (Faction)", historyMsg, fac.Ref())

				oldLeader := fac.ChooseNewLeader(t, fac, m)
				fac.ExecutePerson(oldLeader, "corruption", m)
			} else {
				// The leader will be able to cover up the scandal.
				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has almost been caught %s but managed to cover it up.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Scandal (Faction)", historyMsg, fac.Ref())
			}
		},
	}, {
		fmtStr:           "[LEADER] of faction [FACTION] is planning a violent [FLAVOR].",
		flavors:          []string{"coup", "rebellion", "uprising"},
		changePopularity: 0.7,
		probability:      0.1,
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			if leader.Traits.HasTrait(geneticshuman.TraitKind) || leader.Traits.HasTrait(geneticshuman.TraitContent) {
				return false
			}
			// TODO: Check if the leader has enough foresight to check if they have a chance of success.
			return (leader.Traits.HasTrait(geneticshuman.TraitAmbitious) || leader.Traits.HasTrait(geneticshuman.TraitDeceptive)) && fac != t.Leadership
		},
		consequences: func(fac *Faction, flavor string) {
			leader := fac.Leader
			detectionChance := 0.5
			if leader.Traits.HasTrait(geneticshuman.TraitCareful) {
				detectionChance = 0.2
			} else if leader.Traits.HasTrait(geneticshuman.TraitCareless) {
				detectionChance = 0.8
			}

			// Check if the plan is discovered.
			if rand.Float64() < detectionChance {
				// The plan is discovered and will be stopped.
				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has been caught planning a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Intrigue (Faction)", historyMsg, fac.Ref())
			} else {
				// The plan will be executed.
				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully executed a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Uprising (Faction)", historyMsg, fac.Ref())

				// The faction will take over.
				// Depending on chance and popularity, it might kill the leadership, or simply replace it.
				// If the faction is more popular than the leadership, it will simply replace it.
				executeCoup(t, fac, m)
			}
		},
	}}
	actionsLeaderPositive := []*FactionAction{{
		fmtStr:           "[LEADER] of faction [FACTION] is discovered to be a [FLAVOR].",
		flavors:          []string{"hero", "saint", "genius", "visionary"},
		changePopularity: 1.3,
		probability:      0.1,
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAmbitious) && !leader.Traits.HasTrait(geneticshuman.TraitDeceptive)
		},
	}, {
		fmtStr:           "[LEADER] of faction [FACTION] is pursuing [FLAVOR].",
		flavors:          []string{"charity", "innovation", "reform"},
		changePopularity: 1.3,
		probability:      0.1,
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAmbitious) && !leader.Traits.HasTrait(geneticshuman.TraitDeceptive)
		},
		consequences: func(fac *Faction, flavor string) {
			if flavor == "reform" {
				// Change the type of the faction.
				newType := FactionType(rand.Intn(int(FactionTypeMax)))
				for newType == fac.Type {
					newType = FactionType(rand.Intn(int(FactionTypeMax)))
				}
				oldType := fac.ChangeType(newType, f.LeaderPreferredForm(), h)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has reformed the faction from %s to %s.", fac.ID, fac.Leader.Name(), fac.Name, oldType.String(), newType.String())
				h.AddEvent("Reform (Faction)", historyMsg, fac.Ref())
			}
		},
	}}
	actionsLeaderManipulative := []*FactionAction{{
		fmtStr:           "[LEADER] of faction [FACTION] is stoking [FLAVOR] against a rival faction.",
		flavors:          []string{"hatred", "fear", "distrust"},
		changePopularity: 1.3,
		probability:      0.1,
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAmbitious) && (leader.Traits.HasTrait(geneticshuman.TraitDeceptive) || leader.Traits.HasTrait(geneticshuman.TraitCruel))
		},
		consequences: func(fac *Faction, flavor string) {
			// TODO: Select rival faction.
			rivalIsLeader := true
			rival := t.Leadership
			if rival == fac { // Select a different faction.
				rivalIsLeader = false
				for idx := range rand.Perm(len(t.Factions)) {
					if t.Factions[idx] != fac {
						rival = t.Factions[idx]
						break
					}
				}
			}
			// There is a chance that this plan will backfire and our popularity will decrease
			// while the rival faction will gain popularity.
			// The lower the popularity of the rival faction, the higher the chance that this plan will backfire.
			if rand.Float64() > float64(fac.Popularity)*0.9 {
				// The rival faction will become more popular and we will lose popularity.
				rival.Popularity.Add(0.3)
				if rivalIsLeader {
					t.Satisfaction.Add(0.1)
				}
				fac.Popularity.Add(-0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s tried to discredit faction %d's leader %s of %s but failed.", fac.ID, fac.Leader.Name(), fac.Name, rival.ID, rival.Leader.Name(), rival.Name)
				h.AddEvent("Intrigue (Faction)", historyMsg, fac.Ref())
			} else {
				// The rival faction will lose popularity and we will gain popularity.
				rival.Popularity.Add(-0.3)
				if rivalIsLeader {
					t.Satisfaction.Add(-0.1)
				}
				fac.Popularity.Add(0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully discredited faction %d's leader %s of %s.", fac.ID, fac.Leader.Name(), fac.Name, rival.ID, rival.Leader.Name(), rival.Name)
				h.AddEvent("Intrigue (Faction)", historyMsg, fac.Ref())
			}
		},
	}, {
		fmtStr:           "Leader [LEADER] of faction [FACTION] is planning a [FLAVOR].",
		flavors:          []string{"assassination", "extortion"},
		changePopularity: 1.3,
		probability:      0.1,
		consequences: func(fac *Faction, flavor string) {
			// TODO: Depending on the traits of the leader or the target, the chance of success might change.
			// Carelessness might lead to failure, while carefulness might lead to success.
			// Paranoid leaders might be harder to assassinate, while trusting leaders might be easier.
			// Depending how clever "we" are, we might choose a target that is easier to extort or assassinate,
			// alternatively we choose who we like least. If we are cruel, we might choose the most popular target.
			switch flavor {
			case "extortion":
				// There is a chance that the extortion will be successful.
				// Find a target for the extortion.
				rivalFaction := t.Leadership
				if rivalFaction == fac {
					for idx := range rand.Perm(len(t.Factions)) {
						if t.Factions[idx] != fac {
							rivalFaction = t.Factions[idx]
							break
						}
					}
				}
				// There is a chance that the extortion will be successful.
				origin := fac.Leader
				target := rivalFaction.Leader
				if rand.Float64() > float64(rivalFaction.Popularity)*0.9 {
					// The extortion was successful.
					// For now, we just reduce the popularity of the rival faction leader.
					rivalFaction.Popularity.Add(-0.3)
					fac.Popularity.Add(0.3)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully extorted faction %d's leader %s of %s.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					h.AddEvent("Extortion (Faction)", historyMsg, fac.Ref())
				} else {
					// The extortion failed.
					// Increase the popularity of the rival faction leader and decrease the popularity of the faction leader.
					rivalFaction.Popularity.Add(0.3)
					fac.Popularity.Add(-0.3)

					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to extort faction %d's leader %s of %s but failed.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					h.AddEvent("Extortion (Faction)", historyMsg, fac.Ref())
				}
			case "assassination":
				// There is a chance that the assassination will be successful.
				// Find a target for the assassination.
				rivalFaction := t.Leadership
				if rivalFaction == fac {
					for idx := range rand.Perm(len(t.Factions)) {
						if t.Factions[idx] != fac {
							rivalFaction = t.Factions[idx]
							break
						}
					}
				}
				// There is a chance that the assassination will be successful.
				origin := fac.Leader
				target := rivalFaction.Leader
				if rand.Float64() > float64(rivalFaction.Popularity)*0.9 {
					// The assassination was successful.
					m.killPerson(target, "assassination by faction "+f.Name)
					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully assassinated faction %d's leader %s of %s.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					h.AddEvent("Assassination (Faction)", historyMsg, fac.Ref())
				} else {
					// The assassination failed.
					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to assassinate faction %d's leader %s of %s but failed.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					event := h.AddEvent("Assassination (Faction)", historyMsg, fac.Ref())
					// TODO: There is a chance that the target will become aware of who tried to assassinate them.
					if rand.Float64() > 0.5 {
						target.Opinions.AddOpinion(origin, -0.5, event)
					}
				}
			default:
				if rand.Float64() > 0.5 {
					// The action was successful.
					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully organized a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Criminal (Faction)", historyMsg, fac.Ref())
				} else {
					// The action failed.
					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to organize a %s but failed.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Criminal (Faction)", historyMsg, fac.Ref())
				}
			}
		},
	}}
	actionsFactionReligious := []*FactionAction{{
		fmtStr:           "Leader [LEADER] of faction [FACTION] is performing a miracle by [FLAVOR].",
		flavors:          []string{"turning wine into water", "making a piglet talk", "walking on beer"},
		changePopularity: 1.3,
		probability:      0.1,
		consequences: func(fac *Faction, flavor string) {
			leaderIsCharlatan := false
			if rand.Float64() > 0.9 {
				leaderIsCharlatan = true
			}

			// If the leader is a charlatan, there is a higher chance that the miracle will fail.
			threshold := float64(fac.Popularity)
			if leaderIsCharlatan {
				threshold *= 0.7
			}
			if rand.Float64() > threshold {
				// The miracle failed.
				fac.Popularity.Add(-0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s tried to perform a miracle but failed.", fac.ID, fac.Leader.Name(), fac.Name)
				h.AddEvent("Religious (Faction)", historyMsg, fac.Ref())

				// TODO: If the leader is a charlatan, there is a chance that the leader will be exposed, depending on the popularity of the faction.
				if leaderIsCharlatan && rand.Float64() > float64(fac.Popularity)*0.9 {
					// The leader was exposed.
					oldLeader := fac.ChooseNewLeader(t, fac, m)
					fac.ExecutePerson(oldLeader, "charlatanism", m)
				}
			} else {
				// The miracle was successful.
				fac.Popularity.Add(0.3)

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s performed a miracle.", fac.ID, fac.Leader.Name(), fac.Name)
				h.AddEvent("Religious (Faction)", historyMsg, fac.Ref())
			}
		},
	}, {
		fmtStr:           "Leader [LEADER] of faction [FACTION] is preaching about the [FLAVOR].",
		flavors:          []string{"end of the world", "coming of the savior", "return of the gods"},
		changePopularity: 1.3,
		probability:      0.1,
	}, {
		fmtStr:           "Leader [LEADER] of faction [FACTION] is organizing a [FLAVOR].",
		flavors:          []string{"festival", "pilgrimage", "sacrifice", "ceremony", "ritual"},
		changePopularity: 1.3,
		probability:      0.1,
		consequences: func(fac *Faction, flavor string) {
			// Just for fun, if it is a ceremony, there is a chance a freak accident will happen.
			// Like: The leader being hit by lightning, a statue falling on someone, etc.
			// Maybe a portal to another dimension opens up and something comes through and eats someone.
			misfortuneChance := 0.1

			// If the leader is a charlatan, there is a higher chance that the miracle will fail
			// or that something bad will happen.
			if fac.Leader.Traits.HasTrait(geneticshuman.TraitDeceptive) {
				misfortuneChance = 0.4
			}

			misfortune := rand.Float64() < misfortuneChance
			if !misfortune {
				// The ceremony was successful.
				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s organized a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Religious (Faction)", historyMsg, fac.Ref())
				return
			}
			switch flavor {
			case "ceremony", "ritual":
				// A freak accident happened.
				const (
					IncidentTypeLightning = iota
					IncidentTypeStatue
					IncidentTypePortal
				)
				incidentType := rand.Intn(3)
				switch incidentType {
				case IncidentTypeLightning:
					// The leader was hit by lightning.
					m.killPerson(fac.Leader, "hit by lightning during "+flavor)
					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s was hit by lightning during a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Misfortune (Faction)", historyMsg, fac.Ref())

					// The popularity of the faction will increase, because it was funny.
					fac.Popularity.Add(0.3)
				case IncidentTypeStatue:
					// A statue fell on someone.
					// Find a random person to kill.
					victim := t.People[rand.Intn(len(t.People))]
					m.killPerson(victim, "hit by falling statue during "+flavor)
					// Add a history entry.
					historyMsg := fmt.Sprintf("A statue fell on %s during a %s organized by faction %d's leader %s of %s.", victim.Name(), flavor, fac.ID, fac.Leader.Name(), fac.Name)
					h.AddEvent("Misfortune (Faction)", historyMsg, fac.Ref())

					// The popularity of the faction will decrease, because it was tragic.
					fac.Popularity.Add(-0.3)
				case IncidentTypePortal:
					// A portal to another dimension opened up.
					// Pick a random monster to come through the portal.
					const (
						MonsterTypeDemon = iota
						MonsterTypeAlien
						MonsterTypeEldritch
					)
					monsterType := rand.Intn(3)
					monsterName := "unknown monster"
					switch monsterType {
					case MonsterTypeDemon:
						monsterName = "demon"
					case MonsterTypeAlien:
						monsterName = "alien"
					case MonsterTypeEldritch:
						monsterName = "eldritch horror"
					}
					// Find a random person to kill.
					victim := t.People[rand.Intn(len(t.People))]
					m.killPerson(victim, "eaten by "+monsterName+" from another dimension")
					// Add a history entry.
					historyMsg := fmt.Sprintf("A portal to another dimension opened up during a %s organized by faction %d's leader %s of %s and %s came through and ate %s.", flavor, fac.ID, fac.Leader.Name(), fac.Name, monsterName, victim.Name())
					h.AddEvent("Misfortune (Faction)", historyMsg, fac.Ref())

					// The popularity of the faction will decrease, because it was tragic.
					fac.Popularity.Add(-0.3)
				}
			case "sacrifice":
				// A freak accident happened.
				// Pick an animal to sacrifice.
				animals := []string{
					"sheep", "goat", "pig", "cow", "chicken", "duck", "goose", "turkey", "rabbit", "dog", "cat", "horse", "donkey", "elephant", "rhinoceros", "hippopotamus", "giraffe", "zebra", "lion", "tiger", "bear", "wolf", "fox", "deer", "moose", "elk", "buffalo", "bison", "antelope", "gazelle", "impala", "kudu", "oryx", "springbok", "ibex", "chamois", "goral", "tahr", "muskox", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison", "buffalo", "cattle", "yak", "water buffalo", "banteng", "gayal", "kouprey", "saola", "bison",
				}
				animal := animals[rand.Intn(len(animals))]

				// Add a body part to be bitten by the animal.
				bodyParts := []string{
					"hand", "foot", "leg", "arm", "head", "ear", "nose", "eye", "mouth", "tongue", "cheek", "chin", "forehead", "neck", "shoulder", "back", "chest", "stomach", "belly", "waist", "hip", "buttocks", "thigh", "knee", "calf", "ankle", "heel", "toe", "finger", "thumb", "nail", "palm", "wrist", "elbow", "forearm", "armpit", "breast", "nipple", "heart", "lung", "liver", "kidney", "bladder", "intestine", "stomach", "spleen", "pancreas", "appendix", "brain", "skull", "rib", "spine", "pelvis", "hip", "thorax", "abdomen", "groin", "genitals",
				}
				bodyPart := bodyParts[rand.Intn(len(bodyParts))]

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s was bitten by a %s during a %s on the %s.", fac.ID, fac.Leader.Name(), fac.Name, animal, flavor, bodyPart)
				h.AddEvent("Misfortune (Faction)", historyMsg, fac.Ref())
			default:
				// The ceremony was successful.
				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s organized a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
				h.AddEvent("Religious (Faction)", historyMsg, fac.Ref())
			}
		},
	}}
	actionsFactionMartial := []*FactionAction{{
		fmtStr:           "Leader [LEADER] of faction [FACTION] has initiated a [FLAVOR].",
		flavors:          []string{"purge of the weak", "mandatory conscription"},
		changePopularity: 0.7,
		probability:      0.1,
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAggressive)
		},
		consequences: func(fac *Faction, flavor string) {
			// For this to have a positive effect, the faction must have extremely high popularity.
			if rand.Float64() > float64(fac.Popularity)*0.9 {
				// This action will increase the popularity of the faction.
				fac.Popularity.Add(0.4 * rand.Float64())

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has initiated a %s.", fac.ID, fac.Leader.Name(), fac.Name, "purge of the weak which was welcomed by the members.")
				h.AddEvent("Martial (Faction)", historyMsg, fac.Ref())
			} else {
				// This action will decrease the popularity of the faction.
				t.Satisfaction.Add(-0.4 * rand.Float64())

				// Add a history entry.
				historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has initiated a %s.", fac.ID, fac.Leader.Name(), fac.Name, "purge of the weak which was met with resistance.")
				h.AddEvent("Martial (Faction)", historyMsg, fac.Ref())
			}
		},
	}, {
		fmtStr:           "Leader [LEADER] of faction [FACTION] is preparing for a [FLAVOR].",
		flavors:          []string{"war", "raid", "battle", "training", "tournament"},
		changePopularity: 1.3,
		probability:      0.1,
		requires: func(fac *Faction) bool {
			leader := fac.Leader
			return leader.Traits.HasTrait(geneticshuman.TraitAggressive) && leader.Traits.HasTrait(geneticshuman.TraitAmbitious)
		},
	}}
	actionsFactionCivil := []*FactionAction{{
		fmtStr:           "Leader [LEADER] of faction [FACTION] is organizing a [FLAVOR].",
		flavors:          []string{"festival", "market", "election", "celebration", "meeting"},
		changePopularity: 1.3,
		probability:      0.1,
	}}
	actionsFactionCriminal := []*FactionAction{{
		fmtStr:           "Leader [LEADER] of faction [FACTION] is planning a [FLAVOR].",
		flavors:          []string{"raid", "heist", "smuggling", "assassination", "extortion"},
		changePopularity: 1.3,
		probability:      0.1,
		consequences: func(fac *Faction, flavor string) {
			switch flavor {
			case "assassination":
				// There is a chance that the assassination will be successful.
				// Find a target for the assassination.
				rivalFaction := t.Leadership
				if rivalFaction == fac {
					for idx := range rand.Perm(len(t.Factions)) {
						if t.Factions[idx] != fac {
							rivalFaction = t.Factions[idx]
							break
						}
					}
				}
				// There is a chance that the assassination will be successful.
				origin := fac.Leader
				target := rivalFaction.Leader
				if rand.Float64() > float64(rivalFaction.Popularity)*0.9 {
					// The assassination was successful.
					m.killPerson(target, "assassination by faction "+f.Name)
					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully assassinated faction %d's leader %s of %s.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					h.AddEvent("Assassination (Faction)", historyMsg, fac.Ref())
				} else {
					// The assassination failed.
					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to assassinate faction %d's leader %s of %s but failed.", fac.ID, origin.Name(), fac.Name, rivalFaction.ID, target.Name(), rivalFaction.Name)
					event := h.AddEvent("Assassination (Faction)", historyMsg, fac.Ref())
					// TODO: There is a chance that the target will become aware of who tried to assassinate them.
					if rand.Float64() > 0.5 {
						target.Opinions.AddOpinion(origin, -0.5, event)
					}
				}
			default:
				if rand.Float64() > 0.5 {
					// The action was successful.
					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has successfully organized a %s.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Criminal (Faction)", historyMsg, fac.Ref())
				} else {
					// The action failed.
					// Add a history entry.
					historyMsg := fmt.Sprintf("Faction %d's leader %s of %s has tried to organize a %s but failed.", fac.ID, fac.Leader.Name(), fac.Name, flavor)
					h.AddEvent("Criminal (Faction)", historyMsg, fac.Ref())
				}
			}
		},
	}}
	actionsFactionMerchant := []*FactionAction{{
		fmtStr:           "Leader [LEADER] of faction [FACTION] is organizing a [FLAVOR].",
		flavors:          []string{"trade", "market", "caravan", "deal", "negotiation"},
		changePopularity: 1.3,
		probability:      0.1,
	}}

	// Pick an action based on the type of the faction.
	var actions []*FactionAction
	switch f.Type {
	case FactionTypeReligious:
		actions = actionsFactionReligious
	case FactionTypeMartial:
		actions = actionsFactionMartial
	case FactionTypeCivil:
		actions = actionsFactionCivil
	case FactionTypeCriminal:
		actions = actionsFactionCriminal
	case FactionTypeMerchant:
		actions = actionsFactionMerchant
	}

	actions = append(actions, actionsLeaderNegative...)
	actions = append(actions, actionsLeaderPositive...)
	actions = append(actions, actionsLeaderManipulative...)

	// Filter out actions that require a certain condition.
	var filteredActions []*FactionAction
	for _, action := range actions {
		if action.requires == nil || action.requires(f) {
			filteredActions = append(filteredActions, action)
		}
	}

	// Pick an action based on probability.
	for _, idx := range rand.Perm(len(filteredActions)) {
		action := filteredActions[idx]
		if rand.Float64() < action.probability {
			return action
		}
	}
	return nil
}
