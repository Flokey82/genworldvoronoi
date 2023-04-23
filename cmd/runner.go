package main

import (
	"flag"
	"log"
	"os"
	"runtime/pprof"

	"github.com/Flokey82/genworldvoronoi"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to this file")

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	sp, err := genworldvoronoi.NewMap(1234, 25, 200000, 0.0)
	if err != nil {
		log.Fatal(err)
	}

	sp.GetEmpires()
	exportPNG := true
	exportOBJ := true
	exportSVG := true
	exportWebp := true
	if exportPNG {
		sp.ExportPng("test.png")
	}
	if exportOBJ {
		sp.ExportOBJ("test.obj")
	}
	if exportSVG {
		sp.ExportSVG("test.svg")
	}
	if exportWebp {
		sp.ExportWebp("test.webp")
	}

	if *memprofile != "" {
		f, err := os.Create(*memprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		return
	}
}
