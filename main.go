package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"path/filepath"
	"encoding/json"

	"github.com/paulmach/go.geojson"
	"github.com/fogleman/gg"
	"github.com/davvo/mercator"
)

type ConfigType struct {
	CanvasWidth	int	`json:"canvasWidth"`
	CanvasHeight	int	`json:"canvasHeight"`
	DataDir		string	`json:"dataDir"`
}

type StyleType struct {
	FileName	string		`json:"fileName"`
	CanvasColor	[3]float64	`json:"canvasColor"`
	PointColor	[4]float64	`json:"pointColor"`
	PointRadius	float64		`json:"pointRadius"`
	LineWidth	float64		`json:"lineWidth"`
	LineColor	[4]float64	`json:"lineColor"`
	PolyWidth	float64		`json:"polyWidth"`
	PolyBorderColor [4]float64	`json:"polyBorderColor"`
	PolyColor	[4]float64	`json:"polyColor"`
	MPolyWidth	float64		`json:"mpolyWidth"`
	MPolyBorderColor [4]float64	`json:"mpolyBorderColor"`
	MPolyColor	[4]float64	`json:"mpolyColor"`
}

const MercatorMaxValue float64 = 20037508.342789244
var Styles = make(map[string]StyleType)

func main() {
	config, err := readConfig("./app.config")
	if (err != nil) {
		fmt.Println("Config reading error: ", err)
		return
	}

	err = readStyles("./app.style")
	if (err != nil) {
		fmt.Println("Style reading error: ", err)
		return
	}

	data, err := readData(config.DataDir)
	if (err != nil) {
		fmt.Println(err.Error)
		return
	}

	var MercatorToCanvasScaleFactorX =  float64(config.CanvasWidth) / (MercatorMaxValue)
	var MercatorToCanvasScaleFactorY =  float64(config.CanvasHeight) / (MercatorMaxValue)
	fmt.Println(MercatorToCanvasScaleFactorX, " ", MercatorToCanvasScaleFactorY)
	dc := gg.NewContext(config.CanvasWidth, config.CanvasHeight)
	dc.InvertY()
	dc.SetRGBA(1,1,1,1)
	dc.Clear()
	dc.SetDash()

	for name, fc := range data {
		//dc.SetRGB(Styles[name].CanvasColor[0], Styles[name].CanvasColor[1], Styles[name].CanvasColor[2])
		fmt.Println(name)
		for _, feature := range fc.Features {
			switch(feature.Geometry.Type) {
			case "Point":
				dc.SetRGBA(Styles[name].PointColor[0], Styles[name].PointColor[1], Styles[name].PointColor[2], Styles[name].PointColor[3])
				dc.DrawCircle(feature.Geometry.Point[0], feature.Geometry.Point[1], Styles[name].PointRadius)
				break
			case "LineString":
				dc.SetLineWidth(Styles[name].LineWidth)
				for i := 0; i < len(feature.Geometry.LineString); i++ {
					x, y := mercator.LatLonToMeters(feature.Geometry.LineString[i][1], feature.Geometry.LineString[i][0])
					dc.LineTo(x, y)
				}
				dc.SetRGBA(Styles[name].LineColor[0], Styles[name].LineColor[1], Styles[name].LineColor[2], Styles[name].LineColor[3])
				dc.Stroke()
				break
			case "Polygon":
				for _, poly := range feature.Geometry.Polygon {
					dc.SetLineWidth(Styles[name].MPolyWidth)
					for _, point := range poly {
						x, y := mercator.LatLonToMeters(point[1], point[0])
						if (x > 0) {
							x += MercatorMaxValue
						} else {
							x = MercatorMaxValue + x
						}

						if (y > 0) {
							y += MercatorMaxValue
						} else {
							y = MercatorMaxValue + y
						}

						x *= MercatorToCanvasScaleFactorX
						y *= MercatorToCanvasScaleFactorY

						dc.LineTo(x, y)
					}
					dc.ClosePath()
					dc.SetRGBA(Styles[name].PolyBorderColor[0], Styles[name].PolyBorderColor[1], Styles[name].PolyBorderColor[2], Styles[name].PolyBorderColor[3])
					dc.StrokePreserve()
					dc.SetRGBA(Styles[name].PolyColor[0], Styles[name].PolyColor[1], Styles[name].PolyColor[2], Styles[name].PolyColor[3])
					dc.Fill()
				}
				break
			case "MultiPolygon":
				for _, mpolys := range feature.Geometry.MultiPolygon {
					for _, mpoly := range mpolys {
						for _, mpoint := range mpoly {
							x, y := mercator.LatLonToMeters(mpoint[1], mpoint[0])
							x, y = centerRussia(x,y)

							x *= MercatorToCanvasScaleFactorX
							y *= MercatorToCanvasScaleFactorY

							dc.LineTo(x, y)
						}
						dc.ClosePath()
						dc.SetLineWidth(Styles[name].MPolyWidth)
						dc.SetRGBA(Styles[name].MPolyBorderColor[0], Styles[name].MPolyBorderColor[1], Styles[name].MPolyBorderColor[2], Styles[name].MPolyBorderColor[3])
						dc.StrokePreserve()
						dc.SetRGBA(Styles[name].MPolyColor[0], Styles[name].MPolyColor[1], Styles[name].MPolyColor[2], Styles[name].MPolyColor[3])
						dc.Fill()
					}
				}
				break
			}
		}
	}

	dc.SavePNG("out.png")
	return
}

func readConfig(path string) (config *ConfigType, err error) {
	configFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(configFile, &config)
	if err != nil {
		return nil, err
	}

	return config, err
}

func readStyles(path string) (err error) {
	var style []StyleType
	styleFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	err = json.Unmarshal(styleFile, &style)
	if err != nil {
		return err
	}

	for _, levelStyle := range style {
		Styles[levelStyle.FileName] = levelStyle
	}
	return err
}

func readData(path string) (map[string]*geojson.FeatureCollection, error) {
	var data = make(map[string]*geojson.FeatureCollection)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if (!file.IsDir() && strings.HasSuffix(file.Name(), ".geojson")) {
			rawFeatureJson, err := ioutil.ReadFile(filepath.Join(path, file.Name()))
			if (err != nil) {
				fmt.Println(err)
				return nil, err
			}

			fc, err := geojson.UnmarshalFeatureCollection(rawFeatureJson)
			if (err != nil) {
				fmt.Println(err)
				return nil, err
			}

			data[file.Name()] = fc
		}
	}

	return data, err
}

func centerRussia(x float64, y float64) (float64, float64) {
	var west float64 = 1635093.15883866

	if (x > 0) {
		x -= west
	} else {
		x += 2*MercatorMaxValue - west
	}

	return x, y
}