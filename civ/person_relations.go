package civ

import "github.com/Flokey82/genetics/geneticshuman"

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

// TOOD: Introduce a traversal function that can be used to traverse the family tree up
// to a given depth.

func pickRelatives(p *Person) []relation {
	// TODO: Allow for more distant relatives.
	options := make([]relation, 0, 10)
	relChild := getChildRelationString(p)
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

	// Our relationship to our siblings. (brother, sister, sibling)
	seen := make(map[*Person]bool)
	relSibling := getSiblingRelationString(p)
	getSiblingsAndParent := func(parent *Person) []relation {
		var options []relation
		if !parent.Dead() {
			options = append(options, relation{
				p:                parent,
				relationToPerson: relChild,
				relationOfPerson: getParentRelationString(parent),
			})
		}
		// Add parent's children.
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

			// TODO: Add siblings' children (niece/nephew).
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

	// Add mother and children.
	if p.Mother != nil {
		options = append(options, getSiblingsAndParent(p.Mother)...)
	}

	// Add father and children.
	if p.Father != nil {
		options = append(options, getSiblingsAndParent(p.Father)...)
	}

	// Get aunts/uncles and cousins, nieces/nephews.
	getCousinsAndAuntsUncles := func(parent *Person) []relation {
		var options []relation

		addAuntsUnclesAndCousins := func(grandparent *Person) {
			if grandparent == nil {
				return
			}
			// Add parent's siblings (grandparent's children).
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
				// Add parent's siblings' children (cousins).
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

		// Add children of the grandparents (aunts/uncles and their children (cousins)).
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

	// TODO: Get grandparents.
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

	// TODO: Get Nieces/Nephews.

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

	// Our relationship to our siblings. (brother, sister, sibling)
	if p.Mother == other.Mother && p.Father == other.Father {
		return &relation{
			p:                other,
			relationToPerson: getSiblingRelationString(p),
			relationOfPerson: getSiblingRelationString(other),
		}
	} else if p.Mother == other.Mother || p.Father == other.Father {
		return &relation{
			p:                other,
			relationToPerson: "half-" + getSiblingRelationString(p),
			relationOfPerson: "half-" + getSiblingRelationString(other),
		}
	}

	// Our relationship to our children.
	if p == other.Mother || p == other.Father {
		return &relation{
			p:                other,
			relationToPerson: getParentRelationString(p),
			relationOfPerson: getChildRelationString(other),
		}
	}

	// Our relationship to our spouse's children.
	if p.Spouse != nil && (p.Spouse == other.Mother || p.Spouse == other.Father) {
		return &relation{
			p:                other,
			relationToPerson: "step-" + getParentRelationString(p),
			relationOfPerson: "step-" + getChildRelationString(other),
		}
	}

	// Our relationship to our parents.
	if p.Mother == other || p.Father == other {
		return &relation{
			p:                other,
			relationToPerson: getChildRelationString(p),
			relationOfPerson: getParentRelationString(other),
		}
	}

	// Our relationship to possible step-parents.
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

	// TODO: Implement more distant relatives.

	return nil
}

func (p *Person) GetGrandparents() []*Person {
	gps := make([]*Person, 0, 4)
	getGrans := func(parent *Person) {
		if parent == nil {
			return
		}
		if parent.Mother != nil {
			gps = append(gps, parent.Mother)
		}
		if parent.Father != nil {
			gps = append(gps, parent.Father)
		}
	}
	getGrans(p.Mother)
	getGrans(p.Father)
	return gps
}

func (p *Person) GetAuntsUncles() []*Person {
	auntsUncles := make([]*Person, 0, 10)
	seen := make(map[*Person]bool)
	seen[p.Mother] = true
	seen[p.Father] = true
	seen[p] = true
	getAuntsUncles := func(grandParent *Person) {
		if grandParent == nil {
			return
		}
		for _, auntUncle := range grandParent.Children {
			if seen[auntUncle] {
				continue
			}
			seen[auntUncle] = true
			auntsUncles = append(auntsUncles, auntUncle)
		}
	}
	for _, gp := range p.GetGrandparents() {
		getAuntsUncles(gp)
	}
	return auntsUncles
}

// TODO: Adapt this for "common family", so in-laws, etc.
func (p *Person) findCommonAncestor(other *Person, maxDepth int) (distP, distOther int, ca *Person) {
	if p == other {
		return 0, 0, p
	}

	// Find the common ancestor of both persons.
	seenAncestor := make(map[*Person]int)

	// Use a breadth-first search to find the common ancestor of both.
	type ancestor struct {
		p      *Person // The ancestor.
		origin *Person // The person that found the ancestor.
		depth  int     // The depth of the ancestor as seen from the origin.
	}

	// The queue of ancestors to check.
	var queue []*ancestor
	queue = append(queue, &ancestor{p: p, origin: p, depth: 0})
	queue = append(queue, &ancestor{p: other, origin: other, depth: 0})

	for len(queue) > 0 {
		// Pop the first element from the queue.
		a := queue[0]
		queue = queue[1:]

		// Check if the ancestor is a common ancestor.
		if _, ok := seenAncestor[a.p]; ok {
			// Since we don't know which person found the ancestor first, we need to check the origin.
			if a.origin == p {
				// Other found it first, so seenAncestor[a.p] is the distance from other to the ancestor.
				distP = a.depth
				distOther = seenAncestor[a.p]
			} else {
				// P found it first, so seenAncestor[a.p] is the distance from p to the ancestor.
				distP = seenAncestor[a.p]
				distOther = a.depth
			}
			ca = a.p
			return
		}
		seenAncestor[a.p] = a.depth

		// Add the parent to the queue.
		if a.p.Mother != nil {
			queue = append(queue, &ancestor{p: a.p.Mother, origin: a.origin, depth: a.depth + 1})
		}
		if a.p.Father != nil {
			queue = append(queue, &ancestor{p: a.p.Father, origin: a.origin, depth: a.depth + 1})
		}
	}

	return 0, 0, nil
}

// GetRelationString returns the relation of other to p as a string.
// TODO: Add in-laws, step-parents/siblings, haf-siblings, etc.
func (p *Person) GetRelationString(other *Person) string {
	distP, distOther, ca := p.findCommonAncestor(other, 1000)
	if distP == 0 && distOther == 0 {
		if ca != nil {
			return "self"
		}
		return "unrelated"
	}
	// Other is a descendent, we are the common ancestor.
	if distP == 0 {
		if distOther == 1 {
			// TODO: Add step-child.
			return getChildRelationString(p) // Child
		}
		if distOther == 2 {
			return "grand" + getChildRelationString(p) // Grandchild
		}
		if distOther == 3 {
			return "great-grand" + getChildRelationString(p) // Great-grandchild
		}
		return "descendent"
	}

	// Other is an ancestor, we are the descendent.
	if distOther == 0 {
		if distP == 1 {
			// TODO: Add step-parenthood.
			return getParentRelationString(p) // Parent
		}
		if distP == 2 {
			return getGrandparentRelationString(p) // Grandparent
		}
		if distP == 3 {
			return "great-" + getGrandparentRelationString(p) // Great-grandparent
		}
		return "ancestor"
	}

	// Other is a potential sibling, niece, nephew, we have the same parent.
	if distP == 1 {
		if distOther == 1 {
			// Half-sibling.
			if (p.Father == other.Father) != (p.Mother == other.Mother) {
				return "half-" + getSiblingRelationString(p)
			}
			return getSiblingRelationString(p)
		}
		if distOther == 2 {
			return getNieceNephewRelationString(p)
		}
		if distOther == 3 {
			return "great-" + getNieceNephewRelationString(p)
		}
		return "relative"
	}

	// Now we have uncles, aunts, cousins, etc.
	if distP == 2 {
		if distOther == 1 {
			return getAuntUncleRelationString(p)
		}
		if distOther == 2 {
			return getCousinRelationString(p)
		}
		if distOther == 3 {
			return getCousinRelationString(p) + " once removed"
		}
		if distOther == 4 {
			return getCousinRelationString(p) + " twice removed"
		}
		return "relative"
	}

	// Now we have great-uncles, great-aunts, etc.
	if distP == 3 {
		if distOther == 1 {
			return "great-" + getAuntUncleRelationString(p)
		}
		if distOther == 2 {
			return getCousinRelationString(p) + " once removed"
		}
		if distOther == 3 {
			return "second " + getCousinRelationString(p)
		}
		if distOther == 4 {
			return "second " + getCousinRelationString(p) + " once removed"
		}
		if distOther == 5 {
			return "second " + getCousinRelationString(p) + " twice removed"
		}
		return "relative"
	}

	// Now we have great-great-uncles, great-great-aunts, etc.
	if distP == 4 {
		if distOther == 1 {
			return "great-great-" + getAuntUncleRelationString(p)
		}
		if distOther == 2 {
			return getCousinRelationString(p) + " twice removed"
		}
		if distOther == 3 {
			return "second " + getCousinRelationString(p) + " once removed"
		}
		if distOther == 4 {
			return "third " + getCousinRelationString(p)
		}
		if distOther == 5 {
			return "third " + getCousinRelationString(p) + " once removed"
		}
		if distOther == 6 {
			return "third " + getCousinRelationString(p) + " twice removed"
		}
		return "relative"
	}

	return "relative"
}

// https://github.com/iand/genster/blob/3300deec3de8b4a9f7a57e3e19338cf9929d5af9/model/relation.go#L215
