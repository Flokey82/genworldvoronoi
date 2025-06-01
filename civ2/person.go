package civ2

import civ "github.com/Flokey82/genworldvoronoi/civ/tmp"

type Gender uint8

const (
	GenderMale Gender = iota
	GenderFemale
)

var personID int

func nextPersonID() int {
	personID++
	return personID
}

type Person struct {
	ID        int
	FirstName string
	LastName  string
	Birth     *civ.LifeEvent
	Death     *civ.LifeEvent
	Gender    Gender
}

func NewPerson(firstName, lastName string, day, year, region int) *Person {
	return &Person{
		ID:        nextPersonID(),
		FirstName: firstName,
		LastName:  lastName,
		Birth:     &civ.LifeEvent{Day: day, Year: year, Region: region},
	}
}

func (p Person) GetID() int {
	return p.ID
}
