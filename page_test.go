package pdf

import (
	"fmt"
	"sort"
	"testing"
)

func TestTextHorizontal(t *testing.T) {
	X := 1.1
	Y := 2.1
	hor := TextHorizontal{}
	for i := 0; i < 10; i++ {
		hor = append(hor, Text{X:X, Y:Y, S:fmt.Sprint(i)})
	}
	fmt.Printf("%+v\n", hor)
	sort.Sort(hor)
	fmt.Printf("%+v\n", hor)
}
