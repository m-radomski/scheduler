package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type Times struct {
	Hours []string `json:"hour"`
	WorkMins []string `json:"work"`
	SaturdayMins []string `json:"saturday"`
	HolidayMins []string `json:"holiday"`
}

type Stop struct {
	LineNr int `json:"line"`
	Direction string `json:"direction"`
	Name string `json:"stop_name"`
	Times Times `json:"times"`
}

var app *tview.Application = tview.NewApplication()
var stops []Stop = ReadJson()

func FindInStops(stops []Stop, s string) (ret []Stop) {
	for _, stop := range stops {
		if strings.Contains(stop.Name, s) || strings.Contains(strconv.Itoa(stop.LineNr), s) {
			ret = append(ret, stop)
		}
	}
	return
}

func FindConnections(from, to string, stops []Stop) (ret []Stop) {
	for i := 0; i < len(stops); i++ {
		if stops[i].Name == from {
			line := stops[i].LineNr
			dir := stops[i].Direction

			for j := i; j < len(stops); j++ {
				if line == stops[j].LineNr && dir == stops[j].Direction && to == stops[j].Name {
					ret = append(ret, stops[i])
				} else if line != stops[j].LineNr || dir != stops[j].Direction {
					i += j - 1 - i // skip this many stops, because the are on the same route
					break
				}
			}
		}
	}

	return
}

func Center(width, height int, p tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(tview.NewBox(), 0, 1, false).
			AddItem(p, height, 1, true).
			AddItem(tview.NewBox(), 0, 1, false), width, 1, true).
		AddItem(tview.NewBox(), 0, 1, false)
}

func CreateSearchInputFlex(refreshTable func(stops []Stop)) (input *tview.Flex) {
	connection := tview.NewForm()
	fuzzy := tview.NewForm()
	input = tview.NewFlex()

	from := ""
	captureFrom := func(text string) {
		from = text
	}

	to := ""
	captureTo := func(text string) {
		to = text
	}

	fuzzyTerm := ""
	captureFuzzy := func(text string) {
		fuzzyTerm = text
	}

	showFuzzyResults := func() {
		nstops := FindInStops(stops, fuzzyTerm)
		refreshTable(nstops)
	}

	showConnectionResults := func() {
		nstops := FindConnections(from, to, stops)
		refreshTable(nstops)
	}

	connection.
	AddInputField("From", "", 20, nil, captureFrom).
	AddInputField("To", "", 20, nil, captureTo).
	AddButton("Search", showConnectionResults).
	AddButton("Go to fuzzy search", func() {
		app.SetFocus(fuzzy)
	})

	fuzzy.
	AddInputField("Fuzzy search for", "", 20, nil, captureFuzzy).
	AddButton("Search", showFuzzyResults).
	AddButton("Go to connection search", func() {
		app.SetFocus(input)
	})

	connection.SetBorder(true).SetTitle("Connection form").SetTitleAlign(tview.AlignLeft)
	fuzzy.SetBorder(true).SetTitle("Fuzzy form").SetTitleAlign(tview.AlignLeft)

	input.
	AddItem(connection, 0, 1, true).
	AddItem(fuzzy, 0, 1, true)

	return
}

const (
	BeyondSchedule = -1
	NotWorkDays = -2
)

func MinsToNextBus(stop Stop) (result int) {
	now := time.Now()
	nowHour, nowMin, _ := now.Clock()
	stopHoursCount := len(stop.Times.Hours)

	intOrPanic := func(str string) (result int) {
		result, err := strconv.Atoi(str)
		if err != nil {
			panic(err)
		}

		return
	}
	
	filterLetters := func(r rune) bool {
		return unicode.IsLetter(r)
	}

	// Check if we even fit into todays schedule
	latest := intOrPanic(stop.Times.Hours[stopHoursCount - 1])
	if latest == 0 {
		latest += 24
	}

	if nowHour > latest {
		// We are no longer in todays schedule
		return BeyondSchedule
	}

	hoffset := 0
	for i, stopHour := range stop.Times.Hours {
		if intOrPanic(stopHour) >= nowHour {
			hoffset = i
			break
		}
	}

	moffset := 0
	minHelper := func(mins []string, cmpMin int) int {
		if len(mins) == 0 {
		}
		
		for j, stopMinute := range mins {
			stopMinuteInt := 0
			tmp := strings.TrimFunc(stopMinute, filterLetters)
			if len(tmp) != 0 {
				stopMinuteInt = intOrPanic(tmp)
			} else {
				continue
			}
			
			if stopMinuteInt >= cmpMin {
				return j
			}
		}

		return -1
	}

	cmp := nowMin
	for i := hoffset; i < stopHoursCount; i++ {
		if len(stop.Times.WorkMins) != 0 {
			res := minHelper(strings.Split(stop.Times.WorkMins[hoffset], " "), cmp)
			if res != -1 {
				moffset = res
				break
			}
		} else {
			// Doesn't drive on work days
			return NotWorkDays
		}

		cmp = 0
		hoffset += 1
	}

	if hoffset == len(stop.Times.Hours) {
		// We found the hour in this schedule
		// but we don't fit with the minutes this time
		return BeyondSchedule
	}
	
	reshour := intOrPanic(stop.Times.Hours[hoffset])
	tmp := strings.Split(stop.Times.WorkMins[hoffset], " ")[moffset]
	resmins := intOrPanic(strings.TrimFunc(tmp, filterLetters))
	
	return (reshour - nowHour) * 60 + resmins - nowMin
}

func InfoNextBus(stop Stop) (result string) {
	switch minNext := MinsToNextBus(stop); minNext {
	case BeyondSchedule:
		return "Beyond schedule"
	case NotWorkDays:
		return "Doesn't drive today"
	default:
		return fmt.Sprintln("Next in", minNext, "min")
	}

	return
}

func CreateSearchPage(showTimes func(times Times)) (title string, content tview.Primitive) {
	table := tview.NewTable()

	tableFromArray := func(stops []Stop) {
		table.Clear().
		SetFixed(1, 1).
		SetSelectable(true, false).
		SetSeparator(tview.Borders.Vertical)

		headers := "Line number;Direction;Stop name;Departure in"
		for c, header := range strings.Split(headers, ";") {
			cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
			table.SetCell(0, c, cell)
		}

		for r, stop := range stops {
			cell := tview.NewTableCell(strconv.Itoa(stop.LineNr)).
			SetAlign(tview.AlignCenter).SetExpansion(1)
			table.SetCell(r + 1, 0, cell)

			cell = tview.NewTableCell(stop.Direction).
			SetAlign(tview.AlignCenter).SetExpansion(1)
			table.SetCell(r + 1, 1, cell)

			cell = tview.NewTableCell(stop.Name).
			SetAlign(tview.AlignCenter).SetExpansion(1)
			table.SetCell(r + 1, 2, cell)

			cell = tview.NewTableCell(InfoNextBus(stop)).SetAlign(tview.AlignCenter)
			table.SetCell(r + 1, 3, cell)
		}

		table.SetBorder(true).SetTitle("Stops and their data").SetTitleAlign(tview.AlignCenter)
		table.SetSelectedFunc(func(row, _ int) {
			if row != 0 {
				showTimes(stops[row - 1].Times)
			}
		})

		app.SetFocus(table)
	}

	tableFromArray(stops)

	input := CreateSearchInputFlex(tableFromArray)

	return "search", tview.NewFlex().
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(table, 0, 1, false).
			AddItem(input, 9, 0, true),
		0, 1, true)
}

func CreateTimesPage(searchAgain func()) (title string, content tview.Primitive, refresh func(times Times)) {
	table := tview.NewTable()

	minsOrEmpty := func(mins []string, i int) (result string) {
		result = ""
		if len(mins) != 0 {
			result = mins[i]
		}

		return
	}

	refresh = func(times Times) {
		table.Clear()
		table.SetSelectable(true, true).SetSeparator(tview.Borders.Vertical)

		headers := "Hour;Work Day;Saturday;Holiday"
		for c, header := range strings.Split(headers, ";") {
			cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
			table.SetCell(0, c, cell)
		}

		for r, hour := range times.Hours {
			cell := tview.NewTableCell(hour).SetAlign(tview.AlignRight).SetExpansion(1)
			table.SetCell(r + 1, 0, cell)

			cell = tview.NewTableCell(minsOrEmpty(times.WorkMins, r)).
				SetAlign(tview.AlignLeft).
				SetExpansion(1)
			table.SetCell(r + 1, 1, cell)
			
			cell = tview.NewTableCell(minsOrEmpty(times.SaturdayMins, r)).
				SetAlign(tview.AlignLeft).
				SetExpansion(1)
			table.SetCell(r + 1, 2, cell)
			
			cell = tview.NewTableCell(minsOrEmpty(times.HolidayMins, r)).
				SetAlign(tview.AlignLeft).
				SetExpansion(1)
			table.SetCell(r + 1, 3, cell)
		}

		table.SetBorder(true).SetTitle("Departures/Arrivals").SetTitleAlign(tview.AlignCenter)
		table.SetDoneFunc(func (key tcell.Key) {
			searchAgain()
		})
	}

	return "times", Center(80, 25, table), refresh
}

func main() {
	pages := tview.NewPages()

	refresh := func(times Times) {}
	dummy := func(times Times) {
		refresh(times)
		pages.SwitchToPage("times")
	}

	backToSearch := func() {
		pages.SwitchToPage("search")
	}

	name, primi, refresh := CreateTimesPage(backToSearch)
	pages.AddPage(name, primi, true, false)
	name, primi = CreateSearchPage(dummy)
	pages.AddPage(name, primi, true, true)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
