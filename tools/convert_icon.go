package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"os"
)

// Bilinear interpolation resizer
func resizeImage(src image.Image, targetW, targetH int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	bounds := src.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	for y := 0; y < targetH; y++ {
		for x := 0; x < targetW; x++ {
			// Source coordinates
			gx := float64(x) / float64(targetW) * float64(srcW-1)
			gy := float64(y) / float64(targetH) * float64(srcH-1)
			gxi := int(gx)
			gyi := int(gy)
			dx := gx - float64(gxi)
			dy := gy - float64(gyi)

			c00 := src.At(bounds.Min.X+gxi, bounds.Min.Y+gyi)
			c10 := src.At(bounds.Min.X+gxi+1, bounds.Min.Y+gyi)
			c01 := src.At(bounds.Min.X+gxi, bounds.Min.Y+gyi+1)
			c11 := src.At(bounds.Min.X+gxi+1, bounds.Min.Y+gyi+1)

			r00, g00, b00, a00 := c00.RGBA()
			r10, g10, b10, a10 := c10.RGBA()
			r01, g01, b01, a01 := c01.RGBA()
			r11, g11, b11, a11 := c11.RGBA()

			interp := func(v00, v10, v01, v11 uint32) uint8 {
				top := float64(v00)*(1-dx) + float64(v10)*dx
				bot := float64(v01)*(1-dx) + float64(v11)*dx
				val := top*(1-dy) + bot*dy
				return uint8(val / 257)
			}

			dst.SetRGBA(x, y, color.RGBA{
				R: interp(r00, r10, r01, r11),
				G: interp(g00, g10, g01, g11),
				B: interp(b00, b10, b01, b11),
				A: interp(a00, a10, a01, a11),
			})
		}
	}
	return dst
}

type iconDirEntry struct {
	Width       byte
	Height      byte
	ColorCount  byte
	Reserved    byte
	Planes      uint16
	BitCount    uint16
	BytesInRes  uint32
	ImageOffset uint32
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: convert_icon <input.jpg/png> <output.ico>")
		os.Exit(1)
	}

	inFile := os.Args[1]
	outFile := os.Args[2]

	f, err := os.Open(inFile)
	if err != nil {
		fmt.Printf("Error opening input: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	srcImg, _, err := image.Decode(f)
	if err != nil {
		fmt.Printf("Error decoding image: %v\n", err)
		os.Exit(1)
	}

	sizes := []int{256, 128, 64, 48, 32, 24, 16}
	var pngBuffers [][]byte

	for _, sz := range sizes {
		resized := resizeImage(srcImg, sz, sz)
		var buf bytes.Buffer
		if err := png.Encode(&buf, resized); err != nil {
			fmt.Printf("Error encoding png at %dx%d: %v\n", sz, sz, err)
			os.Exit(1)
		}
		pngBuffers = append(pngBuffers, buf.Bytes())
	}

	// Build ICO file
	var outBuf bytes.Buffer

	// ICONDIR Header (6 bytes)
	binary.Write(&outBuf, binary.LittleEndian, uint16(0)) // Reserved
	binary.Write(&outBuf, binary.LittleEndian, uint16(1)) // Type 1 = ICO
	binary.Write(&outBuf, binary.LittleEndian, uint16(len(sizes))) // Count

	// Calculate offsets
	headerSize := 6 + len(sizes)*16
	currentOffset := uint32(headerSize)

	for i, sz := range sizes {
		w := byte(sz)
		h := byte(sz)
		if sz == 256 {
			w = 0
			h = 0
		}
		entry := iconDirEntry{
			Width:       w,
			Height:      h,
			ColorCount:  0,
			Reserved:    0,
			Planes:      1,
			BitCount:    32,
			BytesInRes:  uint32(len(pngBuffers[i])),
			ImageOffset: currentOffset,
		}
		binary.Write(&outBuf, binary.LittleEndian, entry)
		currentOffset += uint32(len(pngBuffers[i]))
	}

	// Write PNG payloads
	for _, pngData := range pngBuffers {
		outBuf.Write(pngData)
	}

	if err := os.WriteFile(outFile, outBuf.Bytes(), 0644); err != nil {
		fmt.Printf("Error writing ico file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[+] Successfully created multi-resolution ICO: %s (%d sizes, %d bytes)\n",
		outFile, len(sizes), outBuf.Len())
}
