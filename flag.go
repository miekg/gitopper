package main

import (
	"strings"
)

type sliceFlag struct {
	Data *[]string
}

func (s *sliceFlag) String() string {
	if s == nil || s.Data == nil {
		return ""
	}
	return strings.Join(*s.Data, ",")
}
func (s *sliceFlag) Set(v string) error {
	*s.Data = strings.Split(v, ",")
	return nil
}
