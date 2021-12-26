package main

import (
	_ "embed"
	"encoding/json"
)

var (
	//go:embed gradients.json
	gradientBody []byte
	gradients    []Gradient
)

func init() {
	if err := json.Unmarshal(gradientBody, &gradients); err != nil {
		panic(err)
	}
	for i, g := range gradients {
		for _, c := range g.Colors {
			r, g, b := parseHexColor(c)
			gradients[i].colors = append(gradients[i].colors, []float64{r, g, b})
		}
	}
}

type Gradient struct {
	Name   string
	Colors []string
	colors [][]float64
}

func parseHexColor(s string) (r, g, b float64) {
	if s[0] != '#' {
		panic("invalid format")
	}

	hexToByte := func(b byte) byte {
		switch {
		case b >= '0' && b <= '9':
			return b - '0'
		case b >= 'a' && b <= 'f':
			return b - 'a' + 10
		case b >= 'A' && b <= 'F':
			return b - 'A' + 10
		}
		panic("invalid format")
	}

	switch len(s) {
	case 7:
		r = float64(hexToByte(s[1])<<4 + hexToByte(s[2]))
		g = float64(hexToByte(s[3])<<4 + hexToByte(s[4]))
		b = float64(hexToByte(s[5])<<4 + hexToByte(s[6]))
	case 4:
		r = float64(hexToByte(s[1]) * 17)
		g = float64(hexToByte(s[2]) * 17)
		b = float64(hexToByte(s[3]) * 17)
	default:
		panic("invalid format")
	}
	return
}

// point provides the color of a gradient at this point on a 0-256 scale
func (gradient Gradient) point(x int) (r, g, b int32) {
	segments := len(gradient.colors)
	if segments == 2 {
		return linearGradient(
			gradient.colors[0],
			gradient.colors[1],
			float64(x))
	}
	if segments == 3 {
		if x > 128 {
			return linearGradient(
				gradient.colors[1],
				gradient.colors[2],
				float64((x-128)*2),
			)
		}
		return linearGradient(
			gradient.colors[0],
			gradient.colors[1],
			float64(x*2),
		)
	}
	panic("unimplemented")
}

func linearGradient(colorA []float64, colorB []float64, x float64) (int32, int32, int32) {
	d := x / 256
	r := colorA[0] + d*(colorB[0]-colorA[0])
	g := colorA[1] + d*(colorB[1]-colorA[1])
	b := colorA[2] + d*(colorB[2]-colorA[2])
	return int32(r), int32(g), int32(b)
}
