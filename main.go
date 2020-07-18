package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"strconv"
	"strings"
	"encoding/json"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
)

type StopTimes struct {
	LineNr int `json:"line"`
	Direction string `json:"direction"`
	Name string `json:"stop_name"`
	Times []string `json:"times"`
}

var app *tview.Application = tview.NewApplication()
var stops []StopTimes = readJson()

var dbPath string = "/home/mateusz/.local/scheduler/schedule.json"

func readJson() []StopTimes {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("Missing database file, fetching it from set FTP server")
		FTPFetch(ReadFTPCred("my.cred"))
	}

	b, err := ioutil.ReadFile(dbPath)
	if err != nil {
		panic(err)
	}

	var stops []StopTimes
	json.Unmarshal(b, &stops)
	return stops
}

func findInStops(stops []StopTimes, s string) (ret []StopTimes) {
	for _, stop := range stops {
		if strings.Contains(stop.Name, s) || strings.Contains(strconv.Itoa(stop.LineNr), s) {
			ret = append(ret, stop)
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

func CreateSearchPage(showTimes func(times []string)) (title string, content tview.Primitive) {
	table := tview.NewTable()
	input := tview.NewInputField()

	tableFromArray := func(stops []StopTimes) {
		table.Clear().
		SetFixed(1, 1).
		SetSelectable(true, false).
		SetSeparator(tview.Borders.Vertical)
		rows := len(stops)

		cell := tview.NewTableCell("Line number").SetAlign(tview.AlignCenter).SetExpansion(1)
		table.SetCell(0, 0, cell)
		cell = tview.NewTableCell("Direction").SetAlign(tview.AlignCenter).SetExpansion(1)
		table.SetCell(0, 1, cell)
		cell = tview.NewTableCell("Stop name").SetAlign(tview.AlignCenter).SetExpansion(1)
		table.SetCell(0, 2, cell)

		for r := 0; r < rows; r++ {
			c := 0
			cell := tview.NewTableCell(strconv.Itoa(stops[r].LineNr)).
			SetAlign(tview.AlignCenter).SetExpansion(1)
			table.SetCell(r+1, c, cell)

			c += 1
			cell = tview.NewTableCell(stops[r].Direction).
			SetAlign(tview.AlignCenter).SetExpansion(1)
			table.SetCell(r+1, c, cell)

			c += 1
			cell = tview.NewTableCell(stops[r].Name).
			SetAlign(tview.AlignCenter).SetExpansion(1)
			table.SetCell(r+1, c, cell)
		}

		table.SetBorder(true).SetTitle("Stops and their data").SetTitleAlign(tview.AlignCenter)
		table.SetSelectedFunc(func(row, _ int) {
			if row != 0 {
				showTimes(stops[row - 1].Times)
			}
		})
	}

	showResults := func() {
		nstops := findInStops(stops, input.GetText())
		tableFromArray(nstops)
		app.SetFocus(table)
	}

	tableFromArray(stops)

	input.SetLabel("Search for: ").
		SetDoneFunc(func(key tcell.Key) {
			showResults()
		})

	return "search", tview.NewFlex().
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(table, 0, 1, false).
			AddItem(input, 2, 0, true),
		0, 1, true)
}

func CreateTimesPage(searchAgain func()) (title string, content tview.Primitive, refresh func(times []string)) {
	table := tview.NewTable()

	refresh = func(times []string) {
		table.Clear()
		table.SetSelectable(true, true).SetSeparator(tview.Borders.Vertical)

		headers := "Hour;Work Day;Saturday;Holiday"
		for c, header := range strings.Split(headers, ";") {
			cell := tview.NewTableCell(header).SetAlign(tview.AlignCenter).SetExpansion(1)
			table.SetCell(0, c, cell)
		}

		for r, time := range times {
			for c, hms := range strings.Split(time, "; ") {
				align := tview.AlignLeft
				if c == 0 {
					align = tview.AlignRight
				}
				cell := tview.NewTableCell(hms).SetAlign(align).SetExpansion(1)
				table.SetCell(r + 1, c, cell)
			}
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

	refresh := func(times []string) {}
	dummy := func(times []string) {
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
