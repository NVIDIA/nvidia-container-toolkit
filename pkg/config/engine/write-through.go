package engine

type DropInConfig struct {
	Source      RuntimeConfigSource
	Destination RuntimeConfigDestination
}

func (c *DropInConfig) RemoveRuntime(runtime string) error {
	// TODO: For the time being we also allow the SOURCE runtime to be removed.
	// This should only be done if a config file is used.
	// _ = c.Source.RemoveRuntime(runtime)
	return c.Destination.RemoveRuntime(runtime)
}

func (c *DropInConfig) AddRuntime(name string, path string, setAsDefault bool) error {
	options := c.Source.GetDefaultRuntimeOptions()
	return c.Destination.AddRuntimeWithOptions(name, path, setAsDefault, options)
}

func (c *DropInConfig) EnableCDI() {
	c.Destination.EnableCDI()
}

func (c *DropInConfig) DefaultRuntime() string {
	return c.Source.DefaultRuntime()
}

func (c *DropInConfig) GetRuntimeConfig(runtime string) (RuntimeConfig, error) {
	return c.Source.GetRuntimeConfig(runtime)
}

func (c *DropInConfig) Save(path string) (int64, error) {
	// TODO: If the Source has changed -- for example by removing a runtime --
	// then we need to also save the source. Note that the path may change in
	// this case.
	return c.Destination.Save(path)
}

func (c *DropInConfig) String() string {
	return c.Destination.String()
}

type RuntimeConfigSource interface {
	DefaultRuntime() string
	GetRuntimeConfig(string) (RuntimeConfig, error)
	GetDefaultRuntimeOptions() interface{}
	String() string
}

type RuntimeConfigDestination interface {
	AddRuntimeWithOptions(string, string, bool, interface{}) error
	EnableCDI()
	RemoveRuntime(string) error
	Save(string) (int64, error)
	String() string
}
