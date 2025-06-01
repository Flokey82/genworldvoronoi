package civ

import (
	"testing"
)

func TestPersonFindCommonAncestor(t *testing.T) {
	newPerson := func(name string, parent *Person) *Person {
		p := &Person{
			FirstName: name,
			Father:    parent,
		}
		if parent != nil {
			parent.Children = append(parent.Children, p)
		}
		return p
	}

	greatgreatgrandparent := newPerson("greatgreatgrandparent", nil)

	greatgrandparent := newPerson("greatgrandparent", greatgreatgrandparent)

	grandparent := newPerson("grandparent", greatgrandparent)

	parent := newPerson("parent", grandparent)

	self := newPerson("self", parent)

	child := newPerson("child", self)

	grandchild := newPerson("grandchild", child)

	// Next closest.

	uncle := newPerson("uncle", grandparent)

	firstCousin := newPerson("cousin", uncle)

	firstCousinOnceRemoved := newPerson("cousinOnceRemoved", firstCousin)

	firstCousinTwiceRemoved := newPerson("cousinTwiceRemoved", firstCousinOnceRemoved)

	// Next closest.

	greatUncle := newPerson("grandUncle", greatgrandparent)

	firstCousinOnceRemoved2 := newPerson("firstCousinOnceRemoved", greatUncle)

	secondCousin := newPerson("secondCousin", firstCousinOnceRemoved2)

	secondCousinOnceRemoved := newPerson("secondCousinOnceRemoved", secondCousin)

	secondCousinTwiceRemoved := newPerson("secondCousinTwiceRemoved", secondCousinOnceRemoved)

	// Next closest.

	greatgreatUncle := newPerson("greatgreatUncle", greatgreatgrandparent)

	firstCousinTwiceRemoved3 := newPerson("firstCousinTwiceRemoved", greatgreatUncle)

	secondCousinOnceRemoved3 := newPerson("secondCousinTwiceRemoved", firstCousinTwiceRemoved3)

	thirdCousin := newPerson("thirdCousin", secondCousinOnceRemoved3)

	thirdCousinOnceRemoved := newPerson("thirdCousinOnceRemoved", thirdCousin)

	thirdCousinTwiceRemoved := newPerson("thirdCousinTwiceRemoved", thirdCousinOnceRemoved)

	// Test cases.
	tests := []struct {
		other              *Person
		wantDistanceP      int
		wantDistanceOther  int
		wantCommonAncestor *Person
	}{{
		other:              greatgreatgrandparent,
		wantDistanceP:      4,
		wantDistanceOther:  0,
		wantCommonAncestor: greatgreatgrandparent,
	}, {
		other:              greatgrandparent,
		wantDistanceP:      3,
		wantDistanceOther:  0,
		wantCommonAncestor: greatgrandparent,
	}, {
		other:              grandparent,
		wantDistanceP:      2,
		wantDistanceOther:  0,
		wantCommonAncestor: grandparent,
	}, {
		other:              parent,
		wantDistanceP:      1,
		wantDistanceOther:  0,
		wantCommonAncestor: parent,
	}, {
		other:              self,
		wantDistanceP:      0,
		wantDistanceOther:  0,
		wantCommonAncestor: self,
	}, {
		other:              child,
		wantDistanceP:      0,
		wantDistanceOther:  1,
		wantCommonAncestor: self,
	}, {
		other:              grandchild,
		wantDistanceP:      0,
		wantDistanceOther:  2,
		wantCommonAncestor: self,
	}, {
		other:              uncle,
		wantDistanceP:      2,
		wantDistanceOther:  1,
		wantCommonAncestor: grandparent,
	}, {
		other:              firstCousin,
		wantDistanceP:      2,
		wantDistanceOther:  2,
		wantCommonAncestor: grandparent,
	}, {
		other:              firstCousinOnceRemoved,
		wantDistanceP:      2,
		wantDistanceOther:  3,
		wantCommonAncestor: grandparent,
	}, {
		other:              firstCousinTwiceRemoved,
		wantDistanceP:      2,
		wantDistanceOther:  4,
		wantCommonAncestor: grandparent,
	}, {
		other:              greatUncle,
		wantDistanceP:      3,
		wantDistanceOther:  1,
		wantCommonAncestor: greatgrandparent,
	}, {
		other:              firstCousinOnceRemoved2,
		wantDistanceP:      3,
		wantDistanceOther:  2,
		wantCommonAncestor: greatgrandparent,
	}, {
		other:              secondCousin,
		wantDistanceP:      3,
		wantDistanceOther:  3,
		wantCommonAncestor: greatgrandparent,
	}, {
		other:              secondCousinOnceRemoved,
		wantDistanceP:      3,
		wantDistanceOther:  4,
		wantCommonAncestor: greatgrandparent,
	}, {
		other:              secondCousinTwiceRemoved,
		wantDistanceP:      3,
		wantDistanceOther:  5,
		wantCommonAncestor: greatgrandparent,
	}, {
		other:              greatgreatUncle,
		wantDistanceP:      4,
		wantDistanceOther:  1,
		wantCommonAncestor: greatgreatgrandparent,
	}, {
		other:              firstCousinTwiceRemoved3,
		wantDistanceP:      4,
		wantDistanceOther:  2,
		wantCommonAncestor: greatgreatgrandparent,
	}, {
		other:              secondCousinOnceRemoved3,
		wantDistanceP:      4,
		wantDistanceOther:  3,
		wantCommonAncestor: greatgreatgrandparent,
	}, {
		other:              thirdCousin,
		wantDistanceP:      4,
		wantDistanceOther:  4,
		wantCommonAncestor: greatgreatgrandparent,
	}, {
		other:              thirdCousinOnceRemoved,
		wantDistanceP:      4,
		wantDistanceOther:  5,
		wantCommonAncestor: greatgreatgrandparent,
	}, {
		other:              thirdCousinTwiceRemoved,
		wantDistanceP:      4,
		wantDistanceOther:  6,
		wantCommonAncestor: greatgreatgrandparent,
	}}

	for _, test := range tests {
		gotDistanceP, gotDistanceOther, gotCommonAncestor := self.findCommonAncestor(test.other, 10)
		if gotDistanceP != test.wantDistanceP || gotDistanceOther != test.wantDistanceOther || gotCommonAncestor != test.wantCommonAncestor {
			t.Errorf("findCommonAncestor(%v) = %v, %v, %v; want %v, %v, %v", test.other.FirstName, gotDistanceP, gotDistanceOther, gotCommonAncestor.FirstName, test.wantDistanceP, test.wantDistanceOther, test.wantCommonAncestor)
		}

		// Test to see if we can print the correct string.
		gotString := self.GetRelationString(test.other)
		t.Logf("Relation string for %q: %v", test.other.FirstName, gotString)
	}
}
