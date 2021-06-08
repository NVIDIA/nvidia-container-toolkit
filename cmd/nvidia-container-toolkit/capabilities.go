package main

import (
	"log"
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
