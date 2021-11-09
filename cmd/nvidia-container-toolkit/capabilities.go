package main

import (
	"log"
	"strings"
)

const (
	allDriverCapabilities     = DriverCapabilities("compute,compat32,graphics,utility,video,display,ngx")
	defaultDriverCapabilities = DriverCapabilities("utility,compute")

	none = DriverCapabilities("")
	all  = DriverCapabilities("all")
)

func capabilityToCLI(cap string) string {
	switch cap {
	case "compute":
		return "--compute"
	case "compat32":
		return "--compat32"
	case "graphics":
		return "--graphics"
	case "utility":
		return "--utility"
	case "video":
		return "--video"
	case "display":
		return "--display"
	case "ngx":
		return "--ngx"
	default:
		log.Panicln("unknown driver capability:", cap)
	}
	return ""
}

// DriverCapabilities is used to process the NVIDIA_DRIVER_CAPABILITIES environment
// variable. Operations include default values, filtering, and handling meta values such as "all"
type DriverCapabilities string

// Intersection returns intersection between two sets of capabilities.
func (d DriverCapabilities) Intersection(capabilities DriverCapabilities) DriverCapabilities {
	if capabilities == all {
		return d
	}
	if d == all {
		return capabilities
	}

	lookup := make(map[string]bool)
	for _, c := range d.list() {
		lookup[c] = true
	}
	var found []string
	for _, c := range capabilities.list() {
		if lookup[c] {
			found = append(found, c)
		}
	}

	intersection := DriverCapabilities(strings.Join(found, ","))
	return intersection
}

// String returns the string representation of the driver capabilities
func (d DriverCapabilities) String() string {
	return string(d)
}

// list returns the driver capabilities as a list
func (d DriverCapabilities) list() []string {
	var caps []string
	for _, c := range strings.Split(string(d), ",") {
		trimmed := strings.TrimSpace(c)
		if len(trimmed) == 0 {
			continue
		}
		caps = append(caps, trimmed)
	}

	return caps
}
