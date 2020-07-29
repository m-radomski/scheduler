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

type SearchEntry struct {
	LineNr string
	Direction string
	StopName string
	InfoNext string
}

func SearchEntriesFromStops(stops []Stop) (result []SearchEntry) {
	for _, stop := range stops {
		entry := SearchEntry {
			LineNr: strconv.Itoa(stop.LineNr),
			Direction: stop.Direction,
			StopName: stop.Name,
			InfoNext: InfoNextBus(stop),
		}

		result = append(result, entry)
	}

	return 
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
						InfoNext: InfoNextBusOnConnection(stops[i:j + 1]),
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

func TimesToOneDay() {
	for i, stop := range globalDB.Stops {
		for j := 0; j < len(stop.Times.Hours) - 1; j++ {
			if IntOrPanic(stop.Times.Hours[j]) == 23 && IntOrPanic(stop.Times.Hours[j + 1]) == 0 {
				// NOTE(radomski): This is safe
				globalDB.Stops[i].Times.Hours = append(stop.Times.Hours[j + 1:], stop.Times.Hours[:j + 1]...)
				// NOTE(radomski): Those are not
				if len(stop.Times.WorkMins) != 0 {
					globalDB.Stops[i].Times.WorkMins = append(stop.Times.WorkMins[j + 1:], stop.Times.WorkMins[:j + 1]...)
				}
				if len(stop.Times.SaturdayMins) != 0 {
					globalDB.Stops[i].Times.SaturdayMins = append(stop.Times.SaturdayMins[j + 1:], stop.Times.SaturdayMins[:j + 1]...)
				}
				if len(stop.Times.HolidayMins) != 0 {
					globalDB.Stops[i].Times.HolidayMins = append(stop.Times.HolidayMins[j + 1:], stop.Times.HolidayMins[:j + 1]...)
				}
			}
		}
	}
}

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

func ClosestsBusTimeIndexes(currentHour, currentMin int, workingMins, workingHours []string) (hi, mi int) {
	hi = CurrentHourIndex(currentHour, workingHours)
	if hi == -1 {
		return BeyondSchedule, 0
	}

	if len(workingMins) == 0 {
		// Doesn't drive on today's type of day
		return NotWorkDays, 0
	}
	
	for ; hi < len(workingHours); hi++ {
		minsAtHour := strings.Split(workingMins[hi], " ")
		if len(minsAtHour) == 0 {
			currentMin = 0
			continue
		}

		for i, stopMinute := range minsAtHour {
			minTrimed := strings.TrimFunc(stopMinute, func(r rune) bool {
				return unicode.IsLetter(r)
			})

			if len(minTrimed) == 0 {
				continue
			}

			if IntOrPanic(minTrimed) >= currentMin {
				return hi, i
			}
		}
	}

	// We found the hour in this schedule but we don't fit with the minutes this time
	return BeyondSchedule, 0
}

func MinsToNextBus(stop Stop) (result int) {
	now := time.Now()
	nowHour, nowMin, _ := now.Clock()
	lookupMins := TodaysMins(now, stop.Times)
	hoffset, moffset := ClosestsBusTimeIndexes(nowHour, nowMin, lookupMins, stop.Times.Hours)

	// Error propagation
	if hoffset <= BeyondSchedule {
		return hoffset
	}
	
	reshour := IntOrPanic(stop.Times.Hours[hoffset])
	tmp := strings.Split(lookupMins[hoffset], " ")[moffset]
	resmins := IntOrPanic(strings.TrimFunc(tmp, func (r rune) bool {
				return unicode.IsLetter(r)
	}))
	
	return (reshour - nowHour) * 60 + resmins - nowMin
}

func CommuteLengthFromRoute(stops []Stop) (result int) {
	now := time.Now()
	nowHour, nowMin, _ := now.Clock()
	lookupMins := TodaysMins(now, stops[0].Times) // NOTE(radomski): I shouldn't be doing this is a loop
	hi, mi := ClosestsBusTimeIndexes(nowHour, nowMin, lookupMins, stops[0].Times.Hours)
		
	nowHour = IntOrPanic(stops[0].Times.Hours[hi])
	tmp := strings.Split(lookupMins[hi], " ")[mi]
	nowMin = IntOrPanic(strings.TrimFunc(tmp, func (r rune) bool {
		return unicode.IsLetter(r)
	}))
	
	for _, stop := range stops[1:] {
		if hi <= BeyondSchedule {
			// panic("Unreachable")
			return result
		}
		
		resultHour := IntOrPanic(stop.Times.Hours[hi])
		tmp := strings.Split(lookupMins[hi], " ")[mi]
		resultMin := IntOrPanic(strings.TrimFunc(tmp, func (r rune) bool {
			return unicode.IsLetter(r)
		}))
		
		result += (resultHour - nowHour) * 60 + resultMin - nowMin
		
		nowHour = resultHour
		nowMin = resultMin
	}
	
	return 0
}

func InfoNextBus(stop Stop) (result string) {
	switch minNext := MinsToNextBus(stop); minNext {
	case BeyondSchedule:
		return "Beyond schedule"
	case NotWorkDays:
		return "Doesn't drive today"
	default:
		if minNext != 0 {
			return fmt.Sprintf("In %d min", minNext)			
		} else {
			return fmt.Sprintf("Departing right now!")
		}
	}
}

// NOTE(radomski): Idealy this would return two strings, one being 
// minutes until the next bus and second one being the commute length
func InfoNextBusOnConnection(stops []Stop) (result string) {
	switch minNext := MinsToNextBus(stops[0]); minNext {
	case BeyondSchedule:
		return "Beyond schedule"
	case NotWorkDays:
		return "Doesn't drive today"
	default:
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
