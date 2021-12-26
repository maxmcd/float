package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gliderlabs/ssh"
)

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
		defer fmt.Println("Closed         ->", id)

		screen, err := tcell.NewTerminfoScreenFromTty(newSSHSessionTTYWrapper(s))
		if err != nil {
			fmt.Println("error initializing ", err)
			fmt.Fprintln(s, err.Error())
			_ = s.Exit(1)
			return
		}

		if err := f.Run(s.Context(), screen); err != nil {
			fmt.Println("error running", err)
			fmt.Fprintln(s, err.Error())
			_ = s.Exit(1)
			return
		}
		fmt.Fprintln(s, "     ......                 ")
		fmt.Fprintln(s, "         ......             ")
		fmt.Fprintln(s, "Goodbye, thanks for floating")
		fmt.Fprintln(s, "               ......       ")
		fmt.Fprintln(s, "                   ......   ")

	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(time.Millisecond * 1000 / 60).C
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
	options := []ssh.Option{}
	// Provide a key if we have it at a special location
	if _, err := os.Stat("/opt/id_rsa"); err == nil {
		options = append(options, ssh.HostKeyFile("/opt/id_rsa"))
	}
	log.Fatal(ssh.ListenAndServe(addr, nil, options...))
}

func (f Float) Run(ctx context.Context, screen tcell.Screen) error {
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

	for y, line := range []string{
		"Welcome to float.",
		"Float is a shared experience, if others join you'll",
		"see what they draw and they'll see what you draw.",
		"",
		"https://float.maxmcd.com or 'ssh float.maxmcd.com'",
		"",
		"Spacebar to change colors",
	} {
		for x, c := range line {
			screen.SetContent(x, y, c, nil, defStyle)
		}
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
		case <-ctx.Done():
			goto END
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
					// Space changes color
					color = rand.Intn(len(gradients))
				}
				if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
					goto END
				}
			}
		}
	}

END:
	quit <- struct{}{}
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
