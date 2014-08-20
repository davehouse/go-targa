package tga

/*
Targa files do not have magic header bytes.

Targa 2.0 has trailing magic tag/bytes of "TRUEVISION-XFILE.[nul]" (ends with
period followed by nul character 0x00). However, the go image parser only looks
at header bytes for magic bytes.

Because of this, Targa is best handled directly like:
	image, err := tga.Decode(reader)

Otherwise, the image decoder will fall-through to using Targa as a last option
because we register it as having no magic header bytes.
	image, _, err := image.Decode(reader)  // tries every other format first

*/

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash"
	"hash/crc32"
	"image"
	"image/color"
	"io"
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

type decoder struct {
	head          Header
	r             io.Reader
	image         image.Image
	crc           hash.Hash32
	flip          bool
	width, height int
	depth         int
	palette       color.Palette
	cb            int
	stage         int
	idatLength    uint32
	tmp           [3 * 256]byte
}

type reader interface {
	io.Reader
	io.ByteReader
}

type FormatError string

func (e FormatError) Error() string { return "tga format error: " + string(e) }

func (d *decoder) parseHeader() error {
	var head Header
	err := binary.Read(d.r, binary.LittleEndian, &head)
	if err != nil {
		return FormatError(fmt.Sprintf("invalid Targa header: %s", err))
	}

	d.width = int(head.Width)
	d.height = int(head.Height)
	d.depth = int(head.Bitsperpixel)
	d.head = head

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
	fmt.Printf("\n%d", head.Imagedescriptor)
	bpps[0] = head.Imagedescriptor & 1
	bpps[1] = head.Imagedescriptor >> 1 & 1
	bpps[2] = head.Imagedescriptor >> 2 & 1
	bpps[3] = head.Imagedescriptor >> 3 & 1
	fmt.Printf("\n\tbits per pixel [%t,%t,%t,%t]", bpps[0], bpps[1], bpps[2], bpps[3])
	fmt.Printf("\n\tleft to right %t [false]", head.Imagedescriptor>>4&1 == 1)
	fmt.Printf("\n\ttop to bottom %t [false]", head.Imagedescriptor>>5&1 == 1)
	fmt.Printf("\n\tinterleaving %d, %d",
		head.Imagedescriptor>>6&1,
		head.Imagedescriptor>>7&1)
	fmt.Printf("\n")
	d.flip = head.Imagedescriptor>>5&1 == 1

	return nil
}

// Decode reads a Targa image from r and returns it as an image.Image.
// The type of Image returned depends on the Targa contents.
func Decode(r io.Reader) (image.Image, error) {
	d := &decoder{
		r:   r,
		crc: crc32.NewIEEE(),
	}
	// Add buffering if r does not provide ReadByte.
	if rr, ok := r.(reader); ok {
		d.r = rr
	} else {
		d.r = bufio.NewReader(r)
	}
	if err := d.parseHeader(); err != nil {
		fmt.Printf("\nparseHeader err")
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return nil, err
	}

	if d.head.Colourmaplength != 0 {
		/*
			m.Palette, err = d.readColorMap()
			if err != nil {
				return err
			}
		*/
	}

	width := int(d.width)
	height := int(d.height)
	//left := int(d.head.X_origin)
	//top := int(d.head.Y_origin)

	bounds := image.Rect(0, 0, width, height)
	nrgba := image.NewNRGBA(bounds)
	d.image = nrgba

	bypp := d.depth / 8
	lineWidth := d.width * bypp
	cr := make([]uint8, lineWidth)
	fmt.Printf("Image %d x %d %b", d.width, d.height, d.flip)
	fmt.Printf("Reading lines of byte length %d [%d * bytesPerPixel[%d]]\n", lineWidth, width, bypp)
	var y int
	var limit int
	var step int
	if d.flip { // flipped [top-left origin]
		y = 0
		limit = height + 1
		step = 1
	} else { // standard [bottom-left origin]
		y = height - 1
		limit = -1
		step = -1
	}
	switch d.head.Datatypecode {
	case 10:
		i := 0
		x := 0
		for {
			fmt.Println("\n", x, y)
			if y == limit {
				fmt.Printf("y==l:%d %d", y, limit)
				break
			}
			if x >= width {
				fmt.Printf("x>=w:%d %d", x, width)
				x = 0
				y += step
			}
			n, err := r.Read(cr)
			if err != nil {
				if err != io.EOF {
					fmt.Println("read line err")
					return nil, err
				}
			} else if n == 0 {
				fmt.Println("n=0?!")
				break
			}
			fmt.Printf("[n=%d]", n)
			fmt.Println(cr)
			//for i := 0; i < n; i++ {
			p := cr[i]
			i++
			end := int(p&(0x7f)) + 1
			if p>>7&1 == 1 {
				bcol := cr[i]
				i++
				gcol := cr[i]
				i++
				rcol := cr[i]
				i++
				acol := cr[i]
				i++
				c0 := color.NRGBA{rcol, gcol, bcol, acol}
				fmt.Printf("-%d[%d]:%d,%d,%d,%d", i, end, rcol, gcol, bcol, acol)
				for j := 0; j < end; j++ {
					nrgba.SetNRGBA(x, y, c0)
					x++
					fmt.Printf("-%d ", x)
				}
			} else {
				end = end + i
				for ; i < end; i++ {
					bcol := cr[i]
					i++
					gcol := cr[i]
					i++
					rcol := cr[i]
					i++
					acol := cr[i]
					i++
					fmt.Printf(" %d[%d]:%d,%d,%d,%d", i, end, rcol, gcol, bcol, acol)
					c0 := color.NRGBA{rcol, gcol, bcol, acol}
					nrgba.SetNRGBA(x, y, c0)
					x++
					fmt.Printf("%d ", x)
				}
			}
			//}
		}
	default:
		switch bypp {
		case 3:
			for {
				if y == limit {
					break
				}
				n, err := io.ReadFull(d.r, cr)
				if err != nil && err != io.EOF {
					fmt.Println("read line err")
					return nil, err
				}
				if n == 0 {
					break
				}
				for x := 0; x < d.width; x++ {
					bcol := cr[x*bypp+0]
					gcol := cr[x*bypp+1]
					rcol := cr[x*bypp+2]
					acol := uint8(255)
					c0 := color.NRGBA{rcol, gcol, bcol, acol}
					nrgba.SetNRGBA(x, y, c0)
				}
				y += step
			}
		case 4:
			fallthrough
		default:
			for {
				if y == limit {
					break
				}
				n, err := io.ReadFull(d.r, cr)
				if err != nil && err != io.EOF {
					fmt.Println("read line err")
					return nil, err
				}
				if n == 0 {
					break
				}
				for x := 0; x < d.width; x++ {
					bcol := cr[x*bypp+0]
					gcol := cr[x*bypp+1]
					rcol := cr[x*bypp+2]
					acol := cr[x*bypp+3]
					c0 := color.NRGBA{rcol, gcol, bcol, acol}
					nrgba.SetNRGBA(x, y, c0)
				}
				y += step
			}
		}
	}

	return d.image, nil
}

// DecodeConfig returns the global color model and dimensions of a Targa image
// without decoding the entire image.
func DecodeConfig(r io.Reader) (image.Config, error) {
	d := &decoder{
		r:   r,
		crc: crc32.NewIEEE(),
	}
	if err := d.parseHeader(); err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return image.Config{}, err
	}
	cm := color.RGBAModel

	return image.Config{
		ColorModel: cm,
		Width:      d.width,
		Height:     d.height,
	}, nil
}

func init() {
	// If we register format with blank magic, it will match every file as targa
	// last. We can then verify the format with the 18byte header.
	//
	// Targa has no magic start bytes:
	image.RegisterFormat("tga", "", Decode, DecodeConfig)
	// Targa 2.0 has magic footer bytes:
	//image.RegisterFormat("tga", "TRUEVISION-XFILE.?", Decode, DecodeConfig)
}
