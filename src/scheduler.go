package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type SearchViewable struct {
	Table *tview.Table
	Connection *tview.Form
	Fuzzy *tview.Form
}

type Viewable struct {
	Pages *tview.Pages
	Times *tview.Table
	Search SearchViewable
}

var (
	app *tview.Application = tview.NewApplication()
	viewable Viewable
	globalDB Database
)

func FindInStops(stops []Stop, s string) (ret []Stop) {
	for _, stop := range stops {
		if strings.Contains(stop.Name, s) || strings.Contains(strconv.Itoa(stop.LineNr), s) {
			ret = append(ret, stop)
		}
	}
	return
}

func FindConnections(from, to string, stops []Stop) (ret []Stop) {
	filter := func(main, substr string) bool {
		const treshold float64 = 0.9
		return IsFuzzyEqualInsens(main, substr, treshold) || // Case Insesitive
			strings.HasPrefix(main, substr) // Case Sensitive
	}
	
	for i := 0; i < len(stops); i++ {
		//		if stops[i].Name == from { // use this for exact matching
		if filter(stops[i].Name, from) {
			line := stops[i].LineNr
			dir := stops[i].Direction

			for j := i; j < len(stops); j++ {
				if line == stops[j].LineNr && dir == stops[j].Direction &&
					// to == stops[j].Name { // use this for exact matching
					filter(stops[j].Name, to) {
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
	viewable.Search.Connection = tview.NewForm()
	viewable.Search.Fuzzy = tview.NewForm()
	input = tview.NewFlex()
	
	showConnectionResults := func(from, to string) {
		nstops := FindConnections(from, to, globalDB.Stops)
		refreshTable(nstops)
	}

	from := ""
	to := ""
	captureFrom := func(text string) {
		from = text
		if len(to) != 0 {
			showConnectionResults(from, to)
		}
	}

	captureTo := func(text string) {
		to = text
		if len(from) != 0 {
			showConnectionResults(from, to)
		}
	}

	fuzzyTerm := ""
	showFuzzyResults := func() {
		nstops := FindInStops(globalDB.Stops, fuzzyTerm)
		refreshTable(nstops)
	}
	
	captureFuzzy := func(text string) {
		fuzzyTerm = text
		showFuzzyResults()
	}

	viewable.Search.Connection.
	AddInputField("From", "", 20, nil, captureFrom).
	AddInputField("To", "", 20, nil, captureTo)

	viewable.Search.Fuzzy.
	AddInputField("Fuzzy search for", "", 20, nil, captureFuzzy)

	viewable.Search.Connection.SetBorder(true).
		SetTitle("Connection form").
		SetTitleAlign(tview.AlignLeft)
	viewable.Search.Fuzzy.SetBorder(true).
		SetTitle("Fuzzy form").
		SetTitleAlign(tview.AlignLeft)

	input.
	AddItem(viewable.Search.Connection, 0, 1, true).
	AddItem(viewable.Search.Fuzzy, 0, 1, true)

	return
}

const (
	BeyondSchedule = -1
	NotWorkDays = -2
)

func IntOrPanic(str string) (result int) {
	result, err := strconv.Atoi(str)
	if err != nil {
		panic(err)
	}

	return
}
	
func CurrentHourIndex(current int, stopHours []string) int {
	for i, stopHour := range stopHours {
		if IntOrPanic(stopHour) >= current {
			return i
		}
	}

	return -1
}

func TodaysMins(now time.Time, mins Times) []string {
	switch nowDay := now.Weekday(); nowDay {
	case time.Sunday:
		return mins.HolidayMins
	case time.Saturday:
		return mins.SaturdayMins
	default:
		return mins.WorkMins
	}
}

func MinsToNextBus(stop Stop) (result int) {
	now := time.Now()
	nowHour, nowMin, _ := now.Clock()

	filterLetters := func(r rune) bool {
		return unicode.IsLetter(r)
	}

	// Check if we even fit into todays schedule
	stopHoursCount := len(stop.Times.Hours)
	latest := IntOrPanic(stop.Times.Hours[stopHoursCount - 1])
	if latest == 0 {
		latest += 24
	}

	if nowHour > latest {
		// We are no longer in todays schedule
		return BeyondSchedule
	}
	
	hoffset := CurrentHourIndex(nowHour, stop.Times.Hours)
	if hoffset == -1 {
		return BeyondSchedule
	}
	
	moffset := 0
	minHelper := func(mins []string, cmpMin int) int {
		if len(mins) == 0 {
			return -1
		}
		
		for j, stopMinute := range mins {
			tmp := strings.TrimFunc(stopMinute, filterLetters)
			if len(tmp) == 0 {
				continue
			}
			
			if IntOrPanic(tmp) >= cmpMin {
				return j
			}
		}

		return -1
	}

	lookupMins := TodaysMins(now, stop.Times)

	cmp := nowMin
	for ; hoffset < stopHoursCount; hoffset++ {
		if len(lookupMins) == 0 {
			// Doesn't drive on work days
			return NotWorkDays
		}
		
		res := minHelper(strings.Split(lookupMins[hoffset], " "), cmp)
		if res != -1 {
			moffset = res
			break
		}

		cmp = 0
	}

	if hoffset == len(stop.Times.Hours) {
		// We found the hour in this schedule
		// but we don't fit with the minutes this time
		return BeyondSchedule
	}
	
	reshour := IntOrPanic(stop.Times.Hours[hoffset])
	tmp := strings.Split(lookupMins[hoffset], " ")[moffset]
	resmins := IntOrPanic(strings.TrimFunc(tmp, filterLetters))
	
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
}

func CreateSearchPage(showTimes func(times Times)) (title string, content tview.Primitive) {
	viewable.Search.Table = tview.NewTable()

	tableFromArray := func(stops []Stop) {
		viewable.Search.Table.Clear().
		SetFixed(1, 1).
		SetSelectable(true, false).
		SetSeparator(tview.Borders.Vertical)

		headers := "Line number;Direction;Stop name;Departure in"
		for c, header := range strings.Split(headers, ";") {
			cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
			viewable.Search.Table.SetCell(0, c, cell)
		}

		for r, stop := range stops {
			cell := tview.NewTableCell(strconv.Itoa(stop.LineNr)).
			SetAlign(tview.AlignCenter).SetExpansion(1)
			viewable.Search.Table.SetCell(r + 1, 0, cell)

			cell = tview.NewTableCell(stop.Direction).
			SetAlign(tview.AlignCenter).SetExpansion(1)
			viewable.Search.Table.SetCell(r + 1, 1, cell)

			cell = tview.NewTableCell(stop.Name).
			SetAlign(tview.AlignCenter).SetExpansion(1)
			viewable.Search.Table.SetCell(r + 1, 2, cell)

			cell = tview.NewTableCell(InfoNextBus(stop)).SetAlign(tview.AlignCenter)
			viewable.Search.Table.SetCell(r + 1, 3, cell)
		}

		viewable.Search.Table.SetBorder(true).
			SetTitle("Stops and their data").
			SetTitleAlign(tview.AlignCenter)
		viewable.Search.Table.SetSelectedFunc(func(row, _ int) {
			if row != 0 {
				showTimes(stops[row - 1].Times)
			}
		})

		//	app.SetFocus(viewable.Search.Table)
	}

	tableFromArray(globalDB.Stops)

	input := CreateSearchInputFlex(tableFromArray)

	return "search", tview.NewFlex().
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(viewable.Search.Table, 0, 1, false).
			AddItem(input, 9, 0, true),
		0, 1, true)
}

func CreateTimesPage(searchAgain func()) (title string, content tview.Primitive, refresh func(times Times)) {
	viewable.Times = tview.NewTable()

	minsOrEmpty := func(mins []string, i int) (result string) {
		result = ""
		if len(mins) != 0 {
			result = mins[i]
		}

		return
	}

	refresh = func(times Times) {
		viewable.Times.Clear()
		viewable.Times.SetSelectable(true, true).SetSeparator(tview.Borders.Vertical)

		headers := "Hour;Work Day;Saturday;Holiday"
		for c, header := range strings.Split(headers, ";") {
			cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
			viewable.Times.SetCell(0, c, cell)
		}

		for r, hour := range times.Hours {
			cell := tview.NewTableCell(hour).SetAlign(tview.AlignRight).SetExpansion(1)
			viewable.Times.SetCell(r + 1, 0, cell)

			cell = tview.NewTableCell(minsOrEmpty(times.WorkMins, r)).
				SetAlign(tview.AlignLeft).
				SetExpansion(1)
			viewable.Times.SetCell(r + 1, 1, cell)
			
			cell = tview.NewTableCell(minsOrEmpty(times.SaturdayMins, r)).
				SetAlign(tview.AlignLeft).
				SetExpansion(1)
			viewable.Times.SetCell(r + 1, 2, cell)
			
			cell = tview.NewTableCell(minsOrEmpty(times.HolidayMins, r)).
				SetAlign(tview.AlignLeft).
				SetExpansion(1)
			viewable.Times.SetCell(r + 1, 3, cell)
		}

		viewable.Times.SetBorder(true).SetTitle("Departures/Arrivals").SetTitleAlign(tview.AlignCenter)
		viewable.Times.SetDoneFunc(func (key tcell.Key) {
			searchAgain()
		})
	}

	return "times", Center(80, 25, viewable.Times), refresh
}
func UpdateUncompleteTable() {
	const updateInterval = 25 * time.Millisecond
	for !globalDB.Complete {
		app.QueueUpdateDraw(func() {
			viewable.Search.Table.SetTitle("Data is now being loaded")
		})
		time.Sleep(updateInterval)
	}

	app.QueueUpdateDraw(func() {
		viewable.Search.Table.SetTitle("All data is now loaded")
	})
}

func Run() {
	ReadJson()
	go UpdateUncompleteTable()
	
	refresh := func(times Times) {}
	showTimes := func(times Times) {
		refresh(times)
		viewable.Pages.SwitchToPage("times")
	}

	backToSearch := func() {
		viewable.Pages.SwitchToPage("search")
	}

	viewable.Pages = tview.NewPages()
	name, primi, refresh := CreateTimesPage(backToSearch)
	viewable.Pages.AddPage(name, primi, true, false)
	name, primi = CreateSearchPage(showTimes)
	viewable.Pages.AddPage(name, primi, true, true)

	if err := app.SetRoot(viewable.Pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
