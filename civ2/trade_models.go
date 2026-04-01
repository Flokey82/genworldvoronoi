package civ2

// TradeRoute represents an established trade path between two entities.
type TradeRoute struct {
	Source      peopleThing
	Destination peopleThing
	Path        []int // List of region IDs
	Exp         []ResAmount // Resources exported from Source to Destination
	Imp         []ResAmount // Resources imported from Destination to Source
	Active      bool
}
