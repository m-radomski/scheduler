package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/rivo/tview"
)

var (
	app *tview.Application = tview.NewApplication()
	viewable Viewable
	globalDB Database = NewDatabase()
)

type Connection struct {
	LineNr, Direction, StopName, MinutesUntilNext string
}

func FindInStops(stops []Stop, s string) (ret []Stop) {
	for _, stop := range stops {
		if strings.Contains(stop.Name, s) || strings.Contains(strconv.Itoa(stop.LineNr), s) {
			ret = append(ret, stop)
		}
	}
	return
}

func FindConnections(from, to string, stops []Stop) (ret []Connection) {
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
						connection := Connection {
							LineNr: strconv.Itoa(stops[j].LineNr),
							Direction: stops[i].Name + " -> " + stops[j].Name,
							StopName: stops[i].Name,
							MinutesUntilNext: InfoNextBus(stops[i]),
						}
					ret = append(ret, connection)
				} else if line != stops[j].LineNr || dir != stops[j].Direction {
					i += j - 1 - i // skip this many stops, because the are on the same route
					break
				}
			}
		}
	}

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

func Run() {
	ReadJson()

	viewable.CreatePages()
	if err := app.SetRoot(viewable.Pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
