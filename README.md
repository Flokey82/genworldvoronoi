# genworldvoronoi: Graph-based Planetary Map Generator

`genworldvoronoi` is a sophisticated world generation engine written in Go. It uses Voronoi diagrams and graph-based simulations to create detailed planetary maps with realistic geography, climate, and civilization history.

This project is inspired by [Red Blob Games' planet generation](https://www.redblobgames.com/x/1843-planet-generation/), [SimpleHydrology](https://github.com/weigert/SimpleHydrology), and [mewo2's terrain](https://github.com/mewo2/terrain).

## 🌍 Features

### 🏔️ Geology & Topography
- **Tectonic Plates**: Simulates plate movement, subduction, and mountain building.
- **Erosion**: Implements both hydraulic and thermal erosion for realistic terrain carving.
- **Hydrology**: Realistic river systems, lakes, and drainage basins.
- **Volcanism**: Placement of volcanoes based on tectonic activity.

### 🌤️ Climate & Atmosphere
- **Global Winds**: Simulates prevailing winds and their effect on moisture distribution.
- **Temperature & Precipitation**: Calculates seasonal variations based on latitude, elevation, and distance from the ocean.
- **Biomes**: Sophisticated biome determination based on Whittaker diagrams (Rainforest, Tundra, Desert, etc.).

### 👥 Civilization Simulation (Civ2)
The `civ2` package provides a deep simulation of historical development with two core togglable modes:
- **Cradle of Civilization Mode**: The world starts completely empty (with only nomadic tribes). Civilizations, borders, and empires naturally emerge and expand geographically region-by-region proportional to population levels and geography.
- **Legacy Seeding Mode**: Skips organic initial progression; statically pre-seeds the world with cities, cultures, and empires, flood-filling borders instantaneously over the entire map.

Additional simulated systems:
- **Settlements**: Founding, growth, and abandonment of cities and villages.
- **Societies**: Tribes evolving into city-states and powerful empires.
- **Diplomacy & War**: Dynamic relations between entities, including war declarations and peace treaties.
- **Trade & Economy**: Resource production, harvesting, and trade routes.
- **History Logging**: Comprehensive logging of world events to generate a "written history" of the world.

### 📦 Export & Visualization
- **Formats**: PNG, SVG, Wavefront OBJ (3D), WebP, and GeoJSON.
- **Interactive Apps**: Includes an Ebiten-based map viewer and a Leaflet-based tile server.

---

## 🚀 Getting Started

### Prerequisites
- [Go 1.21+](https://golang.org/dl/)
- For the interactive viewer: [Ebiten dependencies](https://ebitengine.org/en/documents/install.html) (C compiler and graphics drivers).

### Installation
```bash
git clone https://github.com/Flokey82/genworldvoronoi.git
cd genworldvoronoi
go mod download
```

---

## 🛠️ Running the Tools

### 1. Map Exporter (CLI)
Generates a world and exports it to several formats (test.png, test.svg, test.obj, etc.).
```bash
go run cmd/runner.go
```

### 2. Interactive Map Viewer
A real-time interactive viewer with different visualization modes.
```bash
go run cmd2/runner.go
```

### 3. Leaflet Tile Server
Serves the generated map as a web-based tile service (accessible at `http://localhost:3333`).
```bash
cd cmd/server
go run .
```
*Note: Requires `static` and `static_cesium` directories to be present.*

---

## 🖼️ Screenshots

### SVG Export
![SVG Export](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/svg.png)

### Political Maps
![Political Maps](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/political.png)

### Biomes & Climate
![Biomes](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/climate.png)

### 3D Export (Blender)
![OBJ Export](https://raw.githubusercontent.com/Flokey82/genworldvoronoi/master/images/obj.png)

---

## 📜 Dev Notes & TODO
Many of the original goals have been implemented in the `civ2` package, including:
- [x] Concurrency improvements
- [x] Separation of Geology/Climate/Biology/Civilization layers
- [x] Advanced Diplomacy and War mechanics
- [x] Basic Trade and Resource systems

Current focus areas:
- Improving performance for high-resolution maps (400k+ points).
- Enhancing seasonal variation logic.
- Refining trade route optimization.
