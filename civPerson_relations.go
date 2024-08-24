package genworldvoronoi

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

func pickRelatives(p *Person) []relation {
	// TODO: Allow for more distant relatives.
	var options []relation
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
		if parent.Mother != nil {
			// Add mother's siblings.
			for _, auntUncle := range parent.Mother.Children {
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
				// Add mother's siblings' children (cousins).
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
		if parent.Father != nil {
			// Add mother's siblings.
			for _, auntUncle := range parent.Father.Children {
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
				// Add mother's siblings' children (cousins).
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
		if parent.Mother != nil {
			if !parent.Mother.Dead() {
				options = append(options, relation{
					p:                parent.Mother,
					relationToPerson: getGrandchildRelationString(p),
					relationOfPerson: getGrandparentRelationString(parent.Mother),
				})
			}
		}
		if parent.Father != nil {
			if !parent.Father.Dead() {
				options = append(options, relation{
					p:                parent.Father,
					relationToPerson: getGrandchildRelationString(p),
					relationOfPerson: getParentRelationString(parent.Father),
				})
			}
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
