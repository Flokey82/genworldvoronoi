package civ2

import (
	"github.com/Flokey82/genetics/geneticshuman"
)

func getLivingRelatives(p *Person) []*Person {
	var rels []*Person
	for _, c := range p.Children {
		if c.Dead() {
			continue
		}
		rels = append(rels, c)
	}
	if p.Spouse != nil && !p.Spouse.Dead() {
		rels = append(rels, p.Spouse)
	}
	seen := make(map[*Person]bool)
	if p.Mother != nil {
		if !p.Mother.Dead() {
			rels = append(rels, p.Mother)
		}
		// Add mother's children.
		for _, c := range p.Mother.Children {
			if c == p || seen[c] || c.Dead() {
				continue
			}
			seen[c] = true
			rels = append(rels, c)
		}
	}
	if p.Father != nil && !p.Father.Dead() {
		rels = append(rels, p.Father)
		// Add father's children.
		for _, c := range p.Father.Children {
			if c == p || seen[c] || c.Dead() {
				continue
			}
			seen[c] = true
			rels = append(rels, c)
		}
	}
	return rels
}

type relation struct {
	p                *Person
	relationToPerson string
	relationOfPerson string
}

func getSiblingRelationString(p *Person) string {
	switch p.Gender() {
	case geneticshuman.GenderFemale:
		return "sister"
	case geneticshuman.GenderMale:
		return "brother"
	default:
		return "sibling"
	}
}
func getParentRelationString(p *Person) string {
	switch p.Gender() {
	case geneticshuman.GenderFemale:
		return "mother"
	case geneticshuman.GenderMale:
		return "father"
	default:
		return "parent"
	}
}
func getChildRelationString(p *Person) string {
	switch p.Gender() {
	case geneticshuman.GenderFemale:
		return "daughter"
	case geneticshuman.GenderMale:
		return "son"
	default:
		return "child"
	}
}
func getSpouseRelationString(p *Person) string {
	switch p.Gender() {
	case geneticshuman.GenderFemale:
		return "wife"
	case geneticshuman.GenderMale:
		return "husband"
	default:
		return "spouse"
	}
}
func getAuntUncleRelationString(p *Person) string {
	switch p.Gender() {
	case geneticshuman.GenderFemale:
		return "aunt"
	case geneticshuman.GenderMale:
		return "uncle"
	default:
		return "aunt/uncle"
	}
}
func getNieceNephewRelationString(p *Person) string {
	switch p.Gender() {
	case geneticshuman.GenderFemale:
		return "niece"
	case geneticshuman.GenderMale:
		return "nephew"
	default:
		return "niece/nephew"
	}
}
func getCousinRelationString(p *Person) string {
	return "cousin"
}
func getGrandparentRelationString(p *Person) string {
	switch p.Gender() {
	case geneticshuman.GenderFemale:
		return "grandmother"
	case geneticshuman.GenderMale:
		return "grandfather"
	default:
		return "grandparent"
	}
}
func getGrandchildRelationString(p *Person) string {
	switch p.Gender() {
	case geneticshuman.GenderFemale:
		return "granddaughter"
	case geneticshuman.GenderMale:
		return "grandson"
	default:
		return "grandchild"
	}
}

func pickRelatives(p *Person) []relation {
	options := make([]relation, 0, 10)
	if len(p.Children) > 0 {
		relParent := getParentRelationString(p)
		for _, c := range p.Children {
			if c.Dead() {
				continue
			}
			options = append(options, relation{
				p:                c,
				relationToPerson: relParent,
				relationOfPerson: getChildRelationString(c),
			})
		}
	}
	if p.Spouse != nil && !p.Spouse.Dead() {
		options = append(options, relation{
			p:                p.Spouse,
			relationToPerson: getSpouseRelationString(p),
			relationOfPerson: getSpouseRelationString(p.Spouse),
		})
	}

	seen := make(map[*Person]bool)
	relSibling := getSiblingRelationString(p)
	relChild := getChildRelationString(p)
	getSiblingsAndParent := func(parent *Person) []relation {
		var options []relation
		if !parent.Dead() {
			options = append(options, relation{
				p:                parent,
				relationToPerson: relChild,
				relationOfPerson: getParentRelationString(parent),
			})
		}
		for _, c := range parent.Children {
			if c == p || seen[c] || c.Dead() {
				continue
			}
			seen[c] = true
			relSiblingHere := relSibling
			siblingRel := getSiblingRelationString(c)
			if c.Father != p.Father && c.Mother != p.Mother {
				relSiblingHere = "step-" + relSibling
				siblingRel = "step-" + siblingRel
			} else if c.Father != p.Father || c.Mother != p.Mother {
				relSiblingHere = "half-" + relSibling
				siblingRel = "half-" + siblingRel
			}
			options = append(options, relation{
				p:                c,
				relationToPerson: relSiblingHere,
				relationOfPerson: siblingRel,
			})

			for _, nieceNephew := range c.Children {
				if nieceNephew == p || seen[nieceNephew] || nieceNephew.Dead() {
					continue
				}
				seen[nieceNephew] = true
				options = append(options, relation{
					p:                nieceNephew,
					relationToPerson: getAuntUncleRelationString(p),
					relationOfPerson: getNieceNephewRelationString(nieceNephew),
				})
			}
		}
		return options
	}

	if p.Mother != nil {
		options = append(options, getSiblingsAndParent(p.Mother)...)
	}

	if p.Father != nil {
		options = append(options, getSiblingsAndParent(p.Father)...)
	}

	getCousinsAndAuntsUncles := func(parent *Person) []relation {
		var options []relation

		addAuntsUnclesAndCousins := func(grandparent *Person) {
			if grandparent == nil {
				return
			}
			for _, auntUncle := range grandparent.Children {
				if auntUncle == parent || seen[auntUncle] {
					continue
				}
				seen[auntUncle] = true
				if !auntUncle.Dead() {
					relAuntUncle := getAuntUncleRelationString(auntUncle)
					if auntUncle.Father != parent.Father && auntUncle.Mother != parent.Mother {
						relAuntUncle = "step-" + relAuntUncle
					} else if auntUncle.Father != parent.Father || auntUncle.Mother != parent.Mother {
						relAuntUncle = "half-" + relAuntUncle
					}
					options = append(options, relation{
						p:                auntUncle,
						relationToPerson: getNieceNephewRelationString(p),
						relationOfPerson: relAuntUncle,
					})
				}
				for _, cousin := range auntUncle.Children {
					if cousin == parent || seen[cousin] || cousin.Dead() {
						continue
					}
					seen[cousin] = true
					options = append(options, relation{
						p:                cousin,
						relationToPerson: getCousinRelationString(p),
						relationOfPerson: getCousinRelationString(cousin),
					})
				}
			}
		}

		addAuntsUnclesAndCousins(parent.Mother)
		addAuntsUnclesAndCousins(parent.Father)

		return options
	}

	if p.Mother != nil {
		options = append(options, getCousinsAndAuntsUncles(p.Mother)...)
	}

	if p.Father != nil {
		options = append(options, getCousinsAndAuntsUncles(p.Father)...)
	}

	getGrandparents := func(parent *Person) []relation {
		var options []relation
		if parent.Mother != nil && !parent.Mother.Dead() {
			options = append(options, relation{
				p:                parent.Mother,
				relationToPerson: getGrandchildRelationString(p),
				relationOfPerson: getGrandparentRelationString(parent.Mother),
			})
		}
		if parent.Father != nil && !parent.Father.Dead() {
			options = append(options, relation{
				p:                parent.Father,
				relationToPerson: getGrandchildRelationString(p),
				relationOfPerson: getParentRelationString(parent.Father),
			})
		}
		return options
	}

	if p.Mother != nil {
		options = append(options, getGrandparents(p.Mother)...)
	}

	if p.Father != nil {
		options = append(options, getGrandparents(p.Father)...)
	}

	return options
}

func (p *Person) GetRelationToPerson(other *Person) *relation {
	if p == other {
		return &relation{
			p:                p,
			relationToPerson: "self",
			relationOfPerson: "self",
		}
	}

	if p.Spouse == other {
		return &relation{
			p:                other,
			relationToPerson: getSpouseRelationString(p),
			relationOfPerson: getSpouseRelationString(other),
		}
	}

	if p.Mother == other.Mother && p.Father == other.Father && p.Mother != nil {
		return &relation{
			p:                other,
			relationToPerson: getSiblingRelationString(p),
			relationOfPerson: getSiblingRelationString(other),
		}
	} else if (p.Mother == other.Mother || p.Father == other.Father) && (p.Mother != nil || p.Father != nil) {
		return &relation{
			p:                other,
			relationToPerson: "half-" + getSiblingRelationString(p),
			relationOfPerson: "half-" + getSiblingRelationString(other),
		}
	}

	if p == other.Mother || p == other.Father {
		return &relation{
			p:                other,
			relationToPerson: getParentRelationString(p),
			relationOfPerson: getChildRelationString(other),
		}
	}

	if p.Spouse != nil && (p.Spouse == other.Mother || p.Spouse == other.Father) {
		return &relation{
			p:                other,
			relationToPerson: "step-" + getParentRelationString(p),
			relationOfPerson: "step-" + getChildRelationString(other),
		}
	}

	if p.Mother == other || p.Father == other {
		return &relation{
			p:                other,
			relationToPerson: getChildRelationString(p),
			relationOfPerson: getParentRelationString(other),
		}
	}

	if p.Mother != nil && p.Mother.Spouse == other {
		return &relation{
			p:                other,
			relationToPerson: "step-" + getParentRelationString(p),
			relationOfPerson: "step-" + getParentRelationString(other),
		}
	}

	if p.Father != nil && p.Father.Spouse == other {
		return &relation{
			p:                other,
			relationToPerson: "step-" + getParentRelationString(p),
			relationOfPerson: "step-" + getParentRelationString(other),
		}
	}

	return nil
}

func (p *Person) findCommonAncestor(other *Person, maxDepth int) (distP, distOther int, ca *Person) {
	if p == other {
		return 0, 0, p
	}

	seenAncestor := make(map[*Person]int)

	type ancestor struct {
		p      *Person
		origin *Person
		depth  int
	}

	var queue []*ancestor
	queue = append(queue, &ancestor{p: p, origin: p, depth: 0})
	queue = append(queue, &ancestor{p: other, origin: other, depth: 0})

	for len(queue) > 0 {
		a := queue[0]
		queue = queue[1:]

		if d, ok := seenAncestor[a.p]; ok {
			if a.origin == p {
				distP = a.depth
				distOther = d
			} else {
				distP = d
				distOther = a.depth
			}
			ca = a.p
			return
		}
		seenAncestor[a.p] = a.depth

		if a.depth >= maxDepth {
			continue
		}

		if a.p.Mother != nil {
			queue = append(queue, &ancestor{p: a.p.Mother, origin: a.origin, depth: a.depth + 1})
		}
		if a.p.Father != nil {
			queue = append(queue, &ancestor{p: a.p.Father, origin: a.origin, depth: a.depth + 1})
		}
	}

	return 0, 0, nil
}

func (p *Person) GetRelationString(other *Person) string {
	distP, distOther, ca := p.findCommonAncestor(other, 10)
	if distP == 0 && distOther == 0 {
		if ca != nil {
			return "self"
		}
		return "unrelated"
	}
	if distP == 0 {
		switch distOther {
		case 1:
			return getChildRelationString(p)
		case 2:
			return "grand" + getChildRelationString(p)
		case 3:
			return "great-grand" + getChildRelationString(p)
		}
		return "descendent"
	}

	if distOther == 0 {
		switch distP {
		case 1:
			return getParentRelationString(p)
		case 2:
			return getGrandparentRelationString(p)
		case 3:
			return "great-" + getGrandparentRelationString(p)
		}
		return "ancestor"
	}

	if distP == 1 {
		if distOther == 1 {
			if (p.Father == other.Father) != (p.Mother == other.Mother) {
				return "half-" + getSiblingRelationString(p)
			}
			return getSiblingRelationString(p)
		}
		if distOther == 2 {
			return getNieceNephewRelationString(p)
		}
		return "relative"
	}

	if distP == 2 {
		if distOther == 1 {
			return getAuntUncleRelationString(p)
		}
		if distOther == 2 {
			return getCousinRelationString(p)
		}
		return "relative"
	}

	return "relative"
}

func (m *Civ) CalcRelationshipDistance(a, b *Person) int {
	if a == b {
		return 0
	}

	queue := NewPersonQueue()
	seen := make(map[*Person]bool)
	distance := make(map[*Person]int)
	seen[a] = true
	distance[a] = 0
	queue.PushBack(a)

	for queue.Len() > 0 {
		p := queue.PopFront()
		if p == b {
			return distance[p]
		}

		if distance[p] > 10 {
			continue
		}

		var related []*Person
		if p.Mother != nil {
			related = append(related, p.Mother)
		}
		if p.Father != nil {
			related = append(related, p.Father)
		}
		if p.Spouse != nil {
			related = append(related, p.Spouse)
		}
		related = append(related, p.Children...)

		for _, r := range related {
			if !seen[r] {
				seen[r] = true
				distance[r] = distance[p] + 1
				queue.PushBack(r)
			}
		}
	}

	return -1
}
