// Command typer is a game for practicing typing.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell"
	"github.com/mattn/go-runewidth"
)

func main() {
	screen, style := setup()
	defer func() {
		// always end properly, even on panic
		screen.Fini()
	}()
	screen.Show()

	var next Mode = &HelpView{screen, style}
	for {
		next = next.Run()
		if next == nil {
			break
		}
	}
}

type GameView struct {
	screen tcell.Screen
	style  tcell.Style
}

func (me *GameView) Run() Mode {
	screen, style := me.screen, me.style
	w, h := screen.Size()

	wordCount := 0
	pos := Position{y: h/2 + 1} // input start
	index := 0                  // position in text

	text := randomText()

	drawLines(screen, style)
	clearDisplay(screen, style)
	fillText(screen, style, 0, 0, text)
	screen.ShowCursor(pos.x, pos.y)
	rtext := []rune(text)
	puts(screen, style.Foreground(tcell.ColorYellow), 0, h-1, "Start typing")
	screen.Sync()

	var start time.Time
	var started bool
	long := longestWord()
	for {
		switch ev := screen.PollEvent().(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				return nil
			case tcell.KeyCtrlD:
				clearInput(screen, style)
				screen.HideCursor()
				return &HelpView{screen, style}

			case tcell.KeyBackspace2:
				pos.x--
				if pos.x < 0 {
					pos.x = 0
				}
				puts(screen, style, pos.x, pos.y, string(ev.Rune()))
				index--
			case tcell.KeyEnter:
				pos.y++
				pos.x = 0
			default:
				r := ev.Rune()
				putexp(screen, style, pos.x, pos.y, r, rtext[index] == r)
				index++
				pos.x++
				if unicode.IsSpace(r) {
					wordCount++
					drawProgress(screen, style, wordCount, start)
					if pos.x >= w-long { // next line
						pos.y++
						pos.x = 0
					}
				}
			}
			screen.ShowCursor(pos.x, pos.y)
			screen.Sync()
			if index == len(rtext) {
				// game over
				screen.HideCursor()
				return &GameOver{screen, style}
			}
			if !started {
				start = time.Now()
				started = true
				hline := strings.Repeat(" ", h)
				puts(screen, style, 0, h-1, hline)
			}
		case *tcell.EventResize:
			screen.Sync()
		}
	}

	return nil
}

// ----------------------------------------

type GameOver struct {
	screen tcell.Screen
	style  tcell.Style
}

func (me *GameOver) Run() Mode {
	screen, style := me.screen, me.style
	centerText(screen, style.Foreground(tcell.ColorYellow), 3, ` Game Over 
 Press ENTER to continue. 
                          `)
	screen.Sync()
	for {
		switch ev := screen.PollEvent().(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				return nil
			case tcell.KeyEnter:
				clearDisplay(screen, style)
				clearInput(screen, style)
				return &HelpView{screen, style}
			}
		case *tcell.EventResize:
			screen.Sync()
		}
	}
	return nil
}

// ----------------------------------------

type HelpView struct {
	screen tcell.Screen
	style  tcell.Style
}

func (me *HelpView) Run() Mode {
	screen, style := me.screen, me.style
	screen.Clear()
	centerText(screen, style, 1, help)
	screen.Sync()
	for {
		switch ev := screen.PollEvent().(type) {
		case *tcell.EventKey:
			switch ev.Key() {
			case tcell.KeyCtrlC:
				return nil
			case tcell.KeyCtrlN:
				return &GameView{screen, style}
			}
		case *tcell.EventResize:
			screen.Sync()
		}
	}
	return nil
}

type Mode interface {
	Run() Mode
}

const help = `
New game    Ctrl-n
Stop game   Ctrl-d
Quit        Ctrl-c`

// ----------------------------------------

func setup() (tcell.Screen, tcell.Style) {
	// Add all characters in the world
	//encoding.Register()

	screen, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := screen.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	var style tcell.Style
	screen.SetStyle(style)

	return screen, style
}

type Position struct {
	x int
	y int
}

func clearDisplay(screen tcell.Screen, style tcell.Style) {
	w, h := screen.Size()
	hline := strings.Repeat(" ", w)
	for y := 0; y < h/2; y++ {
		puts(screen, style, 0, y, hline)
	}
}

func clearInput(screen tcell.Screen, style tcell.Style) {
	w, h := screen.Size()
	hline := strings.Repeat(" ", w)
	for y := h/2 + 1; y < h-2; y++ {
		puts(screen, style, 0, y, hline)
	}
}

func drawProgress(s tcell.Screen, style tcell.Style, wordCount int, start time.Time) {
	_, h := s.Size()
	line := fmt.Sprintf("%v word/min, %v", wpm(wordCount, start), time.Since(start))
	puts(s, style, 0, h-1, line)
}

func wpm(words int, start time.Time) int {
	sec := time.Since(start).Seconds()
	return int(float64(words*60) / sec)

}

func drawLines(s tcell.Screen, style tcell.Style) {
	w, h := s.Size()
	hline := strings.Repeat(string(tcell.RuneHLine), w)
	puts(s, style, 0, h/2, hline)
	puts(s, style, 0, h-2, hline)
}

func drawCursor(screen tcell.Screen, style tcell.Style, pos Position) {
	screen.SetCell(pos.x, pos.y, style, '_')
}

func centerText(screen tcell.Screen, style tcell.Style, y int, text string) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	w, _ := screen.Size()
	row := y
	for scanner.Scan() {
		line := scanner.Text()
		x := w/2 - len(line)/2
		puts(screen, style, x, row, line)
		row++
	}
}

func fillText(screen tcell.Screen, style tcell.Style, x, y int, text string) {
	w, _ := screen.Size()
	r := bufio.NewReader(strings.NewReader(text))
	lx, ly := x, y
	for {
		word, err := r.ReadString(' ')
		width := len(word)
		if lx+width >= w-1 {
			// next row
			lx = x
			ly++
		}
		puts(screen, style, lx, ly, word)
		lx = lx + width
		if err == io.EOF {
			return
		}
	}
}

func drawText(s tcell.Screen, style tcell.Style, x, y int, text string) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	row := y
	for scanner.Scan() {
		line := scanner.Text()
		puts(s, style, x, row, line)
		row++
	}
}

func putexp(s tcell.Screen, style tcell.Style, x, y int, r rune, exp bool) {
	if !exp {
		style = style.Foreground(tcell.ColorRed)
	}
	puts(s, style, x, y, string(r))
}

func puts(s tcell.Screen, style tcell.Style, x, y int, str string) {
	i := 0
	var deferred []rune
	dwidth := 0
	zwj := false
	for _, r := range str {
		if r == '\u200d' {
			if len(deferred) == 0 {
				deferred = append(deferred, ' ')
				dwidth = 1
			}
			deferred = append(deferred, r)
			zwj = true
			continue
		}
		if zwj {
			deferred = append(deferred, r)
			zwj = false
			continue
		}
		switch runewidth.RuneWidth(r) {
		case 0:
			if len(deferred) == 0 {
				deferred = append(deferred, ' ')
				dwidth = 1
			}
		case 1:
			if len(deferred) != 0 {
				s.SetContent(x+i, y, deferred[0], deferred[1:], style)
				i += dwidth
			}
			deferred = nil
			dwidth = 1
		case 2:
			if len(deferred) != 0 {
				s.SetContent(x+i, y, deferred[0], deferred[1:], style)
				i += dwidth
			}
			deferred = nil
			dwidth = 2
		}
		deferred = append(deferred, r)
	}
	if len(deferred) != 0 {
		s.SetContent(x+i, y, deferred[0], deferred[1:], style)
		i += dwidth
	}
}
