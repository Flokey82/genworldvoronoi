package genworldvoronoi

type Opinions struct {
	Opinions map[*Person]*Opinion
}

func NewOpinions() *Opinions {
	return &Opinions{
		Opinions: make(map[*Person]*Opinion),
	}
}

type Opinion struct {
	Opinion float64
	BasedOn []*Event
}

func (o *Opinions) AddOpinion(obj *Person, opinion float64, basedOn *Event) {
	if _, ok := o.Opinions[obj]; !ok {
		o.Opinions[obj] = &Opinion{}
	}
	o.Opinions[obj].BasedOn = append(o.Opinions[obj].BasedOn, basedOn)
	// Calculate running average
	o.Opinions[obj].Opinion = (o.Opinions[obj].Opinion + opinion) / 2
}

func (o *Opinions) GetOpinion(obj *Person) float64 {
	if _, ok := o.Opinions[obj]; !ok {
		return 0
	}
	return o.Opinions[obj].Opinion
}

func (o *Opinions) GetNemesis() *Person {
	const minOpinion = -0.4
	var nemesis *Person
	nemesisOpinion := minOpinion
	// TODO: DO NOT RANGE OVER THE MAP DIRECTLY. USE A SLICE INSTEAD.
	for p, op := range o.Opinions {
		if p.Death.IsSet() {
			continue
		}
		if op.Opinion < nemesisOpinion {
			nemesis = p
			nemesisOpinion = op.Opinion
		}
	}
	return nemesis
}
