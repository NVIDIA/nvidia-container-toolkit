package server

import "github.com/NVIDIA/go-nvml/pkg/nvml/mock/gpus"

func WithGPUs(gpus ...gpus.Config) Option {
	return func(o *options) error {
		o.gpus = gpus
		return nil
	}
}

func WithDriverVersion(version string) Option {
	return func(o *options) error {
		o.DriverVersion = version
		return nil
	}
}

func WithNVMLVersion(version string) Option {
	return func(o *options) error {
		o.NvmlVersion = version
		return nil
	}
}

// TODO: Add a string implementation using generics
func WithCUDADriverVersion(version int) Option {
	return func(o *options) error {
		o.CudaDriverVersion = version
		return nil
	}
}
