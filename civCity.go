package genworldvoronoi

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"

	"github.com/Flokey82/go_gens/gameconstants"
	"github.com/Flokey82/go_gens/genlanguage"
	"github.com/Flokey82/go_gens/utils"
)

func (m *Civ) TickCity(c *City, gDisFunc func(int) GeoDisasterChance, cf func(int) *Culture) {
	m.resetRand()
	m.tickCityDays(c, gDisFunc, cf, 1)
}

func (m *Civ) tickCityDays(c *City, gDisFunc func(int) GeoDisasterChance, cf func(int) *Culture, days int) {
	// Check if the city is abandoned.
	if c.Population <= 0 {
		if c.Population < 0 {
			c.Population = 0
		}
		return
	}

	// Check if a random disaster strikes.
	if m.rand.Intn(100*356) < days {
		m.tickCityDisaster(c, gDisFunc, days)
	}

	// In the middle ages, the average population growth was 0.16%-ish per year.
	// See: https://en.wikipedia.org/wiki/Medieval_demography
	//
	// TODO:
	// - The population growth should be dependent on the economic power
	//   and if there is famine, war, drought, sickness, etc.
	// - Also take in account what size of population the city can sustain.
	//
	// TODO: Compare the actual population with the population we calculate
	// here and kill off people if the actual population is larger.

	// Calculate the new population.

	// This variant uses the logistic growth function, taking into account
	// the carrying capacity of the city.
	maxPop := float64(c.MaxPopulationLimit())
	curPop := float64(c.Population)
	newPop := (curPop * maxPop) / (curPop + (maxPop-curPop)*math.Pow(math.E, -c.PopulationGrowthRate()*float64(days)/365))
	c.Population = int(math.Ceil(newPop))

	// This variant uses the exponential growth function.
	// newPop := float64(c.Population) * math.Pow(math.E, c.PopulationGrowthRate()*float64(days)/365)
	// c.Population = int(math.Ceil(newPop))

	// Calculate the limit of the population based on attractiveness and
	// economic potential and see if we have exceeded the limit of what
	// the city can sustain.
	if maxPop := c.MaxPopulationLimit(); c.Population > maxPop {
		log.Println("City population limit reached:", c.Name, c.Population, maxPop)
		log.Printf("Attractiveness: %.2f, Economic Potential: %.2f, Agriculture: %.2f", c.Attractiveness, c.EconomicPotential, c.Agriculture)

		// The excess population can migrate to other cities or a new
		// settlement might be founded nearby.
		//
		// Since we don't want to constantly migrate people, we just
		// move a larger portion of entire population, so that we drop
		// way below the limit, giving us a chance to grow again for
		// a while.
		excessPopulation := c.Population - maxPop

		// Move 10% of the population or 1.2 times the excess population,
		// whichever is larger.
		excessPopulation = int(math.Max(
			float64(excessPopulation)*m.MigrationOverpopulationExcessPopulationFactor,
			float64(c.Population)*m.MigrationOverpopulationMinPopulationFactor,
		))

		// Make sure we don't move more than the entire population.
		excessPopulation = utils.Min(excessPopulation, c.Population)

		m.relocateFromCity(c, excessPopulation)
	}

	// TODO:
	// - If a city reaches a certain size it might transition from an
	//   agricultural town to a city with a more diverse economy. A mining town
	//   might, if there is enough resources, transition to an industrial city.
	// - Update the economic potential of the city if the population has changed.
	//   This can be pretty expensive, which we should keep in mind.
	//   m.calculateEconomicPotential()

	// Update the peak population.
	//
	// TODO: Maybe store the year when the peak population was reached?
	if c.Population > c.MaxPopulation {
		c.MaxPopulation = c.Population
	}

	// If there is no religion originating from the city, then there is a
	// chance that a new religion might be founded.
	//
	// TODO: Maybe keep note of an inciting event, like a famine, war, etc.
	if c.Religion == nil && m.rand.Intn(3000*356) < days && c.Population > 0 {
		c.Religion = m.genOrganizedReligion(c)
		m.ExpandReligions()
		m.History.AddEvent("Religion", fmt.Sprintf("A new religion was founded in %s", c.Name), c.Ref())
	}
}

func (m *Civ) getCityDisasters(c *City, gDisFunc func(int) GeoDisasterChance) []disaster {
	if c.Population == 0 {
		return nil // No disasters for deserted cities.
	}
	// For example, a mining town should have a chance of a cave in,
	// or a farming town should have a chance of a drought.
	// Towns close to volcanoes should have a chance of a volcanic
	// eruption while fault lines should have a chance of an earthquake.

	// Get the disasters that may occur in this city, depending on the
	// type and size of the city.
	//
	// TODO:
	// - If there is a coal mine, coal mine fires should be possible.
	// - Add other industry specific disasters.
	var ds []disaster
	switch c.Type {
	case TownTypeQuarry, TownTypeMining, TownTypeMiningGems:
		ds = append(ds, disRockslide, disCaveIn)
	case TownTypeDesertOasis:
		ds = append(ds, disSandstorm)
	}
	ds = append(ds, disDrought, disFamine)
	// With increasing population, the city is be more prone to famine
	// or disease.
	//
	// TODO: Improve this with some metrics like population density,
	// sanitation, etc.
	if c.Population > 1000 {
		ds = append(ds, disDisease)
	}
	if c.Population > 10000 {
		ds = append(ds, disPlague)
	}
	// Append the region specific disasters and return.
	return append(ds, gDisFunc(c.ID).getDisasters()...)
}

func (m *Civ) tickCityDisaster(c *City, gDisFunc func(int) GeoDisasterChance, days int) {
	// There is a chance of some form of disaster.
	// If towns are heavily affected, they might be destroyed or abandoned.
	//
	// TODO:
	// - Also the life expectancy might be low in early history.
	// - Wars might also have a negative effect on the population.

	// Enable / disable migration of population when a disaster strikes.
	enableDisasterMigration := true

	// Pick a random disaster given their respective probabilities.
	cityDisasters := m.getCityDisasters(c, gDisFunc)
	dis := randDisaster(cityDisasters)
	if dis == disNone {
		log.Fatalf("No disaster was chosen")
	}

	// Calculate the population loss.
	popLoss := dis.PopulationLoss * (2 + rand.Float64()) / 3
	dead := int(math.Ceil(float64(c.Population) * popLoss))

	// HACK: Kill the people that died in the disaster.
	// c.People = m.killNPeople(c.People, dead)

	year := m.Geo.Calendar.GetYear()

	// Add an event to the calendar.
	m.AddEvent(dis.Name, fmt.Sprintf("%s died", numPeopleStr(dead)), c.Ref())

	// Reduce the population.
	c.Population -= dead
	if c.Population <= 0 {
		c.Population = 0
		return
	}

	// Log the disaster, what type, how many people died and where.
	log.Printf("Year %d: %s, %s died in %s", year, dis.Name, numPeopleStr(dead), c.Name)

	// Since there was a disaster, depending on the number of people that
	// died, some people might leave the city.
	//
	// If there is sickness, war, famine, drought, etc, the population might
	// migrate to other cities that are more prosperous or a new settlement
	// might be founded nearby.
	//
	// The bigger the population loss, the more likely it is that people
	// will leave the city.
	if enableDisasterMigration && rand.Float64() < popLoss {
		// Up to 'popLoss' of the population might leave the city.
		leave := int(float64(c.Population) * (popLoss * rand.Float64()))
		m.relocateFromCity(c, leave)
	}
}

// relocateFromCity moves a portion of the population from the city to
// another city or a new settlement.
//
// TODO: Distribute more evenly if a large group of people are moving.
func (m *Civ) relocateFromCity(c *City, population int) {
	// If no one is migrating, then there is nothing to do.
	if population <= 0 {
		return
	}

	// Move out the migrating population.
	if c.Population < population {
		population = c.Population
	}
	c.Population -= population

	// Add an event to the calendar.
	m.AddEvent("Migration", fmt.Sprintf("%s left", numPeopleStr(population)), c.Ref())

	// Calculate the analog of distance between regions by taking the surface
	// of a sphere with radius 1 and dividing it by the number of regions.
	// The square root will work as a somewhat sensible approximation of distance.
	distRegion := math.Sqrt(4 * math.Pi / float64(m.SphereMesh.numRegions))

	// Per distRegion traversed, there is a defined chance of death.
	calcChanceDeath := func(dist float64) float64 {
		return 1 - math.Pow(1-m.MigrationFatalityChance, dist/distRegion)
	}

	// Get the existing cities as potential destinations.
	cities := m.getExistingCities()

	// Sort the cities by increasing distance from the city.
	sort.Slice(cities, func(i, j int) bool {
		return m.GetDistance(c.ID, cities[i].ID) < m.GetDistance(c.ID, cities[j].ID)
	})

	// The closest city is the city itself, so skip it.
	// Check if any of the n closest cities have enough space.
	for _, city := range cities[1:utils.Min(len(cities), m.MigrationToNClosestCities+1)] {
		maxPop := city.MaxPopulationLimit()
		popCapacity := maxPop - city.Population

		// If there is capacity, a portion of the population might move there.
		if popCapacity > 0 {
			// Now pick a fraction of the population that will move to the city,
			// with the largest fraction going to the closest city.
			numMigrants := utils.Min(population, popCapacity/2)

			// Make sure we don't increase the population by more than 20%,
			// except if the city is abandoned.
			if city.Population > 0 {
				numMigrants = utils.Min(numMigrants, city.Population/5)
			}

			// Depending on the distance, some of the population might die on the way.
			dist := m.GetDistance(c.ID, city.ID)
			dead := int(math.Ceil(calcChanceDeath(dist) * float64(numMigrants)))

			// HACK: Kill the people that died on the way.
			// c.People = m.killNPeople2(c.People, dead)

			// If any survived, move them to the city.
			if survived := numMigrants - dead; survived > 0 {
				// The rest of the population survives and migrates to the new city.
				// m.moveNFromToCity(c, city, survived)

				// If the city is abandoned, set the economic potential to 1 temporarily.
				if city.Population == 0 {
					city.EconomicPotential = 1
				}

				// Move the population to the closest city.
				city.Population += survived
				if city.Population > city.MaxPopulation {
					city.MaxPopulation = city.Population
				}

				// TODO: Update the economic potential of the city.
				m.AddEvent("Migration", fmt.Sprintf("%s arrived", numPeopleStr(survived)), city.Ref())
			}
			log.Printf("%s moved from %s to %s, %d died on the way", numPeopleStr(numMigrants), c.Name, city.Name, dead)

			// Subtract the number of people that moved from the total
			// population that is migrating.
			population -= numMigrants
			if population <= 0 {
				return
			}
		}
	}

	// Make sure we don't place a new settlement in a region that is already
	// occupied or will be occupied in the future.
	//
	// NOTE: We have already determined some cities that will be settled in
	// the future, so we need to take that into account.
	occupied := make(map[int]bool)
	for _, city := range m.Cities {
		occupied[city.ID] = true
	}

	// Since the closest city doesn't have enough space, we need to
	// create a new settlement.
	attFunc := m.getAttractivenessFunc()

	// Find the best suitable neighbor region up to a certain depth.
	bestReg := -1    // most suitable region so far
	bestScore := 0.0 // attractiveness of the most suitable region so far

	// Keep track of the regions that we have already seen.
	seenRegions := make(map[int]bool)

	// Traverse the neighbors of the current region.
	out_r := make([]int, 0, 8)
	var traverseNeighbors func(out_r []int, id int, depth int)
	traverseNeighbors = func(out_r []int, id int, depth int) {
		if depth >= m.MigrationToNewSettlementWithinNRegions { // maximum depth to traverse
			return
		}
		// Instantiate new re-usable slices for the sequential recursive call in the children.
		out_rc := make([]int, 0, 8)

		// Circulate through the neighbors of the current region using the out_r slice
		// to avoid allocating a new slice for each recursive call from the parent.
		for _, nb := range m.r_circulate_r(out_r, id) {
			if seenRegions[nb] {
				continue
			}
			seenRegions[nb] = true
			attr := attFunc(nb)
			if attr > bestScore && !occupied[nb] && nb != c.ID {
				bestScore = attr
				bestReg = nb
			}
			traverseNeighbors(out_rc, nb, depth+1)
		}
	}
	traverseNeighbors(out_r, c.ID, 0)

	// If we didn't find a suitable region, panic.
	// TODO: Handle this case better.
	if bestReg == -1 {
		panic("no suitable location found")
	}

	// If we found a suitable region, create a new city there.
	// Depending on the distance, some of the population might die on the way.
	dist := m.GetDistance(c.ID, bestReg)
	dead := int(math.Ceil(calcChanceDeath(dist) * float64(population)))

	// HACK: Kill the people that died on the way.
	// c.People = m.killNPeople2(c.People, dead)

	// Check if any survived and founded a new city.
	if survived := population - dead; survived > 0 {
		city := m.placeCityAt(bestReg, m.getRegCityType(bestReg), survived, bestScore)
		city.Founded = m.History.GetYear() + 1 // The city is founded next year.
		city.EconomicPotential = 1             // Set the economic potential to 1 temporarily.
		city.Attractiveness = bestScore        // Set the attractiveness to the best score.

		// The rest of the population survives and migrates to the new city.
		// m.moveNFromToCity(c, city, survived)

		// TODO: Set the economic potential and attractiveness of the new city.
		log.Printf("%s moved from %s and founded %s, %d died on the way", numPeopleStr(population), c.Name, city.Name, dead)
	}
}

// City represents a city in the world.
type City struct {
	ID                int                   // Region where the city is located
	Name              string                // Name of the city
	Type              TownType              // Type of city
	Score             float64               // Score of the fitness function
	Population        int                   // Population of the city
	MaxPopulation     int                   // Maximum population of the city
	Culture           *Culture              // Culture of the city region
	Language          *genlanguage.Language // Language of the city
	Religion          *Religion             // Religion originating from the city
	Founded           int64                 // Year when the city was founded
	EconomicPotential float64               // Economic potential of the city (DYNAMIC)
	Trade             float64               // Trade value of the city (DYNAMIC)
	Resources         float64               // Resources value of the city (PARTLY DYNAMIC)
	Agriculture       float64               // Agriculture value of the city (STATIC)
	Attractiveness    float64               // Attractiveness of the city (STATIC)
	TradePartners     int                   // Number of cities within trade range
	People            []*Person             // People living in the city
}

// Ref returns the object reference of the city.
func (c *City) Ref() ObjectReference {
	return ObjectReference{
		ID:   c.ID,
		Type: ObjectTypeCity,
	}
}

func (c *City) radius() float64 {
	// In kilometers.
	if c.Population <= 0 {
		return 0
	}
	return 100 * math.Sqrt(float64(c.Population)/math.Pi) / gameconstants.EarthCircumference
}

// String returns a string representation of the city.
func (c *City) String() string {
	return fmt.Sprintf("%s (%d)", c.Name, c.Population)
}

// MaxPopulationLimit returns the maximum population sustainable by the city.
func (c *City) MaxPopulationLimit() int {
	return 200 + int(20000*math.Pow((c.EconomicPotential+c.Attractiveness), 2))
}

// PopulationGrowthRate returns the population growth rate per year.
func (c *City) PopulationGrowthRate() float64 {
	return 0.0005 + 0.0025*(c.EconomicPotential+c.Attractiveness)/2
}

// PlaceNCities places n cities with the highest fitness scores.
func (m *Civ) PlaceNCities(n int, cType TownType) {
	// The fitness function, returning a score from 0.0 to 1.0 for a given region.
	// Select the fitness function based on the city type.
	scoreFunc := cType.GetFitnessFunction(m)

	// The distance seed point function, returning seed points/regions that we
	// want to be far away from.
	// For now we just maximize the distance to cities of the same type.
	distSeedFunc := cType.GetDistanceSeedFunc(m)

	// Get the stop regions, i.e. regions that we don't want to place cities in.
	stopRegions := make(map[int]bool)

	// Place n cities of the given type.
	regDistanceC := m.assignDistanceField(distSeedFunc(), stopRegions)
	for i := 0; i < n; i++ {
		// Place a city at the region with the highest fitness score.
		c := m.placeCityWithScore(cType, m.CalcCityScoreWithDistanceField(scoreFunc, regDistanceC))
		log.Printf("placing %s city %d: %s", cType, i, c.String())

		// Update the distance field.
		regDistanceC = m.UpdateDistanceField(regDistanceC, distSeedFunc(), stopRegions)
	}
}

// PlaceCity places another city at the region with the highest fitness score.
func (m *Civ) PlaceCity(cType TownType, scoreFunc func(int) float64, distSeedFunc func() []int) *City {
	return m.placeCityWithScore(cType, m.CalcCityScore(scoreFunc, distSeedFunc))
}

func (m *Civ) placeCityWithScore(cType TownType, cityScore []float64) *City {
	// Pick the region with the highest fitness score.
	occupied := make(map[int]bool)
	for _, c := range m.Cities {
		occupied[c.ID] = true
	}

	// Find the best location based on the fitness function.
	newcity := -1
	lastMax := math.Inf(-1)
	for i, val := range cityScore {
		if val > lastMax && !occupied[i] {
			newcity = i
			lastMax = val
		}
	}

	// If no suitable location was found, panic.
	// TODO: Handle this case better.
	if newcity == -1 {
		panic("no suitable location found")
	}

	// Get base population from city type.
	// TODO: Calculate population based on suitability for habitation.
	basePop := cType.FoundingPopulation()
	basePop += 2 * m.rand.Intn(basePop) / (len(m.Cities) + 1)
	return m.placeCityAt(newcity, cType, basePop, lastMax)
}

func (m *Civ) placeCityAt(r int, cType TownType, pop int, score float64) *City {
	// TODO:
	// - Trigger event for city founding.
	// - Allow optionally specifying a founding year.
	// - Set agricultural potential and resources based on the region.
	// - Add the population to the city.
	c := &City{
		ID:            r,
		Score:         score,
		Population:    pop,
		MaxPopulation: pop,
		Type:          cType,
		Culture:       m.GetCulture(r),
		Founded:       m.Settled[r] + m.rand.Int63n(100),
	}

	// If there is no known culture, generate a new one.
	// TODO: Grow this culture.
	if c.Culture == nil {
		c.Culture = m.PlaceCultureAt(r)
	}

	// Use the local language to generate a new city name.
	c.Language = c.Culture.Language
	c.Name = c.Language.MakeCityName()
	m.Cities = append(m.Cities, c)
	return c
}

/*
// Not used yet

	func (m *Civ) tickPeopleAtCity(c *City, nDays int, cf func(int) *Culture) {
		c.People = m.tickPeople(c.People, nDays, cf)
	}

*/
// Not used yet
/*
func (m *Civ) addNToCity(c *City, n int, cf func(int) *Culture) {
	// Generate a number of people and update the city population.
	localPop := m.placePopulationAt(c.ID, n, cf)
	for _, p := range localPop {
		p.City = c
	}

	// Add people to city.
	c.People = append(c.People, localPop...)
}

func (m *Civ) moveNFromToCity(from, to *City, n int) {
	fromAfter := make([]*Person, 0, len(from.People))
	toAfter := make([]*Person, 0, n)
	var migrated int
	for _, i := range rand.Perm(len(from.People)) {
		p := from.People[i]
		if migrated >= n {
			fromAfter = append(fromAfter, p)
			continue
		}

		// TODO: Also migrate spouses and children.
		if !p.isDead() {
			p.Region = to.ID
			p.City = to
			migrated++
			toAfter = append(toAfter, p)
		}
	}
	from.People = fromAfter
	to.People = append(to.People, toAfter...)
}
*/
