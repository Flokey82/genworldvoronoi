package civ

import (
	"sort"
)

type Opinions struct {
	Opinions map[*Person]*Opinion
	People   []*Person
}

func NewOpinions() *Opinions {
	return &Opinions{
		Opinions: make(map[*Person]*Opinion),
		People:   make([]*Person, 0, 10),
	}
}

// Opinion is a struct that holds the opinion of a person about another person.
// The opinion is a float64 between -1 and 1, where -1 is the worst opinion and 1 is the best.
// BasedOn is a slice of events that the opinion is based on.
// TODO:
// - Add a decay function to the opinion.
// - Add a clamping to the opinion value.
type Opinion struct {
	Opinion float64
	BasedOn []*Event
}

func (o *Opinions) AddOpinion(obj *Person, opinion float64, basedOn *Event) {
	op, ok := o.Opinions[obj]
	if ok {
		op.Opinion = (op.Opinion + opinion) / 2
		op.BasedOn = append(checkCapacity(op.BasedOn), basedOn)
		return
	}

	op = &Opinion{
		BasedOn: make([]*Event, 0, 10),
		Opinion: opinion,
	}
	o.Opinions[obj] = op

	// If we've reached 80% of the slice capacity, double the capacity.
	o.People = append(checkCapacity(o.People), obj)

	// Add the event to the slice.
	op.BasedOn = append(op.BasedOn, basedOn)
}

func checkCapacity[T any](slice []T) []T {
	if cap(slice) <= len(slice)*4/5 {
		newSlice := make([]T, len(slice), cap(slice)*2)
		copy(newSlice, slice)
		return newSlice
	}
	return slice
}

func (o *Opinions) GetOpinion(obj *Person) float64 {
	op, ok := o.Opinions[obj]
	if !ok {
		return 0
	}
	return op.Opinion
}

func (o *Opinions) GetNemesis() *Person {
	const minOpinion = -0.4
	var nemesis *Person
	nemesisOpinion := minOpinion
	for _, p := range o.People {
		if p.Death.IsSet() {
			continue
		}
		op := o.GetOpinion(p)
		if op < nemesisOpinion {
			nemesis = p
			nemesisOpinion = op
		}
	}
	return nemesis
}

func (o *Opinions) GetFavorite() *Person {
	const maxOpinion = 0.4
	var favorite *Person
	favoriteOpinion := maxOpinion
	for _, p := range o.People {
		if p.Death.IsSet() {
			continue
		}
		op := o.GetOpinion(p)
		if op > favoriteOpinion {
			favorite = p
			favoriteOpinion = op
		}
	}
	return favorite
}

func (o *Opinions) GetPeopleOpinionDesc() []*Person {
	return o.GetPeopleByOpinion(false)
}

func (o *Opinions) GetPeopleOpinionAsc() []*Person {
	return o.GetPeopleByOpinion(true)
}

func (o *Opinions) GetPeopleByOpinion(asc bool) []*Person {
	people := make([]*Person, len(o.People))
	copy(people, o.People)
	sort.Slice(people, func(i, j int) bool {
		return o.GetOpinion(people[i]) < o.GetOpinion(people[j]) == asc
	})
	return people
}
