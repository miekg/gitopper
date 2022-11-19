package main

import (
	"strings"
)

type sliceFlag []string

func (s *sliceFlag) String() string     { return strings.Join(*s, "s") }
func (s *sliceFlag) Set(v string) error { *s = append(*s, v); return nil }
