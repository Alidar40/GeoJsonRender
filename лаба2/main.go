package main

import (
	"fmt" // пакет для форматированного ввода вывода
	// пакет для логирования

	"io/ioutil"
	"math"
	"math/rand" // пакет для поддержки HTTP протокола
	"strconv"

	"github.com/davvo/mercator"
	"github.com/fogleman/gg"
	geojson "github.com/paulmach/go.geojson"
	// пакет для работы с  UTF-8 строками
)

const width, height = 256, 256

func draw(z, x, y float64) {
	var err error
	var img string

	var featureCollectionJSON []byte
	var filePath = "rf.geojson"

	if featureCollectionJSON, err = ioutil.ReadFile(filePath); err != nil {
		fmt.Println(err.Error())
	}

	if img, err = getPNG(featureCollectionJSON, z, x, y); err != nil {
		fmt.Println(err.Error())
	}

	println(img)
}

func main() {
	var z, x, y float64
	fmt.Scan(&z, &x, &y)
	draw(z, x, y)
}

func getPNG(featureCollectionJSON []byte, z float64, x float64, y float64) (string, error) {
	var coordinates [][][][][]float64
	var err error

	if coordinates, err = getMultyCoordinates(featureCollectionJSON); err != nil {
		return err.Error(), err
	}

	dc := gg.NewContext(width, height)
	scale := 1.0

	dc.InvertY()
	//рисуем полигоны
	forEachPolygon(dc, coordinates, func(polygonCoordinates [][]float64) {
		dc.SetRGB(rand.Float64(), rand.Float64(), rand.Float64())
		drawByPolygonCoordinates(dc, polygonCoordinates, scale, dc.Fill, z, x, y)
	})
	//рисуем контуры полигонов
	dc.SetLineWidth(2)
	forEachPolygon(dc, coordinates, func(polygonCoordinates [][]float64) {
		dc.SetRGB(rand.Float64(), rand.Float64(), rand.Float64())
		drawByPolygonCoordinates(dc, polygonCoordinates, scale, dc.Stroke, z, x, y)
	})

	var out = strconv.Itoa(rand.Intn(10000)) + ".png"

	dc.SavePNG(out)

	return out, nil
}

func getMultyCoordinates(featureCollectionJSON []byte) ([][][][][]float64, error) {
	var featureCollection *geojson.FeatureCollection
	var err error

	if featureCollection, err = geojson.UnmarshalFeatureCollection(featureCollectionJSON); err != nil {
		return nil, err
	}
	var features = featureCollection.Features
	var coordinates [][][][][]float64
	for i := 0; i < len(features); i++ {
		coordinates = append(coordinates, features[i].Geometry.MultiPolygon)
	}
	return coordinates, nil
}

func forEachPolygon(dc *gg.Context, coordinates [][][][][]float64, callback func([][]float64)) {
	for i := 0; i < len(coordinates); i++ {
		for j := 0; j < len(coordinates[i]); j++ {
			callback(coordinates[i][j][0])
		}
	}
}

const mercatorMaxValue float64 = 20037508.342789244

const mercatorToCanvasScaleFactorX = float64(width) / (mercatorMaxValue)
const mercatorToCanvasScaleFactorY = float64(height) / (mercatorMaxValue)

func drawByPolygonCoordinates(dc *gg.Context, coordinates [][]float64, scale float64, method func(), z float64, xTile float64, yTile float64) {

	scale = scale * math.Pow(2, z)

	dx := float64(dc.Width())*(xTile) - 138.5*scale
	dy := float64(dc.Height())*(math.Pow(2, z)-1-yTile) - 128*scale

	for index := 0; index < len(coordinates)-1; index++ {
		x, y := mercator.LatLonToMeters(coordinates[index][1], convertNegativeX(coordinates[index][0]))

		x, y = centerRussia(x, y)

		x *= mercatorToCanvasScaleFactorX * scale * 0.5
		y *= mercatorToCanvasScaleFactorY * scale * 0.5

		x -= dx
		y -= dy

		dc.LineTo(x, y)
	}
	dc.ClosePath()
	method()
}

func centerRussia(x float64, y float64) (float64, float64) {
	var west = float64(1635093.15883866)

	if x > 0 {
		x -= west
	} else {
		x += 2*mercatorMaxValue - west
	}

	return x, y
}

func convertNegativeX(x float64) float64 {
	if x < 0 {
		x = x - 360
	}
	return x
}
