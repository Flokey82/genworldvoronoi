# Genworldvoronoi CLI Generator & Exporter

This is the primary command-line tool for generating `genworldvoronoi` maps directly via the terminal. It evaluates all the internal biome, tectonic, and political simulations and directly exports the resultant planet to standard image or model formats (PNG, SVG, WebP, OBJ). 

## 🚀 How to Run

To run the generation and export process, navigate to this directory and use:

```bash
go run .
```

Alternatively, you could run it from the root directory via:

```bash
go run cmd/runner.go
```

By default, the program evaluates the world against default thresholds and generates the following files in the runtime directory:
- `test.png`
- `test.svg`
- `test.webp`
- `test.obj`

## ⚙️ Configuration Flags

The CLI exposes nearly all fundamental Geography and Civilization toggles as runtime command-line flags. Key flags include:

- `--seed <int>`: Control the map generation seed.
- `--num_cities <int>`: Determine the initial spawn population of cities.
- `--num_empires <int>`: Set the number of resulting empires.
- `--seed_entities <bool>`: Toggle the simulation's initial settlement methodology (Legacy Seeding vs Cradle of Civilization).
- `--enable_city_aging <bool>`: Allow settlements to naturally grow and transition into larger states organically.
- `--cpuprofile <path>` & `--memprofile <path>`: For profiling internal bottleneck performances.
