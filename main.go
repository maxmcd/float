package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gliderlabs/ssh"

	_ "embed"
)

var (
	//go:embed gradients.json
	gradientBody []byte
	gradients    []Gradient
)

func init() {
	os.Setenv("COLORTERM", "truecolor")
	rand.Seed(time.Now().UnixNano())
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

type Float struct {
	cb          *ColorBuffer
	multisetter *ScreenMultisetter
}

func (f Float) Start() {
	f.cb = &ColorBuffer{}
	f.multisetter = &ScreenMultisetter{screens: make(map[int]tcell.Screen)}

	connection := 0
	ssh.Handle(func(s ssh.Session) {
		id := connection
		connection++
		fmt.Println("New connection ->", id)
		screen, err := tcell.NewTerminfoScreenFromTty(NewSSHSessionTTYWrapper(s))
		if err != nil {
			fmt.Fprintln(s, err.Error())
			_ = s.Exit(1)
			return
		}
		if err := f.Run(screen); err != nil {
			fmt.Fprintln(s, err.Error())
			_ = s.Exit(1)
			return
		}
		fmt.Fprintln(s, "     ......")
		fmt.Fprintln(s, "         ......")
		fmt.Fprintln(s, "Goodbye, thanks for floating")
		fmt.Fprintln(s, "               ......")
		fmt.Fprintln(s, "                   ......")

		fmt.Println("Closed         ->", id)
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Millisecond * 10).C
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker:
				points := f.cb.Tick()
				for _, p := range points {
					content := '█'
					if p.i == 255 {
						content = ' '
					}
					f.multisetter.SetContent(p.x, p.y, content, nil,
						tcell.StyleDefault.Foreground(
							tcell.NewRGBColor(
								p.g.point(p.i),
							),
						),
					)
				}
			}
		}
	}()
	_ = cancel
	addr := ":2222"
	go startGotty()

	fmt.Println("Listening for SSH connections at", addr)
	log.Fatal(ssh.ListenAndServe(addr, nil))
}

func (f Float) Run(screen tcell.Screen) error {
	if err := screen.Init(); err != nil {
		return err
	}

	color := rand.Intn(len(gradients))

	// Set default text style
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	screen.SetStyle(defStyle)

	// Clean canvas
	screen.Clear()
	screen.EnableMouse()

	for i, c := range "https://float.maxmcd.com or 'ssh float.maxmcd.com'" {
		screen.SetContent(i, 0, c, nil, defStyle)
	}

	// Add screen to setter queue and save id for removing later
	id := f.multisetter.Add(screen)

	// Render at 60fps
	ticker := time.NewTicker(time.Millisecond * 1000 / 60).C

	events := make(chan tcell.Event)
	quit := make(chan struct{})
	go screen.ChannelEvents(events, quit)

	for {
		// // Update screen
		// screen.Show()
		select {
		case <-ticker:
			screen.Show()
		case ev := <-events:
			switch ev := ev.(type) {
			case *tcell.EventMouse:
				x, y := ev.Position()
				f.cb.AddPoint(x, y, gradients[color])
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
				if ev.Rune() == ' ' {
					color = rand.Intn(len(gradients))
				}
				if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
					quit <- struct{}{}
					goto END
				}
			}
		}
	}

END:
	f.multisetter.Remove(id)
	screen.Clear()
	screen.Fini() // end
	return nil
}

type ScreenMultisetter struct {
	screens map[int]tcell.Screen
	lock    sync.Mutex
}

func (sm *ScreenMultisetter) Add(screen tcell.Screen) (id int) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	for {
		id = rand.Int()
		if _, found := sm.screens[id]; !found {
			break
		}
	}
	sm.screens[id] = screen
	return id
}

func (sm *ScreenMultisetter) SetContent(x int, y int, mainc rune, combc []rune, style tcell.Style) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	for _, screen := range sm.screens {
		screen.SetContent(x, y, mainc, combc, style)
	}
}

func (sm *ScreenMultisetter) Remove(id int) {
	sm.lock.Lock()
	delete(sm.screens, id)
	sm.lock.Unlock()
}

func main() {
	f := Float{}
	f.Start()
}

type Point struct {
	x int
	y int
	i int
	g Gradient
}

type ColorBuffer struct {
	buffers [256][]Point
	index   int

	lock sync.Mutex
}

func (cb *ColorBuffer) indexBuffer(i int) int {
	return (cb.index + i) % 256
}

func (cb *ColorBuffer) AddPoint(x, y int, g Gradient) {
	cb.lock.Lock()
	for i := 0; i < 256; i++ {
		cb.buffers[cb.indexBuffer(i)] = append(cb.buffers[cb.indexBuffer(i)], Point{x, y, i, g})
	}
	cb.lock.Unlock()
}

func (cb *ColorBuffer) Tick() []Point {
	cb.lock.Lock()
	out := cb.buffers[cb.index]
	cb.buffers[cb.index] = make([]Point, 0, 256)
	cb.index++
	cb.index %= 256
	cb.lock.Unlock()
	return out
}
