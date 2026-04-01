# Leaflet & Cesium Tile Server

This server provides a web-based interface to explore the generated world using Leaflet (2D) or Cesium (3D Globe).

## 🚀 How to Run

To start the server, simply use `go run .` from this directory:

```bash
go run .
```

The server will be available at `http://localhost:3333`.

### 🌍 3D Globe Mode (Cesium)
If you want to see the generated world on a 3D globe, use the `--use_globe` flag:

```bash
go run . --use_globe=true
```

## ⌨️ UI Controls

When viewing the map in the browser, you can use the following keys to toggle visualizations:

- **D**: Cycle through Display modes (Biomes, Height, Temperature, etc.)
- **B**: Toggle Borders
- **W**: Toggle Wind and other vectors
- **R**: Toggle Rivers
- **S**: Toggle Shadows (shaded relief)
- **A**: Toggle Aspect shading (improved contrast for shadows)

## 📋 Prerequisites
- Ensure the `static` (for Leaflet) and `static_cesium` (for Cesium) directories are present in this folder as they contain the web assets.
