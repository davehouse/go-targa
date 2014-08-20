package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type Header struct {
	Idlength        uint8
	Colourmaptype   uint8
	Datatypecode    uint8
	Colourmaporigin uint16
	Colourmaplength uint16
	Colourmapdepth  uint8
	X_origin        uint16
	Y_origin        uint16
	Width           uint16
	Height          uint16
	Bitsperpixel    uint8
	Imagedescriptor uint8
}

var DataTypeStrings = map[uint8]string{
	0:  "No image data included.",
	1:  "Uncompressed, color-mapped images.",
	2:  "Uncompressed, RGB images.",
	3:  "Uncompressed, black and white images.",
	9:  "Runlength encoded color-mapped images.",
	10: "Runlength encoded RGB images.",
	11: "Compressed, black and white images.",
	32: "Compressed color-mapped data, using Huffman, Delta, and runlength encoding.",
	33: "Compressed color-mapped data, using Huffman, Delta, and runlength encoding. 4-pass quadtree-type process.",
}

// Reading files requires checking most calls for errors.
// This helper will streamline our error checks below.
func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	filename := "temp.tga"

	f, err := os.Open(filename)
	check(err)

	//data := make([]byte, 18)
	var head Header
	err = binary.Read(f, binary.LittleEndian, &head)
	check(err)

	fmt.Printf("\nIdlength  %d", head.Idlength)
	fmt.Printf("\nColourmaptype  %d", head.Colourmaptype)
	fmt.Printf("\nDatatypecode  %d %s", head.Datatypecode, DataTypeStrings[head.Datatypecode])
	fmt.Printf("\n\tColourmaporigin  %d", head.Colourmaporigin)
	fmt.Printf("\n\tColourmaplength  %d", head.Colourmaplength)
	fmt.Printf("\n\tColourmapdepth  %d", head.Colourmapdepth)
	fmt.Printf("\nX_origin  %d", head.X_origin)
	fmt.Printf("\nY_origin  %d", head.Y_origin)
	fmt.Printf("\nWidth  %d", head.Width)
	fmt.Printf("\nHeight  %d", head.Height)
	fmt.Printf("\nBitsperpixel  %d", head.Bitsperpixel)
	fmt.Printf("\nImagedescriptor  %d [0 standard]", head.Imagedescriptor)
	var bpps [4]uint8
	bpps[0] = head.Imagedescriptor & 1
	bpps[1] = head.Imagedescriptor & (1 << 1)
	bpps[2] = head.Imagedescriptor & (1 << 2)
	bpps[3] = head.Imagedescriptor & (1 << 3)
	fmt.Printf("\n\tbits per pixel [%d,%d,%d,%d]", bpps[0], bpps[1], bpps[2], bpps[3])
	fmt.Printf("\n\treserved %d [must be 0]", head.Imagedescriptor&(1<<4))
	fmt.Printf("\n\tscreen origin %d [0: lower left, 1: upper left]", head.Imagedescriptor&(1<<5))
	fmt.Printf("\n\tinterleaving %d, %d",
		head.Imagedescriptor&(1<<6),
		head.Imagedescriptor&(1<<7))

	w := head.Width
	h := head.Height
	bpp := uint16(head.Bitsperpixel)

	// Read image identification field
	if head.Idlength > 0 {
		iddata := make([]byte, head.Idlength)
		err = binary.Read(f, binary.LittleEndian, &iddata)
		check(err)
		fmt.Print("Id data:", iddata)
	}

	// Read Color map data
	if head.Colourmaptype > 0 {
	}

	//dataIndex := 18 + head.Idlength + head.Colourmaptype
	fmt.Printf("\n%d,%d * %d\n", w, h, bpp)
	r := bufio.NewReader(f)
	buf := make([]byte, 1024)
	tick := 0
	pixels := 0
	maxPixels := int(w * h)
	for {
		if pixels >= maxPixels {
			break
		}
		n, e := r.Read(buf)
		if e != nil && e != io.EOF {
			panic(e)
		}
		if n == 0 {
			break
		}
		tick++
		fmt.Printf("%d ", tick)
		i := 0
		for {
			if pixels >= maxPixels {
				break
			}
			p := buf[i]
			switch head.Datatypecode {
			case 10:
				i++
				end := int(p&(0x7f)) + 1
				pixels += end
				if p>>7&1 == 1 {
					b := buf[i]
					i++
					g := buf[i]
					i++
					r := buf[i]
					i++
					a := buf[i]
					i++
					//for j := 0; j < end; j++ {
					fmt.Printf("-%d[%d]:%d,%d,%d,%d", i, end, r, g, b, a)
					//}
				} else {
					end = end + i
					for ; i < end; i++ {
						b := buf[i]
						i++
						g := buf[i]
						i++
						r := buf[i]
						i++
						a := buf[i]
						i++
						fmt.Printf(" %d[%d]:%d,%d,%d,%d", i, end, r, g, b, a)
					}
				}
			default:
				for i, p := range buf[:n] {
					fmt.Printf(" %d:%d", i, p)
				}
			}
		}
	}

	fmt.Printf("\nBitsperpixel  %d", head.Bitsperpixel)
	fmt.Printf("\ndata")

	//box := img.Bounds()
	//fmt.Println(box.Min.X, box.Min.Y)

	// Close the file when you're done (usually this would
	// be scheduled immediately after `Open`ing with
	// `defer`).
	f.Close()
	fmt.Println("")
}
