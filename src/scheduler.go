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
	LineNr, Direction, InfoNext string

	// NOTE(radomski): See comment in `FindConnections`
	// CommuteLength, MinutesUntilNext string
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
		// Both of this calls are case insensitive, I don't really know if
		// someone would ever want it to be case sensitive.
		const treshold float64 = 0.9
		return IsFuzzyEqualInsens(main, substr, treshold) ||
			strings.HasPrefix(strings.ToLower(main), strings.ToLower(substr))
	}
	
	for i := 0; i < len(stops); i++ {
		if !filter(stops[i].Name, from) {
			continue
		}

		line := stops[i].LineNr
		dir := stops[i].Direction

		for j := i; j < len(stops); j++ {
			if line == stops[j].LineNr && dir == stops[j].Direction && filter(stops[j].Name, to) {
					connection := Connection {
						LineNr: strconv.Itoa(stops[j].LineNr),
						Direction: stops[i].Name + " -> " + stops[j].Name,
						// TODO(radomski): I don't know if we really need this
						// But it would be nice to not be so reliant on InfoNextBus
						// returning a formated string. Idealy we would want something
						// that just returns mins until next bus and commute lenght
						// as ints and we would transform them somewhere down the road
						// CommuteLength: minsLength,
						InfoNext: InfoNextBus(stops[i:j + 1]),
					}
				ret = append(ret, connection)
			} else if line != stops[j].LineNr || dir != stops[j].Direction {
				i += j - 1 - i // skip this many stops, because the are on the same route
				break
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

	// TODO(radomski): Each hour after 23 should be increased by 24
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

func CommuteLengthFromRoute(stops []Stop) (result int) {
	return 0
}

func InfoNextBus(stops []Stop) (result string) {
	switch minNext := MinsToNextBus(stops[0]); minNext {
	case BeyondSchedule:
		return "Beyond schedule"
	case NotWorkDays:
		return "Doesn't drive today"
	default:

		// TODO(radomski): Once again this is an awful hack, FIX THISSS please
		// just create a sensible data structure for this
		if len(stops) == 1 {
			if minNext != 0 {
				return fmt.Sprintf("In %d min", minNext)			
			} else {
				return fmt.Sprintf("Departing right now!")
			}
		}

		commuteLength := CommuteLengthFromRoute(stops)
		if minNext != 0 {
			return fmt.Sprintf("In %d min [%d min ride]", minNext, commuteLength)			
		} else {
			return fmt.Sprintf("Departing right now! [%d min ride]", commuteLength)
		}
	}
}

func Run() {
	ReadJson()

	viewable.CreatePages()
	if err := app.SetRoot(viewable.Pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
