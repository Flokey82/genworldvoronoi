# genworldvoronoi: Graph based planetary map generator

It simulates (somewhat) global winds and attempts to calculate precipitation and temperature for more intricate simulations in the future.
It features SVG, PNG, and Wavefront OBJ output.

This is based on https://www.redblobgames.com/x/1843-planet-generation/ and a port of https://github.com/redblobgames/1843-planet-generation to Go. 

I draw further inspiration from various other projects such as https://github.com/weigert/SimpleHydrology and https://github.com/mewo2/terrain

... if you haven't noticed yet, this is a placeholder for when I feel less lazy and add more information :D

# Dev notes

This thing needs a use, and I think the major drawback right now is the time that it takes to generate a reasonably complex planet with enough detail to be interesting. So here are tne following important points that need to be addressed:

* Generation speed
  * Use concurrency where sensible
* Export / import of generated data
  * Separate generation steps into
    * Geology, Climate
    * Biology, Species, Survivability
    * Civilization, Cities, Empires
  * Binary format for writing to / reading from disk
* Simulation
  * Seasons, Weather, Disasters
  * Biology, Species, Migrations
  * Population Growth, Migrations
  * Wars, Diplomacy
  * Founding, Development, Abandonment, Fall of Cities, Empires, Religions
  * Written History, Legends
  
## TODO

* Use cached temperature instead of getRegTemperature every single time
* Climate
  * Add desert oases that are fed from underground aquifers. Look at these examples: https://www.google.com/maps/d/viewer?mid=1BvY10l3yzWt48IwCXqDcyeuawpA&hl=en&ll=26.715853962142784%2C28.408963168787885&z=6
  * Climate seems too wet at times (too many wetlands?)
  * Seasonal forests should not be at the equator, where there are no seasons.
* Elevation
  * Move away from linear interpolation
  * Add improved noise
* Winds
  * Push temperature around [DONE]
  * Push dry air around, not just humid air
  * Re-evaluate rainfall and moisture distribution
* Civilization
  * Industry and trade
    * Introduce industry
    * Introduce production / harvesting of goods
    * Improve trade routes
  * Cities
    * Better city fitness functions
    * Separate world generation better from everything else
    * Assign goods and resources to cities (for trade)
  * Empires
    * Introduce empires with capitals [DONE]
    * Provide simpler means to query information on an empire
  * Cultures
    * Add fitness function for "natural" population density estimates
* Resources
  * Improve resource distribution
    * Fitness functions
    * Amount / quality / discoverability?
  * Add more resource types
* Species
  * Allow for overlapping populations
  * Allow for species migration

Here some old pictures what it does...

## SVG export with rivers, capital city placement and stuff.
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/svg.png "Screenshot of SVG!")

## Leaflet server (and sad flavor text).
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/leaflet.png "Flavortext Maps!")

## Poor man's relief map.
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/relief.png "Relief Maps!")

## Slightly wealthier man's relief map.
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/relief_2.png "Relief Maps!")

## Does political maps.
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/political.png "Political Maps!")

## Simulates climate (-ish)
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/climate.png "Screenshot of Biomes!")

## Simulates seasons (-ish)
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/seasons.webp "Screenshot of Seasons!")

## Exports to Wavefront OBJ
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/obj.png "Screenshot of OBJ Export in Blender!")

## Webglearth sample
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/webglobe.png "Screenshot of Webglearth!")

## Cesium sample
![alt text](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/cesium.png "Screenshot of Cesium!")
