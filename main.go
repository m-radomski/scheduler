package main

import (
	"fmt"
	"log"
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

func readJson() []StopTimes {
	b, err := ioutil.ReadFile("/home/mateusz/normal.json")
	if err != nil {
		log.Fatal(err)
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

func CreateSearchPage(showTimes func(chosen StopTimes)) (title string, content tview.Primitive) {
	table := tview.NewTable()
	input := tview.NewInputField()

	// cycleFocus := func() {
	// 	focused := app.GetFocus()
	// 	if focused == input {
	// 		app.SetFocus(table)
	// 	} else {
	// 		app.SetFocus(input)
	// 	}
	// }

	tableFromArray := func(stops []StopTimes) {
		table.Clear().
		SetFixed(1, 1).
		SetSelectable(true, false).
		SetSeparator(tview.Borders.Vertical)
		rows := len(stops)

		cell := tview.NewTableCell("Line number").SetAlign(tview.AlignCenter)
		table.SetCell(0, 0, cell)
		cell = tview.NewTableCell("Direction").SetAlign(tview.AlignCenter)
		table.SetCell(0, 1, cell)
		cell = tview.NewTableCell("Stop name").SetAlign(tview.AlignCenter)
		table.SetCell(0, 2, cell)

		for r := 0; r < rows; r++ {
			c := 0
			cell := tview.NewTableCell(strconv.Itoa(stops[r].LineNr)).
			SetAlign(tview.AlignCenter)
			table.SetCell(r+1, c, cell)

			c += 1
			cell = tview.NewTableCell(stops[r].Direction).
			SetAlign(tview.AlignCenter)
			table.SetCell(r+1, c, cell)

			c += 1
			cell = tview.NewTableCell(stops[r].Name).
			SetAlign(tview.AlignCenter)
			table.SetCell(r+1, c, cell)
		}

		table.SetBorder(true).SetTitle("Stops and their data").SetTitleAlign(tview.AlignCenter)
		table.SetSelectedFunc(func(row, _ int) {
			if row != 0 {
				showTimes(stops[row - 1])
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
			AddItem(table, 0, 1, true).
			AddItem(input, 2, 0, false),
		0, 1, false)
}

func main() {
	// app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
	// 	if event.Key() == tcell.KeyCtrlN {
	// 		cycleFocus()
	// 	}
	// 	return event
	// })

	dummy := func(chosen StopTimes) {
		app.Stop()
		fmt.Println("Ended on", chosen)
	}

	pages := tview.NewPages()
	name, primi := CreateSearchPage(dummy)
	pages.AddAndSwitchToPage(name, primi, true)

	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
