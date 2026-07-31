package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
	"log"
	"math"
	"os"
)

func main() {
	image_filepath := "imgs/img.jpeg"
	img := loadAsNRGBA(image_filepath)

	var multiplyer float64 = 79 / float64(img.Bounds().Size().X)
	resized := resize(getImageGrid(img), multiplyer)
	saveGrid("imgs/resized.jpeg", resized)

	_, _, _, colors := createColorFrequencyMap(resized)
	fmt.Println(getColorChannelWithGreatestRange(colors))
}

func loadAsNRGBA(filePath string) *image.NRGBA {
	imgFile, err := os.Open(filePath)
	if err != nil {
		log.Println("Cannot read file: ", err)
	}
	defer imgFile.Close()

	decoded_img, _, err := image.Decode(imgFile)
	if err != nil {
		log.Println("Cannot decode file: ", err)
	}

	if decoded_img.ColorModel() != color.NRGBAModel {
		converted_img := toNRGBA(decoded_img)
		return converted_img
	} else {
		return decoded_img.(*image.NRGBA)
	}
}

func getImageGrid(img *image.NRGBA) (grid [][]color.Color) {
	size := img.Bounds().Size()
	for i := 0; i < size.X; i++ {
		var y []color.Color
		for j := 0; j < size.Y; j++ {
			y = append(y, img.At(i, j))
		}
		grid = append(grid, y)
	}
	return
}

func toNRGBA(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	converted := image.NewNRGBA(bounds)
	draw.Draw(converted, bounds, src, bounds.Min, draw.Src)
	return converted
}

func resize(grid [][]color.Color, scale float64) (resized [][]color.Color) {
	xlen, ylen := int(float64(len(grid))*scale), int(float64(len(grid[0]))*scale)
	resized = make([][]color.Color, xlen)
	for i := 0; i < len(resized); i++ {
		resized[i] = make([]color.Color, ylen)
	}
	for x := 0; x < xlen; x++ {
		for y := 0; y < ylen; y++ {
			xp := int(math.Floor(float64(x) / scale))
			yp := int(math.Floor(float64(y) / scale))
			resized[x][y] = grid[xp][yp]
		}
	}
	return
}

func saveGrid(filePath string, grid [][]color.Color) {
	xlen, ylen := len(grid), len(grid[0])
	rect := image.Rect(0, 0, xlen, ylen)
	img := image.NewNRGBA(rect)
	for x := 0; x < xlen; x++ {
		for y := 0; y < ylen; y++ {
			img.Set(x, y, grid[x][y])
		}
	}
	imgFile, err := os.Create(filePath)

	if err != nil {
		log.Println("Cannot create file: ", err)
	}
	defer imgFile.Close()

	jpeg.Encode(imgFile, img.SubImage(img.Rect), nil)
}

func save(filePath string, img *image.NRGBA) {
	imgFile, err := os.Create(filePath)

	if err != nil {
		log.Println("Cannot create file: ", err)
	}
	defer imgFile.Close()

	jpeg.Encode(imgFile, img.SubImage(img.Rect), nil)
}

func createColorFrequencyMap(grid [][]color.Color) (colorsMap map[color.Color]int, leadColorFrequency int, leadColor string, colors []color.Color) {
	xlen, ylen := int(float64(len(grid))), int(float64(len(grid[0])))
	colorsMap = make(map[color.Color]int)
	var reds, greens, blues uint32
	for x := 0; x < xlen; x++ {
		for y := 0; y < ylen; y++ {
			pixel := grid[x][y]

			elem, ok := colorsMap[pixel]
			if ok == false {
				colorsMap[pixel] = 1
				colors = append(colors, pixel)
			} else {
				colorsMap[pixel] = elem + 1
			}

			var r, g, b, _ uint32 = pixel.RGBA()
			reds += r
			greens += g
			blues += b
		}
	}

	if reds > greens {
		if reds > blues {
			leadColorFrequency = int(reds)
			leadColor = "red"
		} else {
			leadColorFrequency = int(blues)
			leadColor = "blue"
		}
	} else {
		if greens > blues {
			leadColorFrequency = int(greens)
			leadColor = "green"
		} else {
			leadColorFrequency = int(blues)
			leadColor = "blue"
		}
	}

	return colorsMap, leadColorFrequency, leadColor, colors
}

func getColorChannelWithGreatestRange(colors []color.Color) string {
	var minR, minG, minB uint32 = 0xffff, 0xffff, 0xffff

	var maxR, maxG, maxB uint32 = 0, 0, 0

	for _, elem := range colors {
		red, green, blue, _ := elem.RGBA()

		if red < minR {
			minR = red
		}
		if green < minG {
			minG = green
		}
		if blue < minB {
			minB = blue
		}

		if red > maxR {
			maxR = red
		}
		if green > maxG {
			maxG = green
		}
		if blue > maxB {
			maxB = blue
		}
	}

	rangeR := maxR - minR
	rangeB := maxB - minB
	rangeG := maxG - minG

	if rangeR > rangeG {
		if rangeR > rangeB {
			return "red"
		}
		return "blue"
	} else {
		if rangeG > rangeB {
			return "green"
		}
		return "blue"
	}
}
