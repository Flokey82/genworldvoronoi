package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"image"
	"image/png"
	"log"
	"net/http"
	"strconv"

	"github.com/Flokey82/genworldvoronoi"
	"github.com/gorilla/mux"
)

var worldmap *genworldvoronoi.Map

var (
	seed                    int64   = 12345
	numPlates               int     = 25
	oceanPlatesFraction     float64 = 0.65
	oceanPlatesAltSelection bool    = true
	numPoints               int     = 400000
	numVolcanoes            int     = 10
	jitter                  float64 = 0.0
	useGlobe                bool    = false
)

func init() {
	flag.Int64Var(&seed, "seed", seed, "the world seed")
	flag.IntVar(&numPlates, "num_plates", numPlates, "number of plates")
	flag.IntVar(&numPoints, "num_points", numPoints, "number of points")
	flag.IntVar(&numVolcanoes, "num_volcanoes", numVolcanoes, "number of volcanoes")
	flag.BoolVar(&useGlobe, "use_globe", useGlobe, "use 3D globe")
	flag.Float64Var(&jitter, "jitter", jitter, "jitter")
}

func main() {
	flag.Parse()

	// Initialize the config.
	cfg := genworldvoronoi.NewConfig()
	cfg.GeoConfig.NumPlates = numPlates
	cfg.GeoConfig.OceanPlatesFraction = oceanPlatesFraction
	cfg.GeoConfig.OceanPlatesAltSelection = oceanPlatesAltSelection
	cfg.GeoConfig.NumPoints = numPoints
	cfg.GeoConfig.NumVolcanoes = numVolcanoes
	cfg.GeoConfig.Jitter = jitter

	// Initialize the planet.
	sp, err := genworldvoronoi.NewMapFromConfig(seed, cfg)
	if err != nil {
		log.Fatal(err)
	}
	worldmap = sp

	// Start the server.
	router := mux.NewRouter()
	router.HandleFunc("/tiles/{z}/{x}/{y}", tileHandler)
	router.HandleFunc("/terrain/{z}/{x}/{y}", tileHeightMapHandler)
	router.HandleFunc("/terrain3d/layer.json", tile3dJSONHandler)
	router.HandleFunc("/terrain3d/{z}/{x}/{y}.terrain", tile3dHandler)
	router.HandleFunc("/geojson_cities/{z}/{la1}/{lo1}/{la2}/{lo2}", geoJSONCitiesHandler)
	router.HandleFunc("/geojson_borders/{z}/{la1}/{lo1}/{la2}/{lo2}", geoJSONBorderHandler)
	if useGlobe {
		router.PathPrefix("/").Handler(http.FileServer(http.Dir("static_cesium")))
	} else {
		router.PathPrefix("/").Handler(http.FileServer(http.Dir("static")))
	}
	log.Fatal(http.ListenAndServe(":3333", router))
}

func parseBoundingBox(req *http.Request) (la1, lo1, la2, lo2 float64, z int, err error) {
	// Get the tile coordinates and zoom level.
	vars := mux.Vars(req)
	la1, err = strconv.ParseFloat(vars["la1"], 64)
	if err != nil {
		return
	}
	la2, err = strconv.ParseFloat(vars["la2"], 64)
	if err != nil {
		return
	}
	lo1, err = strconv.ParseFloat(vars["lo1"], 64)
	if err != nil {
		return
	}
	lo2, err = strconv.ParseFloat(vars["lo2"], 64)
	if err != nil {
		return
	}
	z, err = strconv.Atoi(vars["z"])
	if err != nil {
		return
	}
	return
}

func geoJSONCitiesHandler(res http.ResponseWriter, req *http.Request) {
	// Get the tile coordinates and zoom level.
	tileLa1, tileLo1, tileLa2, tileLo2, tileZ, err := parseBoundingBox(req)
	if err != nil {
		panic(err)
	}
	data, err := worldmap.GetGeoJSONCities(tileLa1, tileLo1, tileLa2, tileLo2, tileZ)
	if err != nil {
		panic(err)
	}
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Length", strconv.Itoa(len(data)))
	res.Write(data)
}

func geoJSONBorderHandler(res http.ResponseWriter, req *http.Request) {
	// Get the url parameter 'd'.
	d := req.URL.Query().Get("d")
	if d == "" {
		d = "0"
	}
	displayMode, err := strconv.Atoi(d)
	if err != nil {
		panic(err)
	}

	// Get the tile coordinates and zoom level.
	tileLa1, tileLo1, tileLa2, tileLo2, tileZ, err := parseBoundingBox(req)
	if err != nil {
		panic(err)
	}
	data, err := worldmap.GetGeoJSONBorders(tileLa1, tileLo1, tileLa2, tileLo2, tileZ, displayMode)
	if err != nil {
		panic(err)
	}
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Content-Length", strconv.Itoa(len(data)))
	res.Write(data)
}

func tileHandler(res http.ResponseWriter, req *http.Request) {
	// Get the url parameter 'd'.
	d := req.URL.Query().Get("d")
	if d == "" {
		d = "0"
	}
	displayMode, err := strconv.Atoi(d)
	if err != nil {
		panic(err)
	}

	// get the url parameter 'vectors'.
	vectors := req.URL.Query().Get("vectors")
	if vectors == "" {
		vectors = "0"
	}
	vectorMode, err := strconv.Atoi(vectors)
	if err != nil {
		panic(err)
	}

	// get the url parameter 'rivers'.
	rivers := req.URL.Query().Get("rivers")
	if rivers == "" {
		rivers = "false"
	}

	// get the url parameter 'trade'.
	trade := req.URL.Query().Get("trade")
	if trade == "" {
		trade = "false"
	}

	// get the url parameter 'lakes'.
	lakes := req.URL.Query().Get("lakes")
	if lakes == "" {
		lakes = "false"
	}

	// get the url parameter 'shadows'.
	shadows := req.URL.Query().Get("shadows")
	if shadows == "" {
		shadows = "false"
	}

	// get the url parameter 'aspectshadows'.
	aspectshadows := req.URL.Query().Get("aspectshadows")
	if aspectshadows == "" {
		aspectshadows = "false"
	}

	// Get the tile coordinates and zoom level.
	vars := mux.Vars(req)
	tileX, err := strconv.Atoi(vars["x"])
	if err != nil {
		panic(err)
	}
	tileY, err := strconv.Atoi(vars["y"])
	if err != nil {
		panic(err)
	}
	tileZ, err := strconv.Atoi(vars["z"])
	if err != nil {
		panic(err)
	}

	// Get the tile image.
	img := worldmap.GetTile(tileX, tileY, tileZ, displayMode, vectorMode, rivers == "true", trade == "true", lakes == "true", shadows == "true", aspectshadows == "true")
	writeImage(res, &img)
}

// writeImage writes the image to the response writer.
func writeImage(w http.ResponseWriter, img *image.Image) {
	buffer := new(bytes.Buffer)
	if err := png.Encode(buffer, *img); err != nil {
		log.Println("unable to encode image.")
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(buffer.Bytes())))
	if _, err := w.Write(buffer.Bytes()); err != nil {
		log.Println("unable to write image.")
	}
}

const layerJson = `{
	"tilejson": "2.1.0",
	"name": "MDT_Navarra_2014_5m-epsg25830",
	"description": "",
	"version": "1.1.0",
	"format": "quantized-mesh-1.0",
	"attribution": "",
	"schema": "tms",
	"tiles": [ "{z}/{x}/{y}.terrain?v={version}" ],
	"minzoom": 1,
	"maxzoom": 16,
	"projection": "EPSG:3857",
	"bounds": [ -180.00, -90.00, 0.00, 90.00 ],
    "available" : [
        [
            {
                "startX" : 0,
                "startY" : 0,
                "endX" : 1,
                "endY" : 0
            }
        ],
        [
            {
                "startX" : 0,
                "startY" : 0,
                "endX" : 3,
                "endY" : 1
            }
        ]
    ]
  }`

func tile3dJSONHandler(res http.ResponseWriter, req *http.Request) {
	buf := bytes.NewBuffer([]byte(layerJson))
	// GZIP the data.
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(buf.Bytes()); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}

	// Set the headers and write the data.
	data := b.Bytes()
	res.Header().Set("Content-Type", "application/octet-stream")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Content-Length", strconv.Itoa(len(data)))
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	res.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	res.Write(data)
}

func tile3dHandler(res http.ResponseWriter, req *http.Request) {
	// Get the tile coordinates and zoom level.
	vars := mux.Vars(req)
	tileX, err := strconv.Atoi(vars["x"])
	if err != nil {
		panic(err)
	}
	tileY, err := strconv.Atoi(vars["y"])
	if err != nil {
		panic(err)
	}
	tileZ, err := strconv.Atoi(vars["z"])
	if err != nil {
		panic(err)
	}

	// Get the tile image.
	t3d := worldmap.Get3DTile(tileX, tileY, tileZ)
	buf := bytes.NewBuffer(nil)
	if err := t3d.Write(buf); err != nil {
		panic(err)
	}
	// GZIP the data.
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(buf.Bytes()); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}

	// Set the headers and write the data.
	data := b.Bytes()
	res.Header().Set("Content-Type", "application/octet-stream")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Content-Length", strconv.Itoa(len(data)))
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	res.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	res.Write(data)
}

func tileHeightMapHandler(res http.ResponseWriter, req *http.Request) {
	// Get the tile coordinates and zoom level.
	vars := mux.Vars(req)
	tileX, err := strconv.Atoi(vars["x"])
	if err != nil {
		panic(err)
	}
	tileY, err := strconv.Atoi(vars["y"])
	if err != nil {
		panic(err)
	}
	tileZ, err := strconv.Atoi(vars["z"])
	if err != nil {
		panic(err)
	}

	// Get the tile image.
	dat := worldmap.GetHeightMapTile(tileX, tileY, tileZ)

	// GZIP the data.
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write(dat); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}

	// Set the headers and write the data.
	data := b.Bytes()
	res.Header().Set("Content-Type", "application/octet-stream")
	res.Header().Set("Content-Encoding", "gzip")
	res.Header().Set("Content-Length", strconv.Itoa(len(data)))
	res.Header().Set("Access-Control-Allow-Origin", "*")
	res.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
	res.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	res.Write(data)
}
