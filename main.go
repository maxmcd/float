package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gliderlabs/ssh"
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

type Float struct {
	cb          *ColorBuffer
	multisetter *ScreenMultisetter
}

func (f Float) Start() {
	f.cb = &ColorBuffer{}
	f.multisetter = &ScreenMultisetter{screens: make(map[int]tcell.Screen)}

	ssh.Handle(func(s ssh.Session) {
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
		fmt.Fprintln(s, "Goodbye, thanks for floating")
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
								linearGradient(
									float64(p.i),
								),
							),
						),
					)
				}
			}
		}
	}()
	_ = cancel
	log.Fatal(ssh.ListenAndServe(":2222", nil))
}

func (f Float) Run(screen tcell.Screen) error {
	if err := screen.Init(); err != nil {
		return err
	}

	// Set default text style
	defStyle := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	screen.SetStyle(defStyle)

	// Clean canvas
	screen.Clear()
	screen.EnableMouse()

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
				f.cb.AddPoint(ev.Position())
			case *tcell.EventResize:
				screen.Sync()
			case *tcell.EventKey:
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
	fmt.Println("exiting")
	screen.Fini() // end
	fmt.Println("fini")
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
	cb.buffers[cb.index] = make([]Point, 0, 256)
	cb.index++
	cb.index %= 256
	cb.lock.Unlock()
	return out
}
