package internal

// priority 1 is higher than priority two
var topLevelCategoryToPriority = map[string]int{
	"MINIMAL":   1,
	"BASIC":     2,
	"DEFAULT":   3,
	"COMMENTS":  4,
	"UNARY_RPC": 5,
	"FILE":      1,
	"PACKAGE":   2,
	"WIRE_JSON": 3,
	"WIRE":      4,
}

func categoryCompare(one string, two string) int {
	onePriority, oneIsTopLevel := topLevelCategoryToPriority[one]
	twoPriority, twoIsTopLevel := topLevelCategoryToPriority[two]
	if oneIsTopLevel && !twoIsTopLevel {
		return -1
	}
	if !oneIsTopLevel && twoIsTopLevel {
		return 1
	}
	if oneIsTopLevel && twoIsTopLevel {
		if onePriority < twoPriority {
			return -1
		}
		if onePriority > twoPriority {
			return 1
		}
	}
	if one < two {
		return -1
	}
	if one > two {
		return 1
	}
	return 0
}
