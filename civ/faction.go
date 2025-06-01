package civ

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

// compare two leaderships and return a the compatibility.
func (l *Leadership) compare(other *Leadership) float64 {
	if l == other {
		return 1.0
	}
	if l == nil || other == nil {
		return -1.0
	}

	compatibility := 0.0

	// Compare the form
	if l.Form == other.Form {
		compatibility += 1.0
	} else {
		compatibility -= 1.0
	}

	// Compare the successions
	if l.Succession == other.Succession {
		compatibility += 1.0
	} else {
		compatibility -= 1.0
	}

	// Compare gender restrictions
	if l.GenderRestriction == other.GenderRestriction {
		compatibility += 1.0
	} else {
		compatibility -= 1.0
	}

	// Compare the leaders.
	compatibility = l.Leader.compare(other.Leader)

	// Get the opinions of the leaders.
	opA := l.Leader.Opinions.GetOpinion(other.Leader)
	opB := other.Leader.Opinions.GetOpinion(l.Leader)
	compatibility += (opA + opB) / 2

	return compatibility / 5
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

type peopleThing interface {
	GetID() int
	Ref() ObjectReference
	GetPeople() []*Person
	GetGoverningPeople() *GoverningPeople
	NewRandomPerson(m *Civ, gender geneticshuman.Gender) *Person
	NewRandomChild(m *Civ, gender geneticshuman.Gender, parent *Person) *Person
	getPreferredLeadershipForm() gengovernment.LeadershipForm
	getPossibleLeadershipForms() []gengovernment.LeadershipForm
	findNaturalProgression() gengovernment.LeadershipForm
	findCoupProgression() gengovernment.LeadershipForm
	getLanguage() *genlanguage.Language
	getFactionActions(m *Civ, f *Faction) []*FactionAction
	String() string
}

// ChooseNewLeader chooses a new leader for the leadership.
// TODO: This should allow for multiple leaders, a council, etc.
// So the leader that needs to be replaced should be specified.
func (l *Leadership) ChooseNewLeader(t peopleThing, f *Faction, m *Civ) *Person {
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
		people := t.GetPeople()
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
func (l *Leadership) ReplaceLeader(newLeader *Person, t peopleThing, f *Faction, h *History) *Person {
	if l.Leader == nil {
		l.Leader = newLeader
		return nil
	}
	// Update the title of the old leader.
	oldLeader := l.Leader
	oldLeader.Title += " (former)"

	// Update the title of the new leader.
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
	ID         int                   // ID of the faction.
	Name       string                // Name of the faction.
	Type       FactionType           // Type of the faction (e.g. religious, military, etc.)
	lang       *genlanguage.Language // Used for naming.
	Popularity ClampedVal            // Popularity will determine the happiness of the tribe.
	// Influence  ClampedVal // Influence or popularity will determine the power of the leadership.
	*Leadership // Leadership of the faction, including the form of leadership and the leader(s).
}

func genFaction(t peopleThing, m *Civ, f gengovernment.LeadershipForm, leader *Person) *Faction {
	lang := t.getLanguage()
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

// compare compares the faction to another faction.
func (f *Faction) compare(other *Faction) float64 {
	if f == nil || other == nil {
		return -1.0
	}

	compatibility := f.Leadership.compare(other.Leadership)
	// Compare the type.
	if f.Type == other.Type {
		compatibility += 1.0
	} else {
		compatibility -= 1.0
	}
	// Compare the popularity.
	compatibility += float64(f.Popularity-other.Popularity) / 2
	return compatibility / 3
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
	// TODO: Choose method of execution.
	// - This might be a preference of the faction.
	// - This might be chosen based on the severity of the infraction.
	// - This might be chosen based on the popularity of the leader.
	m.killPerson(p, reason)

	// Add history event.
	h.AddEvent("faction", fmt.Sprintf("%s %s of faction %s was executed for %s.", f.GetTitleForPerson(p), p.Name(), f.Name, reason), p.Ref())
}

// Tick the faction for one year.
func (f *Faction) Tick(t peopleThing, m *Civ) {
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
}

func (f *Faction) pickAction(t peopleThing, g *GoverningPeople, m *Civ) *FactionAction {
	actions := t.getFactionActions(m, f)

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
		if rand.Float64() < action.probability(f) {
			return action
		}
	}
	return nil
}

// FactionAction represents an action that a faction can take.
// TODO: Add consequences, etc.
type FactionAction struct {
	probability  func(fac *Faction) float64
	requires     func(fac *Faction) bool
	consequences func(c peopleThing, m *Civ, fac *Faction)
}

func facActionString(str string, fac *Faction, flavor string) string {
	str = strings.ReplaceAll(str, "[FACTION]", fac.Name)
	str = strings.ReplaceAll(str, "[LEADER]", fac.GetTitleForPerson(fac.Leader)+" "+fac.Leader.Name())
	if flavor != "" {
		str = strings.ReplaceAll(str, "[FLAVOR]", flavor)
	}
	return str
}

func (f *FactionAction) Execute(c peopleThing, m *Civ, fac *Faction) {
	if f.consequences != nil {
		f.consequences(c, m, fac)
	}
}

type GoverningPeople struct {
	Leadership      *Faction
	Factions        []*Faction
	Satisfaction    ClampedVal           // Happiness overall.
	SatisfactionAvg *RunningAverageLimit // Running average of satisfaction.
	Gold            int                  // Gold of the governing people.
}

func newGoverningPeople() *GoverningPeople {
	return &GoverningPeople{
		Satisfaction:    1.0,
		SatisfactionAvg: NewRunningAverageLimit(100),
	}
}

func (g *GoverningPeople) handleLeadership(t peopleThing, m *Civ) {
	// The lower the satisfaction, the higher the chance that a faction will form.
	if g.Satisfaction < 0.75 && rand.Float64() > float64(g.Satisfaction) {
		log.Printf("%s is unhappy. A new faction might form.", t.String())
		if len(g.Factions) == 0 || rand.Float64() < 0.1/float64(len(g.Factions)) {
			// Pick a LeadershipForm opposed to the current leadership.
			// - If the leadership is a monarchy, the faction should be a republic, etc.
			// - If the leadership is a council, the faction should be a dictatorship, etc.
			form := t.findCoupProgression()

			// Find a leader for the new faction.
			origLeader := g.Leadership.Leader

			// Avoid anyone that is already a leader.
			// TODO: Find a better way to do this.
			// There should be a reference to a leadership role for each person.
			seenLeaders := make(map[*Person]bool)
			for _, l := range g.Leadership.Leaders() {
				seenLeaders[l] = true
			}

			for _, f := range g.Factions {
				for _, l := range f.Leaders() {
					seenLeaders[l] = true
				}
			}

			// Find the most different person.
			// TODO:
			// - Also take opinion and reputation into account.
			// - And pick a person that is open to the form we want.
			mostDifferent := 0.0
			var leader *Person
			for _, p := range t.GetPeople() {
				if p.Dead() || p.Age < ageOfAdulthood || seenLeaders[origLeader] {
					continue
				}
				similarity := p.compare(origLeader)
				if similarity < mostDifferent {
					mostDifferent = similarity
					leader = p
				}
			}

			// Create a new faction.
			f := genFaction(t, m, form, leader)
			g.Factions = append(g.Factions, f)

			// Add a history entry.
			historyMsg := fmt.Sprintf("%s (at %.2f) has formed a new faction %s.", t.String(), float64(g.Satisfaction), f.String())
			m.History.AddEvent("Founding (Faction)", historyMsg, t.Ref())
		} else if len(g.Factions) > 0 && rand.Float64() > float64(g.Leadership.Popularity) {
			// There is a chance that a faction, more popular than the leadership, will try to take over.
			sort.Slice(g.Factions, func(i, j int) bool {
				return g.Factions[i].Popularity > g.Factions[j].Popularity
			})
			if topFaction := g.Factions[0]; topFaction.Popularity > g.Leadership.Popularity {
				executeTakeoverTribe(t, g, topFaction, m)
			}
		}
	}
}

// executeTakeoverTribe executes a takeover of the tribe by a faction.
func executeTakeoverTribe(t peopleThing, g *GoverningPeople, topFaction *Faction, m *Civ) {
	// The top faction will take over.
	// Depending on chance and popularity, it might kill the leadership, or simply replace it.
	//
	// TODO:
	// - Take note of this event.
	// - Change opinion of factions, etc.
	if oldLeadership := g.Leadership; rand.Float64() > float64(oldLeadership.Popularity) {
		// The more unpopular the leadership, the higher the chance that leadership will be killed.
		executeCoup(t, g, topFaction, m)
	} else {
		// Non-violent transition.
		executeTransition(t, g, topFaction, m)
	}
}

func executeCoup(t peopleThing, g *GoverningPeople, newLeadership *Faction, m *Civ) {
	// Get the new type based on a coup.
	newForm := t.findCoupProgression()
	oldLeadership := g.executeCoup(newForm, newLeadership, m)

	// Add a history entry.
	historyMsg := fmt.Sprintf("Tribe %d's unpopular leadership (%s at %.2f) was lynched by faction %s at %.2f", t.GetID(), oldLeadership.Name, oldLeadership.Popularity, newLeadership.Name, newLeadership.Popularity)
	m.History.AddEvent("Uprising (Faction)", historyMsg, t.Ref())
}

func executeTransition(t peopleThing, g *GoverningPeople, newLeadership *Faction, m *Civ) {
	// Get the new type based on a transition.
	newForm := t.findNaturalProgression()
	oldLeadership := g.executeTransition(newForm, newLeadership, m)

	// Add a history entry.
	historyMsg := fmt.Sprintf("Tribe %d's leadership (%s at %.2f) was replaced by faction %s at %.2f", t.GetID(), oldLeadership.Name, oldLeadership.Popularity, newLeadership.Name, newLeadership.Popularity)
	m.History.AddEvent("Election (Faction)", historyMsg, t.Ref())
}

func (g *GoverningPeople) executeTransition(newForm gengovernment.LeadershipForm, newLeadership *Faction, m *Civ) *Faction {
	oldLeadership := g.Leadership

	// The faction will simply replace the current leadership
	// and we swap the spot in the secondary factions.
	for i, tf := range g.Factions {
		if tf == oldLeadership {
			g.Factions[i] = oldLeadership
			break
		}
	}
	g.Leadership = newLeadership

	newLeadership.ChangeType(FactionTypeCivil, newForm, m.History)
	oldLeadership.ChangeType(FactionTypeCivil, gengovernment.LeadershipFormChiefdom, m.History) // TODO: If this was a dictatorship, autocratic... milder version of ruling class

	return oldLeadership
}

func (g *GoverningPeople) executeCoup(newForm gengovernment.LeadershipForm, newLeadership *Faction, m *Civ) *Faction {
	oldLeadership := g.Leadership

	// Remove new leadership from the list of secondary factions.
	newFactions := make([]*Faction, 0, len(g.Factions)-1)
	for _, fac := range g.Factions {
		if fac == newLeadership {
			continue
		}
		newFactions = append(newFactions, fac)
	}
	g.Factions = newFactions
	g.Leadership = newLeadership

	newLeadership.ChangeType(FactionTypeCivil, newForm, m.History)

	// Execute the old leadership.
	for _, oldLead := range oldLeadership.Leaders() {
		g.Leadership.ExecutePerson(oldLead, "was deemed to be person non grata by the new leadership", m)
	}

	return oldLeadership
}

var defaultFactionActions = []*FactionAction{{
	probability: func(fac *Faction) float64 { return 0.1 },
	requires: func(fac *Faction) bool {
		return fac.Popularity < 0.5
	},
	consequences: func(c peopleThing, m *Civ, fac *Faction) {
		// Low popularity, the faction will try to increase its popularity.
		// Depending on the personality of the leader, this might be done
		// in different ways.
		//
		// A kind leader might try charity, or a public event.
		// A cruel leader might try to intimidate the people.
		// A deceptive leader might try to manipulate the people.
		switch {
		case fac.Leader.Traits.HasTrait(geneticshuman.TraitKind):
			// Charity or a public event.
			// TODO: Check how much money we have, etc.
			fac.Popularity.Add(0.5)
			fac.Leader.Popularity.Add(0.2)

			// Add history event.
			m.History.AddEvent("faction", facActionString("[FACTION] held a public event to increase popularity.", fac, ""), fac.Ref())
		case fac.Leader.Traits.HasTrait(geneticshuman.TraitCruel):
			// Intimidation
			// Execute a prisoner or a criminal for entertainment.
			// The impact on the popularity should depend on the culture of the people.
			fac.Popularity.Add(0.1)
			fac.Leader.Popularity.Add(0.1)

			// Add history event.
			m.History.AddEvent("faction", facActionString("[FACTION] executed a prisoner in a display of power.", fac, ""), fac.Ref())
		case fac.Leader.Traits.HasTrait(geneticshuman.TraitDeceptive):
			// Manipulation
			// Spread rumors about the other factions, or the leadership.
			// This scheme might backfire, and the popularity might decrease if it is discovered.
			if fac.Leader.Traits.HasTrait(geneticshuman.TraitCareless) {
				fac.Popularity.Add(-0.1)
				fac.Leader.Popularity.Add(-0.1)

				// Add history event.
				m.History.AddEvent("faction", facActionString("[FACTION] spread rumors about the other factions, but it backfired.", fac, ""), fac.Ref())
			} else {
				fac.Popularity.Add(0.1)
				fac.Leader.Popularity.Add(0.1)

				// Add history event.
				m.History.AddEvent("faction", facActionString("[FACTION] spread rumors about the other factions.", fac, ""), fac.Ref())
			}
		default:
			// Random action
			fac.Popularity.Add(0.1)
			fac.Leader.Popularity.Add(0.1)

			// Add history event.
			m.History.AddEvent("faction", facActionString("[FACTION] had a monument built out of spoons.", fac, ""), fac.Ref())
		}
	},
}}
