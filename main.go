package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

type colorBucket struct {
	colors     []color.Color
	colorRange int
}

func main() {
	image_filepath := "imgs/img.jpeg"
	processImage(image_filepath, 79, 52, "imgs/Converted/peripetia")
	processFrames("imgs/frames", 79, 52, "test")
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

func resize(grid [][]color.Color, scaleX float64, scaleY float64) (resized [][]color.Color) {
	xlen, ylen := int(float64(len(grid))*scaleX), int(float64(len(grid[0]))*scaleY)
	resized = make([][]color.Color, xlen)
	for i := 0; i < len(resized); i++ {
		resized[i] = make([]color.Color, ylen)
	}
	for x := 0; x < xlen; x++ {
		for y := 0; y < ylen; y++ {
			xp := int(math.Floor(float64(x) / scaleX))
			yp := int(math.Floor(float64(y) / scaleY))
			resized[x][y] = grid[xp][yp]
		}
	}
	return
}

func mergeImages(imgs []image.Image, filePath string) {
	var finalImg [][]color.Color
	for i := 0; i < len(imgs); i++ {
		grid := getImageGrid(toNRGBA(imgs[i]))
		if i == 0 {
			finalImg = grid
			continue
		}

		for y := 0; y < len(grid); y++ {
			finalImg[y] = append(finalImg[y], grid[y]...)
		}
	}

	saveGrid(filePath, finalImg)
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

	png.Encode(imgFile, img.SubImage(img.Rect))
}

func gridToDefault(grid [][]color.Color) image.Image {
	xlen, ylen := len(grid), len(grid[0])
	rect := image.Rect(0, 0, xlen, ylen)
	img := image.NewNRGBA(rect)
	for x := 0; x < xlen; x++ {
		for y := 0; y < ylen; y++ {
			img.Set(x, y, grid[x][y])
		}
	}
	return img
}

func gridColors(grid [][]color.Color) (colors []color.Color) {
	xlen, ylen := int(float64(len(grid))), int(float64(len(grid[0])))
	for x := 0; x < xlen; x++ {
		for y := 0; y < ylen; y++ {
			pixel := grid[x][y]
			colors = append(colors, pixel)
		}
	}
	return colors
}

func save(filePath string, img *image.NRGBA) {
	imgFile, err := os.Create(filePath)

	if err != nil {
		log.Println("Cannot create file: ", err)
	}
	defer imgFile.Close()

	jpeg.Encode(imgFile, img.SubImage(img.Rect), nil)
}

// creates map of colors, frequency of a lead color, lead color channel and a slice of unique colors
// might wanna reduce return values
func createColorFrequencyMap(grid [][]color.Color) (colorsMap map[color.Color]int) {
	xlen, ylen := int(float64(len(grid))), int(float64(len(grid[0])))
	colorsMap = make(map[color.Color]int)
	for x := 0; x < xlen; x++ {
		for y := 0; y < ylen; y++ {
			pixel := grid[x][y]

			elem, ok := colorsMap[pixel]
			if ok == false {
				colorsMap[pixel] = 1
			} else {
				colorsMap[pixel] = elem + 1
			}
		}
	}

	return colorsMap
}

// return "red", "green" or "blue" str representing the color channel with greatest range
func getColorChannelWithGreatestRange(colors []color.Color) (colorchannel string) {
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

// String arg must be "red", "blue" or "green". Otherwise will return 0
func getColorRange(colors []color.Color, colorChannel string) (colorRange int) {
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

	switch colorChannel {
	case "red":
		return int(rangeR)
	case "blue":
		return int(rangeB)
	case "green":
		return int(rangeG)
	default:
		return 0
	}
}

// String arg must be "red", "blue" or "green".
func sortByColor(colors []color.Color, colorChannel string) {
	switch colorChannel {
	case "red":
		sort.Slice(colors, func(i, j int) bool {
			r1, _, _, _ := colors[i].RGBA()
			r2, _, _, _ := colors[j].RGBA()
			return r1 > r2
		})
	case "green":
		sort.Slice(colors, func(i, j int) bool {
			_, g1, _, _ := colors[i].RGBA()
			_, g2, _, _ := colors[j].RGBA()
			return g1 > g2
		})
	case "blue":
		sort.Slice(colors, func(i, j int) bool {
			_, _, b1, _ := colors[i].RGBA()
			_, _, b2, _ := colors[j].RGBA()
			return b1 > b2
		})
	}
}

func get16Buckets(sortedColors []color.Color) []colorBucket {
	colorBuckets := []colorBucket{
		{colors: append([]color.Color(nil), sortedColors...),
			colorRange: getColorRange(sortedColors, getColorChannelWithGreatestRange(sortedColors))},
	}

	for len(colorBuckets) < 16 {
		index := findBucketToSplit(colorBuckets)
		newBucket := colorBuckets[index]
		colorChannel := getColorChannelWithGreatestRange(newBucket.colors)
		sortByColor(newBucket.colors, colorChannel)

		middle := len(newBucket.colors) / 2
		leftColors := append([]color.Color(nil), newBucket.colors[middle:]...)
		rightColors := append([]color.Color(nil), newBucket.colors[:middle]...)

		leftBucket := makeBucket(leftColors)
		rightBucket := makeBucket(rightColors)

		colorBuckets = append(colorBuckets[:index], colorBuckets[index+1:]...)
		colorBuckets = append(colorBuckets, leftBucket, rightBucket)
	}

	return colorBuckets
}

func findBucketToSplit(colorBuckets []colorBucket) int {
	if len(colorBuckets) < 2 {
		return 0
	}

	cur := -1
	widestRange := -1

	for i, bucket := range colorBuckets {
		if bucket.colorRange > widestRange {
			cur = i
			widestRange = bucket.colorRange
		}
	}

	return cur
}

func makeBucket(colors []color.Color) colorBucket {
	colorToSortBy := getColorChannelWithGreatestRange(colors)
	sortByColor(colors, colorToSortBy)
	WidestColorColorRange := getColorRange(colors, colorToSortBy)

	return colorBucket{
		colors:     colors,
		colorRange: WidestColorColorRange,
	}
}

func averageColors(colors []color.Color) color.Color {
	length := len(colors)

	var reds, greens, blues uint64

	for _, elem := range colors {
		red, green, blue, _ := elem.RGBA()
		reds += uint64(red)
		blues += uint64(blue)
		greens += uint64(green)
	}

	avgRed := reds / uint64(length)
	avgGreen := greens / uint64(length)
	avgBlue := blues / uint64(length)

	return color.RGBA{
		R: uint8(avgRed >> 8),
		G: uint8(avgGreen >> 8),
		B: uint8(avgBlue >> 8),
		A: 255,
	}
}

func PaletteFromBuckets(buckets []colorBucket) (NewPalette []color.Color) {
	for _, elem := range buckets {
		newColor := averageColors(elem.colors)
		NewPalette = append(NewPalette, newColor)
	}

	return NewPalette
}

func Quantize(colors []color.Color) []color.Color {
	buckets := get16Buckets(colors)

	return PaletteFromBuckets(buckets)
}

func writePaletteToHexFile(filename string, p color.Palette) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, c := range p {
		// Get 16-bit color channels and scale down to 8-bit (0-255)
		r, g, b, _ := c.RGBA()
		hexStr := fmt.Sprintf("%02X%02X%02X\n", uint8(r>>8), uint8(g>>8), uint8(b>>8))

		_, err := file.WriteString(hexStr)
		if err != nil {
			return err
		}
	}
	return nil
}

func writePixelsToFile(filename string, img image.Image, colorPalette color.Palette) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	for y := img.Bounds().Min.Y; y < img.Bounds().Size().Y; y++ {
		var row string
		for x := img.Bounds().Min.X; x < img.Bounds().Size().X; x++ {
			pix := img.At(x, y)
			row += fmt.Sprintf("%x", colorPalette.Index(pix))
		}
		row = row + "\n"
		_, err := file.WriteString(row)
		if err != nil {
			return err
		}
	}
	return nil
}

func processImage(filePath string, targetXScale int, targetYScale int, outputFilePath string) {
	src := loadAsNRGBA(filePath)
	var multiplyerX float64 = float64(targetXScale) / float64(src.Bounds().Size().X)
	var multiplyerY float64 = float64(targetYScale) / float64(src.Bounds().Size().Y)

	resized := resize(getImageGrid(src), multiplyerX, multiplyerY)
	resizedImgColors := gridColors(resized)

	quantized := Quantize(resizedImgColors)
	CC_Custom_Palette := color.Palette{}
	for _, clr := range quantized {
		CC_Custom_Palette = append(CC_Custom_Palette, clr)
	}

	quantizedImg := image.NewPaletted(image.Rect(0, 0, targetXScale, targetYScale), CC_Custom_Palette)
	draw.Draw(quantizedImg, quantizedImg.Rect, gridToDefault(resized), gridToDefault(resized).Bounds().Min, draw.Src)

	// saving result as image
	fileName := outputFilePath + ".png"
	file, err := os.Create(fileName)

	if err != nil {
		log.Println("Failed to create quantized.png: ", err)
	}
	defer file.Close()

	err = png.Encode(file, quantizedImg)
	if err != nil {
		log.Println("Failed to encode quantized.png: ", err)
	}

	// saving result for CC
	writePaletteToHexFile(outputFilePath+"palette.txt", CC_Custom_Palette)
	writePixelsToFile(outputFilePath+"pixels.txt", quantizedImg, CC_Custom_Palette)
}

func processFrames(dir string, targetXScale int, targetYScale int, outputFilesName string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Println(err)
	}

	tmpDir, err := os.MkdirTemp("imgs/", "temp*")
	if err != nil {
		log.Println("Couldn't create temp dir: ", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	for i, e := range entries {
		inputFilePath := filepath.Join(dir, e.Name())
		outputFilePath := filepath.Join(tmpDir, outputFilesName+strconv.Itoa(i))
		processImage(inputFilePath, targetXScale, targetYScale, outputFilePath)
	}

	var imgsToMerge []image.Image

	for i := 0; i < len(entries); i++ {
		filepath := filepath.Join(tmpDir, outputFilesName+strconv.Itoa(i)+".png")
		img := loadAsNRGBA(filepath)
		imgsToMerge = append(imgsToMerge, img)
	}

	mergedImgsFP := filepath.Join(tmpDir, "merged.png")
	mergeImages(imgsToMerge, mergedImgsFP)
	mergedImg := loadAsNRGBA(mergedImgsFP)
	processImage(mergedImgsFP, mergedImg.Rect.Max.X, mergedImg.Rect.Max.Y, "imgs/Converted/Frames/"+outputFilesName)
}
