# Leaflet Server

To run the server, simply build the binary and run it.

```bash
go build main.go
./main
```

The server will serve http on port 3333 by default, so you can access it via http://localhost:3333.

Enjoy! :)

## UI elements

D: Display mode (will cycle through various visualizations)

B: Borders

W: Wind (and other) vectors

R: Rivers

S: Shadows (shaded relief)

A: Enable aspect shading for shadows (slightly better contrast)

## webglearth

If you want to see the generated world on a globe, use the following command:

```bash
go build main.go
./main --use_globe=true
```

This will render the map on a simple globe. We'll switch to a different library in the future, but it is a nice proof of concept for now.