package models

type CardColor string

type UnoCard struct {
	Value CardValue
	Color CardColor
}

const (
	Red    CardColor = "RED"
	Blue   CardColor = "BLUE"
	Green  CardColor = "GREEN"
	Yellow CardColor = "YELLOW"
	Wild   CardColor = "WILD"
)

type CardValue string

const (
	u0 CardValue = "0"
	u1 CardValue = "1"
	u2 CardValue = "2"
	u3 CardValue = "3"
	u4 CardValue = "4"
	u5 CardValue = "5"
	u6 CardValue = "6"
	u7 CardValue = "7"
	u8 CardValue = "8"
	u9 CardValue = "9"

	Skip     CardValue = "SKIP"
	Rev      CardValue = "REVERSE"
	Pl2      CardValue = "DRAW_TWO"
	Pl4      CardValue = "WILD_DRAW_FOUR"
	WildCard CardValue = "WILD" // plain Wild: recolors play without forcing a draw
)

var NumberToCardvalueUnoMap = map[int]CardValue{
	1: u1,
	2: u2,
	3: u3,
	4: u4,
	5: u5,
	6: u6,
	7: u7,
	8: u8,
	9: u9,
}

// var CardvalueToNumberUnoMap = map[CardValue]int{
// 	u1: 1,
// 	u2: 2,
// 	u3: 3,
// 	u4: 4,
// 	u5: 5,
// 	u6: 6,
// 	u7: 7,
// 	u8: 8,
// 	u9: 9,
// }
