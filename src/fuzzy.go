package scheduler

import "strings"

func Max(a, b int) int {
	if a > b {
		return a
	} else {
		return b
	}
}

// Based on
// https://en.wikipedia.org/wiki/Jaro%E2%80%93Winkler_distance
func JWDist(str1, str2 string) (weight float64) {
	// We can't compare empty strings
	if len(str1) == 0 || len(str2) == 0 {
		return 0.0
	}

	// If they are the same, we don't need to compare them
	if str1 == str2 {
		return 1.0
	}

	m := 0
	checkRange := (Max(len(str1), len(str2)) / 2) - 1
	str1Matches := make([]bool, len(str1))
	str2Matches := make([]bool, len(str2))

	// Iterate over chars from the first string
	for i, _ := range str1 {
		// Set the boundries on the chars that we are going to iterate over
		// from the second string
		low := 0
		if i > checkRange {
			low = i - checkRange
		}

		high := i + checkRange
		if high > len(str2) {
			high = len(str2)
		}

		for j := low; j < high; j++ {
			if str1Matches[i] == false && str2Matches[j] == false && str1[i] == str2[j] {
				m += 1
				str1Matches[i] = true
				str2Matches[j] = true
				// TODO(radomski): try to remove it and see what happens
				break
			}
		}
	}

	// If not a single character matched
	if m == 0 {
		return 0.0
	}

	// Count the transposed chars
	tmp := 0
	transpos := 0
	for i := 0; i < len(str1); i++ {
		if str1Matches[i] == false {
			continue
		}

		j := 0
		for j = tmp; j < len(str2); j++ {
			if str2Matches[j] == false {
				continue
			}

			tmp = j + 1
			break;
		}

		if str1[i] != str2[j] {
			transpos += 1
		}
	}

	// Calculating the weight based on the set equations
	mf := float64(m)
	len1f := float64(len(str1))
	len2f := float64(len(str2))
	weight = (mf / len1f + mf / len2f + (mf - (float64(transpos) / 2.0)) / mf) / 3.0
	const scaling float64 = 0.1
	shortest := Min(len(str1), len(str2))
	l := 0

	// Counting up to 4 chars, the ones that are the same at the start of the string
	if weight > 0.7 {
		for l < shortest && str1[l] == str2[l] && l < 4 {
			l += 1
		}
	}

	weight += float64(l) * scaling * (1 - weight)
	
	return
}

func FuzzyScaleInsens(str1, str2 string) float64 {
	lower1 := strings.ToLower(str1)
	lower2 := strings.ToLower(str2)

	dist := JWDist(lower1, lower2)
	return dist
}
