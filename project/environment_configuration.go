package project

// EnvironmentConfigJson is the struct within your environment.json that
// describes a single
type EnvironmentConfigJson struct {
	InstanceHost string            `json:"instance"`
	Variables    map[string]string `json:"vars"`

	Name string
}
