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
	TimesBanner *tview.Table
	TimesConnectionId int
	
	SearchTable *tview.Table
	SearchConnection *tview.Form
	SearchFuzzy *tview.Form
	CurrentFocus SearchFocused

	ConnectionsDisplayed []Connection
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
		} else if len(from) == 0 {
			connections := ConnectionsFromStops(database.Stops)
			ui.PopulateSearchTable(connections)
			ui.SearchTable.ScrollToBeginning()
		}
	}

	captureTo := func(text string) {
		to = text
		if len(from) != 0 {
			showConnectionResults(from, to)
		} else if len(to) == 0 {
			connections := ConnectionsFromStops(database.Stops)
			ui.PopulateSearchTable(connections)
			ui.SearchTable.ScrollToBeginning()
		}
	}

	fuzzyTerm := ""
	showFuzzyResults := func() {
		nstops := FindInStops(database.Stops, fuzzyTerm)
		connections := ConnectionsFromStops(nstops)
		ui.PopulateSearchTable(connections)
	}
	
	captureFuzzy := func(text string) {
		fuzzyTerm = text
		if len(fuzzyTerm) != 0 {
			showFuzzyResults()
		} else {
			connections := ConnectionsFromStops(database.Stops)
			ui.PopulateSearchTable(connections)
			ui.SearchTable.ScrollToBeginning()
		}
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

func (ui *UI) PopulateSearchTable(connections []Connection) {
	ui.SearchTable.Clear()
	ui.ConnectionsDisplayed = connections

	headers := "Line number;Direction;Stop name;Departure in"
	for c, header := range strings.Split(headers, ";") {
		cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(0, c, cell)
	}

	for r, connection := range connections {
		cell := tview.NewTableCell(strconv.Itoa(connection.Stop.LineNr)).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(r + 1, 0, cell)

		cell = tview.NewTableCell(connection.Stop.Direction).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(r + 1, 1, cell)

		cell = tview.NewTableCell(connection.Stop.Name).
			SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.SearchTable.SetCell(r + 1, 2, cell)

		cell = tview.NewTableCell(connection.InfoNext).SetAlign(tview.AlignCenter)
		ui.SearchTable.SetCell(r + 1, 3, cell)
	}
}

func (ui *UI) PopulateConnectionsTable(connections []Connection) {
	ui.SearchTable.Clear()
	ui.ConnectionsDisplayed = connections

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

func (ui *UI) CreateSearchPage(database *Database) (title string, content tview.Primitive) {
	ui.SearchTable = tview.NewTable()

	ui.SearchTable.SetFixed(1, 1).
		SetSelectable(true, false).
		SetSeparator(tview.Borders.Vertical)

	ui.SearchTable.SetBorder(true).
		SetTitle("Stops and their data").
		SetTitleAlign(tview.AlignCenter)
	ui.SearchTable.SetSelectedFunc(func(row, _ int) {
		if row != 0 {
			// TODO(radomski): This is broken when searching, because it doesn't use the
			// relative rows of the stops that are currently on display
			ui.RefreshTimesInfo(ui.ConnectionsDisplayed[row - 1])
			ui.Pages.SwitchToPage("times")
		}
	})

	connections := ConnectionsFromStops(database.Stops)
	ui.PopulateSearchTable(connections)

	input := ui.CreateSearchInputFlex(database)

	return "search", tview.NewFlex().
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(ui.SearchTable, 0, 1, false).
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
	
	ui.TimesBanner = tview.NewTable()

	ui.TimesBanner.SetSeparator(tview.Borders.Vertical)
	ui.TimesBanner.SetBorder(true).SetTitle("Bus information").SetTitleAlign(tview.AlignLeft)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(ui.TimesBanner, 4, 0, false).
		AddItem(ui.Times, 0, 1, true)
	
	return "times", Center(80, 30, flex)
}

func (ui *UI) CreatePages(database *Database) {
	ui.Pages = tview.NewPages()
	
	name, primi := ui.CreateTimesPage()
	ui.Pages.AddPage(name, primi, true, false)
	
	name, primi = ui.CreateSearchPage(database)
	ui.Pages.AddPage(name, primi, true, true)
	ui.SetKeybindings(database)
}

func (ui *UI) SetKeybindings(database *Database) {
	app.SetInputCapture(func (event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlR:
			database.RefreshWithWeb()
			ui.UpdateUncompleteTable(database)
			return event
		case tcell.KeyCtrlN:
			if name, _ := ui.Pages.GetFrontPage(); name != "times" {
				return event
			}

			nextId := ui.TimesConnectionId + 1

			// Early out
			if nextId > len(database.Stops) {
				return event
			}

			connection := ConnectionFromStop(database.Stops[nextId])
			ui.RefreshTimesInfo(connection)
		case tcell.KeyCtrlP:
			if name, _ := ui.Pages.GetFrontPage(); name != "times" {
				return event
			}

			nextId := ui.TimesConnectionId - 1

			// Early out
			if nextId < 0 {
				return event
			}
			
			connection := ConnectionFromStop(database.Stops[nextId])
			ui.RefreshTimesInfo(connection)
		case tcell.KeyCtrlSpace:
			if name, _ := ui.Pages.GetFrontPage(); name != "search" {
				return event;
			}

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

func (ui *UI) RefreshTimesInfo(connection Connection) {
	ui.Times.Clear()
	ui.TimesConnectionId = connection.Stop.Id;
	
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

	for r, hour := range connection.Stop.Times.Hours {
		cell := tview.NewTableCell(hour).SetAlign(tview.AlignRight).SetExpansion(1)
		ui.Times.SetCell(r + 1, 0, cell)

		cell = tview.NewTableCell(minsOrEmpty(connection.Stop.Times.WorkMins, r)).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)
		ui.Times.SetCell(r + 1, 1, cell)
		
		cell = tview.NewTableCell(minsOrEmpty(connection.Stop.Times.SaturdayMins, r)).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)
		ui.Times.SetCell(r + 1, 2, cell)
		
		cell = tview.NewTableCell(minsOrEmpty(connection.Stop.Times.HolidayMins, r)).
			SetAlign(tview.AlignLeft).
			SetExpansion(1)
		ui.Times.SetCell(r + 1, 3, cell)
	}


	ui.TimesBanner.Clear()
	headers = "Line number;Direction;Stop name;Departure in"
	for c, header := range strings.Split(headers, ";") {
		cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
		ui.TimesBanner.SetCell(0, c, cell)
	}

	cell := tview.NewTableCell(strconv.Itoa(connection.Stop.LineNr)).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	ui.TimesBanner.SetCell(1, 0, cell)

	cell = tview.NewTableCell(connection.Stop.Direction).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	ui.TimesBanner.SetCell(1, 1, cell)
	
	cell = tview.NewTableCell(connection.Stop.Name).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	ui.TimesBanner.SetCell(1, 2, cell)
	
	cell = tview.NewTableCell(connection.InfoNext).
		SetAlign(tview.AlignCenter).
		SetExpansion(1)
	ui.TimesBanner.SetCell(1, 3, cell)
}

func (ui *UI) UpdateUncompleteTable(database *Database) {
	const updateInterval = 25 * time.Millisecond
	for (database.Status & DatabaseComplete) == 0 {
		app.QueueUpdateDraw(func() {
			ui.SearchTable.SetTitle("Data is now being loaded").SetTitleAlign(tview.AlignLeft)
			connections := ConnectionsFromStops(database.Stops)
			ui.PopulateSearchTable(connections)
		})
		time.Sleep(updateInterval)
	}

	loadedHeader := "All data is now loaded"
	for i := 0; i < 4; i++ {
		app.QueueUpdateDraw(func() {
			ui.SearchTable.SetTitle(loadedHeader).SetTitleAlign(tview.AlignLeft)
		})
		loadedHeader += "."
		time.Sleep(updateInterval * 10)
	}

	app.QueueUpdateDraw(func() {
		ui.SearchTable.SetTitle("Stops and their data").SetTitleAlign(tview.AlignCenter)
	})
}
