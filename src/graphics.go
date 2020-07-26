package scheduler

import (
	"strings"
	"strconv"
	
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

func CreateSearchPage() (title string, content tview.Primitive) {
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
				viewable.RefreshTimesTable(stops[row - 1].Times)
				viewable.Pages.SwitchToPage("times")
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

func CreateTimesPage() (title string, content tview.Primitive) {
	viewable.Times = tview.NewTable()

	viewable.Times.SetSelectable(true, true).SetSeparator(tview.Borders.Vertical)	
	viewable.Times.SetBorder(true).SetTitle("Departures/Arrivals").SetTitleAlign(tview.AlignCenter)
	viewable.Times.SetDoneFunc(func (key tcell.Key) {
		viewable.Pages.SwitchToPage("search")
	})
	
	return "times", Center(80, 25, viewable.Times)
}

func (view *Viewable) CreatePages() {
	view.Pages = tview.NewPages()
	
	name, primi := CreateTimesPage()
	view.Pages.AddPage(name, primi, true, false)
	
	name, primi = CreateSearchPage()
	view.Pages.AddPage(name, primi, true, true)
	view.InitChangingFocus()
}

func (view *Viewable) InitChangingFocus() {
	app.SetInputCapture(func (event *tcell.EventKey) *tcell.EventKey {
		if name, _ := view.Pages.GetFrontPage(); name != "search" {
			return event
		}

		if event.Key() == tcell.KeyCtrlSpace {
			view .Search.FocusNext()
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
