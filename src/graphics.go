package scheduler

import (
	"time"
	"strings"
	
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type SearchFocused int

const (
	ConnectionFocused SearchFocused = iota
	FuzzyFocused
	TableFocused
)

type SearchViewable struct {
	Table *tview.Table
	Connection *tview.Form
	Fuzzy *tview.Form
	CurrentFocus SearchFocused
}

type Viewable struct {
	Pages *tview.Pages
	Times *tview.Table
	Search SearchViewable
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

func (searchView *SearchViewable)CreateSearchInputFlex() (input *tview.Flex) {
	searchView.Connection = tview.NewForm()
	searchView.Fuzzy = tview.NewForm()
	input = tview.NewFlex()
	
	showConnectionResults := func(from, to string) {
		nstops := FindConnections(from, to, globalDB.Stops)
		searchView.PopulateConnectionsTable(nstops)
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
		searchEntires := SearchEntriesFromStops(nstops)
		searchView.PopulateSearchTable(searchEntires)
	}
	
	captureFuzzy := func(text string) {
		fuzzyTerm = text
		showFuzzyResults()
	}

	searchView.Connection.
	AddInputField("From", "", 20, nil, captureFrom).
	AddInputField("To", "", 20, nil, captureTo)

	searchView.Fuzzy.
	AddInputField("Fuzzy search for", "", 20, nil, captureFuzzy)

	searchView.Connection.SetBorder(true).
		SetTitle("Connection form").
		SetTitleAlign(tview.AlignLeft)
	searchView.Fuzzy.SetBorder(true).
		SetTitle("Fuzzy form").
		SetTitleAlign(tview.AlignLeft)

	input.
	AddItem(searchView.Connection, 0, 1, true).
	AddItem(searchView.Fuzzy, 0, 1, true)

	return
}

func (searchView *SearchViewable) PopulateSearchTable(entires []SearchEntry) {
	searchView.Table.Clear()

	headers := "Line number;Direction;Stop name;Departure in"
	for c, header := range strings.Split(headers, ";") {
		cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
		searchView.Table.SetCell(0, c, cell)
	}

	for r, entry := range entires {
		cell := tview.NewTableCell(entry.LineNr).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		searchView.Table.SetCell(r + 1, 0, cell)

		cell = tview.NewTableCell(entry.Direction).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		searchView.Table.SetCell(r + 1, 1, cell)

		cell = tview.NewTableCell(entry.StopName).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		searchView.Table.SetCell(r + 1, 2, cell)

		cell = tview.NewTableCell(entry.InfoNext).SetAlign(tview.AlignCenter)
		searchView.Table.SetCell(r + 1, 3, cell)
	}
}

func (searchView *SearchViewable) PopulateConnectionsTable(connections []Connection) {
	searchView.Table.Clear()

	headers := "Line number;Direction;Departure in"
	for c, header := range strings.Split(headers, ";") {
		cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
		searchView.Table.SetCell(0, c, cell)
	}

	for r, connection := range connections {
		cell := tview.NewTableCell(connection.LineNr).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		searchView.Table.SetCell(r + 1, 0, cell)

		cell = tview.NewTableCell(connection.Direction).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		searchView.Table.SetCell(r + 1, 1, cell)

		cell = tview.NewTableCell(connection.InfoNext).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		searchView.Table.SetCell(r + 1, 2, cell)
	}	
}

func (view *Viewable) CreateSearchPage() (title string, content tview.Primitive) {
	view.Search.Table = tview.NewTable()

	view.Search.Table.SetFixed(1, 1).
		SetSelectable(true, false).
		SetSeparator(tview.Borders.Vertical)

	view.Search.Table.SetBorder(true).
		SetTitle("Stops and their data").
		SetTitleAlign(tview.AlignCenter)
	view.Search.Table.SetSelectedFunc(func(row, _ int) {
		if row != 0 {
			view.RefreshTimesTable(globalDB.Stops[row - 1].Times)
			view.Pages.SwitchToPage("times")
		}
	})

	searchEntires := SearchEntriesFromStops(globalDB.Stops)
	view.Search.PopulateSearchTable(searchEntires)

	input := view.Search.CreateSearchInputFlex()

	return "search", tview.NewFlex().
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(view.Search.Table, 0, 1, false).
			AddItem(input, 9, 0, true),
		0, 1, true)
}

func (view *Viewable) CreateTimesPage() (title string, content tview.Primitive) {
	view.Times = tview.NewTable()

	view.Times.SetSelectable(true, true).SetSeparator(tview.Borders.Vertical)	
	view.Times.SetBorder(true).SetTitle("Departures/Arrivals").SetTitleAlign(tview.AlignCenter)
	view.Times.SetDoneFunc(func (key tcell.Key) {
		view.Pages.SwitchToPage("search")
	})
	
	return "times", Center(80, 25, view.Times)
}

func (view *Viewable) CreatePages() {
	view.Pages = tview.NewPages()
	
	name, primi := view.CreateTimesPage()
	view.Pages.AddPage(name, primi, true, false)
	
	name, primi = view.CreateSearchPage()
	view.Pages.AddPage(name, primi, true, true)
	view.InitChangingFocus()
}

func (view *Viewable) InitChangingFocus() {
	app.SetInputCapture(func (event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlR {
			RefreshJson()
		}
		
		if name, _ := view.Pages.GetFrontPage(); name != "search" {
			return event
		}

		if event.Key() == tcell.KeyCtrlSpace {
			view.Search.FocusNext()
		}

		return event
	})	
}

func (searchView *SearchViewable) FocusNext() {
	switch searchView.CurrentFocus {
	case ConnectionFocused:
		app.SetFocus(searchView.Fuzzy)
		searchView.CurrentFocus += 1
	case FuzzyFocused:
		app.SetFocus(searchView.Table)
		searchView.CurrentFocus += 1
	case TableFocused:		
		app.SetFocus(searchView.Connection)
		searchView.CurrentFocus = ConnectionFocused
	}
}

func (view *Viewable) RefreshTimesTable(times Times) {
	view.Times.Clear()
	
	minsOrEmpty := func(mins []string, i int) (result string) {
		result = ""
		if len(mins) != 0 {
			result = mins[i]
		}

		return
	}

	headers := "Hour;Work Day;Saturday;Holiday"
	for c, header := range strings.Split(headers, ";") {
		cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
		view.Times.SetCell(0, c, cell)
	}

	for r, hour := range times.Hours {
		cell := tview.NewTableCell(hour).SetAlign(tview.AlignRight).SetExpansion(1)
		view.Times.SetCell(r + 1, 0, cell)

		cell = tview.NewTableCell(minsOrEmpty(times.WorkMins, r)).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)
		view.Times.SetCell(r + 1, 1, cell)
		
		cell = tview.NewTableCell(minsOrEmpty(times.SaturdayMins, r)).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)
		view.Times.SetCell(r + 1, 2, cell)
		
		cell = tview.NewTableCell(minsOrEmpty(times.HolidayMins, r)).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)
		view.Times.SetCell(r + 1, 3, cell)
	}

}

// For corectness sake this is not good because it's the only place where we use
// the globaly defined `viewable` object. In order to replace it I would need some
// way of handling events or a functions that just waits for something to happen and
// then does something which seems unpractical or even slow. Let's just leave it
// like it is, hoping it won't bite us
func UpdateUncompleteTable() {
	const updateInterval = 25 * time.Millisecond
	for !globalDB.Complete {
		app.QueueUpdateDraw(func() {
			viewable.Search.Table.SetTitle("Data is now being loaded")
			searchEntires := SearchEntriesFromStops(globalDB.Stops)
			viewable.Search.PopulateSearchTable(searchEntires)
		})
		time.Sleep(updateInterval)
	}

	app.QueueUpdateDraw(func() {
		viewable.Search.Table.SetTitle("All data is now loaded")
	})
}
