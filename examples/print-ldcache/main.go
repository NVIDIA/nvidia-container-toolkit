package main

import (
	"github.com/NVIDIA/nvidia-container-toolkit/internal/ldcache"
	log "github.com/sirupsen/logrus"
)

var logger = log.StandardLogger()

func main() {
	logger.SetLevel(log.DebugLevel)
	logger.Infof("Starting device discovery with NVML")

	cache, err := ldcache.NewLDCacheWithLogger(logger, "/run/nvidia/driver")
	if err != nil {
		logger.Errorf("Error loading ldcache: %v", err)
		return
	}
	defer cache.Close()

	libs32, libs64 := cache.Lookup("lib")

	logger.Infof("32-bit: %v", libs32)
	logger.Infof("64-bit: %v", libs64)

}
