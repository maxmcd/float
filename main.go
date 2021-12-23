package main

import (
	"log"
	"sync"
	"time"

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
	s.SetContent(0, 0, 'H', nil, defStyle)
	s.SetContent(1, 0, 'i', nil, defStyle)
	s.SetContent(2, 0, '!', nil, defStyle)

	s.EnableMouse()

	cb := ColorBuffer{}

	go func() {
		for range time.NewTicker(time.Millisecond * 10).C {
			points := cb.Tick()
			for _, p := range points {
				content := '0'
				if p.i == 255 {
					content = ' '
				}
				s.SetContent(p.x, p.y, content, nil,
					tcell.StyleDefault.Foreground(
						tcell.NewRGBColor(
							linearGradient(
								float64(p.i),
							),
						),
					),
				)
			}
			s.Show()
		}
	}()

	for {
		// Update screen
		s.Show()

		// Poll event
		ev := s.PollEvent()
		// Process event
		switch ev := ev.(type) {
		case *tcell.EventMouse:
			x, y := ev.Position()
			s.SetContent(x, y, 'O', nil, defStyle)
			cb.AddPoint(x, y)
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
