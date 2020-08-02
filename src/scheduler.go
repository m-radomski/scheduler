package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"sort"

	"github.com/rivo/tview"
)

var (
	app *tview.Application = tview.NewApplication()
)

type Connection struct {
	Stop *Stop
	Path, InfoNext string

	// NOTE(radomski): See comment in `FindConnections`
	// CommuteLength, MinutesUntilNext string
}

type SearchEntry struct {
	LineNr string
	Direction string
	StopName string
	InfoNext string
}

func ConnectionFromStop(stop Stop) (result Connection) {
	return Connection {
		Stop: &stop,
		InfoNext: InfoNextBus(stop),
	}

	return 
}

func ConnectionsFromStops(stops []Stop) (result []Connection) {
	for _, stop := range stops {
		result = append(result, ConnectionFromStop(stop))
	}

	return
}

func InputFilter(main, s string) bool {
	// Both of this calls are case insensitive, I don't really know if
	// someone would ever want it to be case sensitive.
	const treshold float64 = 0.9
	return IsFuzzyEqualInsens(main, s, treshold) ||
		strings.HasPrefix(strings.ToLower(main), strings.ToLower(s))
}

func InputMapFindOrInsert(main, s string,  m *map[string]bool) bool {
	passed, present := (*m)[main]
	if !present {
		matches := InputFilter(main, s)
		(*m)[main] = matches

		return matches
	}
	
	return passed
}

func FindInStops(stops []Stop, s string) (ret []Stop) {
	namePassed := make(map[string]bool)
	
	for _, stop := range stops {
		nondigit := strings.IndexFunc(s, func(r rune) bool {
			return !unicode.IsDigit(r)
		})
		
		if nondigit == -1 && strings.HasPrefix(strconv.Itoa(stop.LineNr), s) {
			ret = append(ret, stop)
		} else if InputMapFindOrInsert(stop.Name, s, &namePassed) {
				ret = append(ret, stop)
		}
	}
	return
}

func FindConnections(from, to string, stops []Stop) (ret []Connection) {
	fromPassed := make(map[string]bool)
	toPassed := make(map[string]bool)
	
	for i := 0; i < len(stops); i++ {
		if !InputMapFindOrInsert(stops[i].Name, from, &fromPassed) {
			continue
		}

		line := stops[i].LineNr
		dir := stops[i].Direction

		for j := i; j < len(stops); j++ {
			if line != stops[j].LineNr || dir != stops[j].Direction {
				i += j - 1 - i // skip this many stops, because the are on the same route
				break
			} else if InputMapFindOrInsert(stops[j].Name, to, &toPassed) {
				connection := Connection {
						Stop: &stops[i],
						Path: stops[i].Name + " -> " + stops[j].Name,
						InfoNext: InfoNextBusOnConnection(stops[i:j + 1]),
					}
				ret = append(ret, connection)
			}
		}
	}

	return
}

func FindConnectionsOnlyFrom(from string, stops []Stop) (ret []Connection) {
	fromPassed := make(map[string]bool)

	stopsLength := len(stops)
	for i := 0; i < stopsLength; i++ {
		if !InputMapFindOrInsert(stops[i].Name, from, &fromPassed) {
			continue
		}

		line := stops[i].LineNr
		dir := stops[i].Direction

		j := i
		for (j + 1) < stopsLength &&
			line == stops[j + 1].LineNr &&
			dir == stops[j + 1].Direction {
			j++
		}

		connection := Connection {
			Stop: &stops[i],
			Path: stops[i].Name + " -> " + stops[j].Name,
			InfoNext: InfoNextBusOnConnection(stops[i:j + 1]),
		}

		ret = append(ret, connection)
	}
	
	return 
}

func FindConnectionsOnlyTo(to string, stops []Stop) (ret []Connection) {
	toPassed := make(map[string]bool)

	stopsLength := len(stops)
	for i := 0; i < stopsLength; i++ {
		line := stops[i].LineNr
		dir := stops[i].Direction

		for j := i; j < stopsLength; j++ {
			if line != stops[j].LineNr || dir != stops[j].Direction {
				i += j - 1 - i // skip this many stops, because the are on the same route
				break
			} else if InputMapFindOrInsert(stops[j].Name, to, &toPassed) {
				connection := Connection {
					Stop: &stops[i],
					Path: stops[i].Name + " -> " + stops[j].Name,
					InfoNext: InfoNextBusOnConnection(stops[i:j + 1]),
				}
				ret = append(ret, connection)
			}
		}
	}

	return
}

func SortConnectionsOnTime(connections []Connection) (result []Connection) {
	valueOnInfo := func(connection Connection) int {
		const INT32_MAX int = (1 << 31) - 1
		if strings.HasPrefix(connection.InfoNext, "Depart") {
			return 0
		} else if strings.HasPrefix(connection.InfoNext, "In") {
			mins := strings.TrimFunc(connection.InfoNext, func (r rune) bool {
				return unicode.IsLetter(r) || unicode.IsSpace(r)
			})
			
			nondigit := strings.IndexFunc(mins, func(r rune) bool {
				return !unicode.IsDigit(r)
			})

			if nondigit != -1 {
				mins = mins[:nondigit]
			}

			minsInt, err := strconv.Atoi(mins)
			if err != nil {
				panic(err)
			}

			return minsInt
		} else if strings.HasPrefix(connection.InfoNext, "Beyond") {
			return INT32_MAX - 1
		} else if strings.HasPrefix(connection.InfoNext, "Doesn") {
			return INT32_MAX
		} else {
			panic("Unreachable")
		}
	}
	
	sort.Slice(connections, func(i, j int) bool {
		return valueOnInfo(connections[i]) < valueOnInfo(connections[j])
	})

	return connections
}

const (
	BeyondSchedule = -1
	NotWorkDays = -2
)

func TimesToOneDay(stops []Stop) (result []Stop) {
	for _, stop := range stops {
		toAppend := stop
		for j := 0; j < len(stop.Times.Hours) - 1; j++ {
			if IntOrPanic(stop.Times.Hours[j]) == 23 && IntOrPanic(stop.Times.Hours[j + 1]) == 0 {
				new := Times {
					// NOTE(radomski): This is safe
					Hours: append(stop.Times.Hours[j + 1:], stop.Times.Hours[:j + 1]...),
				}
				// NOTE(radomski): Those are not
				if len(stop.Times.WorkMins) != 0 {
					new.WorkMins = append(stop.Times.WorkMins[j + 1:], stop.Times.WorkMins[:j + 1]...)
				}
				if len(stop.Times.SaturdayMins) != 0 {
					new.SaturdayMins = append(stop.Times.SaturdayMins[j + 1:], stop.Times.SaturdayMins[:j + 1]...)
				}
				if len(stop.Times.HolidayMins) != 0 {
					new.HolidayMins = append(stop.Times.HolidayMins[j + 1:], stop.Times.HolidayMins[:j + 1]...)
				}

				toAppend = Stop {
					Id: stop.Id,
					LineNr: stop.LineNr,
					Direction: stop.Direction,
					Name: stop.Name,
					Times: new,
				}
				break;
			}
		}
		result = append(result, toAppend)
	}

	return 
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

		currentMin = 0
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
	lookupMins := TodaysMins(now, stops[0].Times)
	hi, mi := ClosestsBusTimeIndexes(nowHour, nowMin, lookupMins, stops[0].Times.Hours)
		
	nowHour = IntOrPanic(stops[0].Times.Hours[hi])
	tmp := strings.Split(lookupMins[hi], " ")[mi]
	nowMin = IntOrPanic(strings.TrimFunc(tmp, func (r rune) bool {
		return unicode.IsLetter(r)
	}))

	for _, stop := range stops[1:] {
		lookupMins := TodaysMins(now, stop.Times)
		hi, mi = ClosestsBusTimeIndexes(nowHour, nowMin, lookupMins, stop.Times.Hours)
		if hi <= BeyondSchedule {
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
	
	return result
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
	database := NewDatabase()
	database.CreateFromJSON()

	ui := NewUI()
	go ui.UpdateUncompleteTable(&database)
	ui.CreatePages(&database)
	if err := app.SetRoot(ui.Pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
