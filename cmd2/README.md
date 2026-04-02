# Genworldvoronoi Interactive Map Viewer (Ebiten)

This application is an experimental, interactive desktop map viewer built using the robust Ebitengine package (`ebiten/v2`). Unlike the web-based map exporters (Leaflet or Cesium) which serve pre-generated static tiles via an HTTP server, this viewer functions as a natively responsive desktop application that processes the planet's data internally and renders to a window directly.

## 🚀 How to Run

Execute from the current directory:
```bash
go run .
```

Alternatively, you could run it directly from the project root:
```bash
go run cmd2/runner.go
```

## ⚙️ Configuration Flags

The standalone engine mirrors the standard configuration parameters from the simulation API, meaning you control what exactly constitutes the newly generated world when you invoke the command.

- `--seed <int>`: Default seed override. 
- `--num_cities <int>`: Starting city allocations.
- `--seed_entities <bool>`: Toggle the usage of Legacy flood-fill civilization scaling versus Cradle of Civilization organic territorial growth mechanics.
- `--jitter <float>`: Perturb generation parameters.

## ⌨️ Controls

While interacting directly with the native viewer, use standard window keybindings to aggressively query the world state or control simulation layers and pacing over time:

- `W`, `A`, `S`, `D` / `Up`, `Left`, `Down`, `Right`: Pan the map view/camera gracefully.
- `E` / `PageUp`: Zoom Out.
- `C` / `PageDown`: Zoom In.
- `M`: Cycles to toggle the Display Layers (Elevation, Topography, Political Borders, Empires, etc).
- `L`: Dumps local nomad tribe demographics entirely to the terminal output.
- `T`: Tick the simulation forward in low increments (allows time flow visualization).
- `G`: Fast-forward tick the simulation rapidly 100 times.
- `K`: Extremely fast-forward tick the simulation 2000 times (spits out thousands of years of growth).
- `O`: Toggles the City/Settlement indicator overlay rendering.
- `R`: Toggles River lines overlay specifically.
- `B`: Toggles Culture markers.
- `U` & `I`: Shift the visual time of day (Insolation lighting adjustments forward and backwards respectively).
- `P`: Highlights suitability indices across the global terrain layout.
