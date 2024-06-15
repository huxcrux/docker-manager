package config

type Config struct {
	AppConfig  AppConfig         `yaml:"app_config"`
	Containers []ContainerConfig `yaml:"containers"`
}

type AppConfig struct {
	Debug                    bool `yaml:"debug"`
	UpdateCheck              bool `yaml:"update_check"`
	RemoveUnwantedContainers bool `yaml:"remove_unwanted_containers"`
}

type ContainerConfig struct {
	Image        string        `yaml:"image"`
	Name         string        `yaml:"name"`
	PortBindings []PortBinding `yaml:"port_bindings"`
	Env          []string      `yaml:"env"`
	Cmd          []string      `yaml:"cmd"`
}

type PortBinding struct {
	Port     string `yaml:"port"`
	Protocol string `yaml:"protocol"`
	HostIP   string `yaml:"host_ip"`
	HostPort string `yaml:"host_port"`
}
