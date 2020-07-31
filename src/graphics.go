package scheduler

import (
	"time"
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

type UI struct {
	Pages *tview.Pages
	Times *tview.Table
	SearchTable *tview.Table
	SearchConnection *tview.Form
	SearchFuzzy *tview.Form
	CurrentFocus SearchFocused
}

func NewUI() UI {
	return UI {
	}
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

func (ui *UI) CreateSearchInputFlex(database *Database) (input *tview.Flex) {
	ui.SearchConnection = tview.NewForm()
	ui.SearchFuzzy = tview.NewForm()
	input = tview.NewFlex()
	
	showConnectionResults := func(from, to string) {
		nstops := FindConnections(from, to, database.Stops)
		ui.PopulateConnectionsTable(nstops)
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
		nstops := FindInStops(database.Stops, fuzzyTerm)
		searchEntires := SearchEntriesFromStops(nstops)
		ui.PopulateSearchTable(searchEntires)
	}
	
	captureFuzzy := func(text string) {
		fuzzyTerm = text
		showFuzzyResults()
	}

	ui.SearchConnection.
	AddInputField("From", "", 20, nil, captureFrom).
	AddInputField("To", "", 20, nil, captureTo)

	ui.SearchFuzzy.
	AddInputField("Fuzzy search for", "", 20, nil, captureFuzzy)

	ui.SearchConnection.SetBorder(true).
		SetTitle("Connection form").
		SetTitleAlign(tview.AlignLeft)
	ui.SearchFuzzy.SetBorder(true).
		SetTitle("Fuzzy form").
		SetTitleAlign(tview.AlignLeft)

	input.
	AddItem(ui.SearchConnection, 0, 1, true).
	AddItem(ui.SearchFuzzy, 0, 1, true)

	return
}

func (ui *UI) PopulateSearchTable(entries []SearchEntry) {
	ui.SearchTable.Clear()

	headers := "Line number;Direction;Stop name;Departure in"
	for c, header := range strings.Split(headers, ";") {
		cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(0, c, cell)
	}

	for r, entry := range entries {
		cell := tview.NewTableCell(entry.LineNr).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(r + 1, 0, cell)

		cell = tview.NewTableCell(entry.Direction).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(r + 1, 1, cell)

		cell = tview.NewTableCell(entry.StopName).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(r + 1, 2, cell)

		cell = tview.NewTableCell(entry.InfoNext).SetAlign(tview.AlignCenter)
		ui.SearchTable.SetCell(r + 1, 3, cell)
	}
}

func (ui *UI) PopulateConnectionsTable(connections []Connection) {
	ui.SearchTable.Clear()

	headers := "Line number;Direction;Departure in"
	for c, header := range strings.Split(headers, ";") {
		cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(0, c, cell)
	}

	for r, connection := range connections {
		cell := tview.NewTableCell(strconv.Itoa(connection.Stop.LineNr)).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(r + 1, 0, cell)

		cell = tview.NewTableCell(connection.Path).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(r + 1, 1, cell)

		cell = tview.NewTableCell(connection.InfoNext).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(r + 1, 2, cell)
	}	
}

func (view *UI) CreateSearchPage(database *Database) (title string, content tview.Primitive) {
	view.SearchTable = tview.NewTable()

	view.SearchTable.SetFixed(1, 1).
		SetSelectable(true, false).
		SetSeparator(tview.Borders.Vertical)

	view.SearchTable.SetBorder(true).
		SetTitle("Stops and their data").
		SetTitleAlign(tview.AlignCenter)
	view.SearchTable.SetSelectedFunc(func(row, _ int) {
		if row != 0 {
			view.RefreshTimesTable(database.Stops[row - 1].Times)
			view.Pages.SwitchToPage("times")
		}
	})

	searchEntires := SearchEntriesFromStops(database.Stops)
	view.PopulateSearchTable(searchEntires)

	input := view.CreateSearchInputFlex(database)

	return "search", tview.NewFlex().
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(view.SearchTable, 0, 1, false).
			AddItem(input, 9, 0, true),
		0, 1, true)
}

func (ui *UI) CreateTimesPage() (title string, content tview.Primitive) {
	ui.Times = tview.NewTable()

	ui.Times.SetSelectable(true, true).SetSeparator(tview.Borders.Vertical)	
	ui.Times.SetBorder(true).SetTitle("Departures/Arrivals").SetTitleAlign(tview.AlignCenter)
	ui.Times.SetDoneFunc(func (key tcell.Key) {
		ui.Pages.SwitchToPage("search")
	})
	
	return "times", Center(80, 25, ui.Times)
}

func (ui *UI) CreatePages(database *Database) {
	ui.Pages = tview.NewPages()
	
	name, primi := ui.CreateTimesPage()
	ui.Pages.AddPage(name, primi, true, false)
	
	name, primi = ui.CreateSearchPage(database)
	ui.Pages.AddPage(name, primi, true, true)
	ui.InitChangingFocus(database)
}

func (ui *UI) InitChangingFocus(database *Database) {
	app.SetInputCapture(func (event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlR {
			RefreshJson(database)
		}
		
		if name, _ := ui.Pages.GetFrontPage(); name != "search" {
			return event
		}

		if event.Key() == tcell.KeyCtrlSpace {
			ui.SearchFocusNext()
		}

		return event
	})	
}

func (ui *UI) SearchFocusNext() {
	switch ui.CurrentFocus {
	case ConnectionFocused:
		app.SetFocus(ui.SearchFuzzy)
		ui.CurrentFocus += 1
	case FuzzyFocused:
		app.SetFocus(ui.SearchTable)
		ui.CurrentFocus += 1
	case TableFocused:		
		app.SetFocus(ui.SearchConnection)
		ui.CurrentFocus = ConnectionFocused
	}
}

func (ui *UI) RefreshTimesTable(times Times) {
	ui.Times.Clear()
	
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
		ui.Times.SetCell(0, c, cell)
	}

	for r, hour := range times.Hours {
		cell := tview.NewTableCell(hour).SetAlign(tview.AlignRight).SetExpansion(1)
		ui.Times.SetCell(r + 1, 0, cell)

		cell = tview.NewTableCell(minsOrEmpty(times.WorkMins, r)).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)
		ui.Times.SetCell(r + 1, 1, cell)
		
		cell = tview.NewTableCell(minsOrEmpty(times.SaturdayMins, r)).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)
		ui.Times.SetCell(r + 1, 2, cell)
		
		cell = tview.NewTableCell(minsOrEmpty(times.HolidayMins, r)).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)
		ui.Times.SetCell(r + 1, 3, cell)
	}

}

// For corectness sake this is not good because it's the only place where we use
// the globaly defined `UI` object. In order to replace it I would need some
// way of handling events or a functions that just waits for something to happen and
// then does something which seems unpractical or even slow. Let's just leave it
// like it is, hoping it won't bite us
func (ui *UI) UpdateUncompleteTable(database *Database) {
	const updateInterval = 25 * time.Millisecond
	for !database.Complete {
		app.QueueUpdateDraw(func() {
			ui.SearchTable.SetTitle("Data is now being loaded")
			searchEntires := SearchEntriesFromStops(database.Stops)
			ui.PopulateSearchTable(searchEntires)
		})
		time.Sleep(updateInterval)
	}

	app.QueueUpdateDraw(func() {
		ui.SearchTable.SetTitle("All data is now loaded")
	})
}
