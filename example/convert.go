package main

import (
	"fmt"
	"log"
	"os"

	"image"

	_ "github.com/davehouse/go-targa"

	"image/png"
)

func main() {
	var e error

	filename := "temp.tga"

	f, err := os.Open(filename)
	if e != nil {
		panic(e)
	}

	m, _, err := image.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	//fmt.Printf("read image %d,%d", m.Width, m.Height)
	bounds := m.Bounds()

	// Calculate a 16-bin histogram for m's red, green, blue and alpha components.
	//
	// An image's bounds do not necessarily start at (0, 0), so the two loops start
	// at bounds.Min.Y and bounds.Min.X. Looping over Y first and X second is more
	// likely to result in better memory access patterns than X first and Y second.
	var histogram [16][4]int
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			p := m.At(x, y)
			r, g, b, a := p.RGBA()
			//fmt.Print(p)
			// A color's RGBA method returns values in the range [0, 65535].
			// Shifting by 12 reduces this to the range [0, 15].
			histogram[r>>12][0]++
			histogram[g>>12][1]++
			histogram[b>>12][2]++
			histogram[a>>12][3]++
		}
	}

	// Print the results.
	fmt.Printf("%-14s %6s %6s %6s %6s\n", "bin", "red", "green", "blue", "alpha")
	for i, x := range histogram {
		fmt.Printf("0x%04x-0x%04x: %6d %6d %6d %6d\n", i<<12, (i+1)<<12-1, x[0], x[1], x[2], x[3])
	}

	w, err := os.Create("output.png")
	err = png.Encode(w, m)
	if err != nil {
		log.Fatal(err)
	}
	w.Close()

	f.Close()
	fmt.Println("")
}
