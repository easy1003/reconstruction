package main

import (
	"fmt"
	"testing"
)


func TestFormat(t *testing.T)  {
	format := func(value float64) string {
		return fmt.Sprintf("$%v",fmt.Sprintf("%.2f", value))
	}
	s := format(float64(1730))
	fmt.Println(s)
}
