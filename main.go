package main

import (
	"log"
	"math"
	"sync"
	"time"

	"github.com/exrook/drawille-go"
	"github.com/gdamore/tcell/v2"
)

var (
	colorB = [3]float64{194, 229, 156}
	colorA = [3]float64{100, 179, 244}
)

func linearGradient(x float64) (int32, int32, int32) {
	d := x / 256
	r := colorA[0] + d*(colorB[0]-colorA[0])
	g := colorA[1] + d*(colorB[1]-colorA[1])
	b := colorA[2] + d*(colorB[2]-colorA[2])
	return int32(r), int32(g), int32(b)
}

func main() {
	s, err := tcell.NewScreen()
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}

	// Set default text style
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	s.SetStyle(defStyle)
	s.Clear()
	s.EnableMouse()

	canvas := drawille.NewCanvas()
	// cb := ColorBuffer{}

	// go func() {
	// 	for range time.NewTicker(time.Millisecond * 10).C {
	// 		points := cb.Tick()
	// 		for _, p := range points {
	// 			if p.i == 255 {
	// 				// Reset
	// 				s.SetContent(p.x, p.y, ' ', nil, tcell.StyleDefault)
	// 				continue
	// 			}
	// 			mainc, _, _, _ := s.GetContent(p.x, p.y)
	// 			s.SetContent(p.x, p.y, mainc, nil,
	// 				tcell.StyleDefault.Foreground(
	// 					tcell.NewRGBColor(
	// 						linearGradient(
	// 							float64(p.i),
	// 						),
	// 					),
	// 				),
	// 			)
	// 		}
	// 		s.Show()
	// 	}
	// }()

	var firstMouse = true
	prevX, prevY := 0, 0

	last := time.Now()
	for {
		// Update screen
		s.Show()

		// Poll event
		ev := s.PollEvent()
		// Process event
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			if time.Since(last) < time.Millisecond*10 {
				continue
			}
			last = time.Now()
			x, y := ev.Position()
			if firstMouse {
				firstMouse = false
				prevX, prevY = x, y
			}
			DrawLine(canvas, float64(prevX*2), float64(prevY*4), float64(x*2), float64(y*4))
			diffx := abs(prevX - x)
			diffy := abs(prevY - y)
			minx := min(prevX, x)
			miny := min(prevY, y)

			for xo := 0; xo <= diffx; xo++ {
				for yo := 0; yo <= diffy; yo++ {
					s.SetContent(minx+xo, miny+yo, canvas.GetScreenCharacter(minx+xo, miny+yo), nil, defStyle)
				}
			}

			prevX, prevY = x, y
			// cb.AddPoint(x, y)

		case *tcell.EventResize:
			s.Sync()
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				goto END
			}
		}
	}

END:
	s.Clear()
	s.Fini() // end
}

func abs(y int) int {
	if y < 0 {
		return y * -1
	}
	return y
}
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func round(x float64) int {
	return int(x + 0.5)
}

func distance(x1, y1, x2, y2 int) float64 {
	return math.Sqrt(float64((x2 - x1) + (y2 - y1)))
}

type Point struct {
	x int
	y int
	i int
}

type ColorBuffer struct {
	buffers [256][]Point
	index   int

	lock sync.Mutex
}

func (cb *ColorBuffer) indexBuffer(i int) int {
	return (cb.index + i) % 256
}

func (cb *ColorBuffer) AddPoint(x, y int) {
	cb.lock.Lock()
	for i := 0; i < 256; i++ {
		cb.buffers[cb.indexBuffer(i)] = append(cb.buffers[cb.indexBuffer(i)], Point{x, y, i})
	}
	cb.lock.Unlock()
}

func (cb *ColorBuffer) Tick() []Point {
	cb.lock.Lock()
	out := cb.buffers[cb.index]
	cb.buffers[cb.index] = nil
	cb.index++
	cb.index %= 256
	cb.lock.Unlock()
	return out
}

func DrawLine(c drawille.Canvas, x1, y1, x2, y2 float64) {
	xdiff := math.Abs(x1 - x2)
	ydiff := math.Abs(y2 - y1)

	var xdir, ydir float64
	if x1 <= x2 {
		xdir = 1
	} else {
		xdir = -1
	}
	if y1 <= y2 {
		ydir = 1
	} else {
		ydir = -1
	}

	r := math.Max(xdiff, ydiff)

	for i := 0; i < round(r)+1; i++ {
		x, y := x1, y1
		if ydiff != 0 {
			y += (float64(i) * ydiff) / (r * ydir)
		}
		if xdiff != 0 {
			x += (float64(i) * xdiff) / (r * xdir)
		}
		c.Set(round(x), round(y))
	}
}
