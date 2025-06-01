package civ2

import "github.com/Flokey82/genworldvoronoi/geo"

type ResourceAmount struct {
	Resource *geo.Resource
	Amount   int
}

var professionID int

// TODO: General availability of the professiom should be determined by a culture's skills.
// Regional availability can be determined by a function that checks the region's resources.
// There should be different levels of proficiency for each profession, depending on the
// available tools, resources, and skills. For example, basic hunting can be done with
// a pointy stick, rocks, or bare hands, but advanced hunting requires bows, traps, and
// other tools. The latter will return more resources and be more efficient than the former.
// The same goes for other professions.

type Profession struct {
	ID          int
	Name        string
	IsAvailable func(r int, m *Civ) bool // Is the profession available in the region?
	Consumes    []ResourceAmount         // Resources consumed by the profession in a tick.
	Produces    []ResourceAmount         // Resources produced by the profession in a tick.
}

func nextProfessionID() int {
	professionID++
	return professionID
}

const ResourceTypeFood geo.ResourceType = geo.ResourceTypeMax + 1

var ResFoodMeat = &geo.Resource{
	Name:  "Meat",
	Type:  ResourceTypeFood,
	Value: 1,
}

var ResFoodBerries = &geo.Resource{
	Name:  "Berries",
	Type:  ResourceTypeFood,
	Value: 1,
}

var (
	ProfessionHunter = &Profession{
		ID:   nextProfessionID(),
		Name: "Hunter",
		IsAvailable: func(r int, m *Civ) bool {
			return true
		},
		Consumes: nil,
		Produces: []ResourceAmount{{
			Resource: ResFoodMeat,
			Amount:   1,
		}},
	}
	ProfessionGatherer = &Profession{
		ID:   nextProfessionID(),
		Name: "Gatherer",
		IsAvailable: func(r int, m *Civ) bool {
			return true
		},
		Consumes: nil,
		Produces: []ResourceAmount{{
			Resource: ResFoodBerries,
			Amount:   1,
		}},
	}
)

func (m *Civ) calculateResourceRequirements(population int) {
	// Now we need a function that calculates the resource consumption of a population.
	// NOTE: The required and desired resources differ based on a culture, and how
	// developed it is. For example, a hunter-gatherer culture will be happy with simple
	// meat, berries, and water, while a settled culture will desire luxury goods,
	// like high quality fabrics, spices, and jewelry.
	// If the population is not satisfied with the resources, they will be unhappy and
	// may revolt, migrate, or die out if the most basic needs are not met.

	// For now we will assume that the population consumes 1 unit of food per day per person.
	// NOTE: This doesn't take into account the
}
