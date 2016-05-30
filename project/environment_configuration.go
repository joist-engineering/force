package project

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// EnvironmentMatch can be specified as the `match` value in an environment stanza in
// environments.json.  The goal is to allow the author of a project to specify how to match
// a given target environment's configuration with the current active force user.
type EnvironmentMatch struct {
	// LoginRegex can be specified to match against the active login's username.
	LoginRegex *string `json:"login"`

	// InstanceRegex can be specified to match against the active login's sf instance hostname (eg.,
	// `https://na00.salesforce.com`)
	InstanceRegex *string `json:"instance"`
}

// ReplacementValueAsCommand can optionally be used instead of a string as a `var` in the environment
// config JSON to have heroku/force execute a command, grab its stdout, and use that as the
// replacement value.
type ReplacementValueAsCommand struct {
	// CommandToExecute is a command (and paramters) to be executed.  It is a list of the command
	// and the paramters to pass to it, and it will be executed with the PWD in the root of your
	// source directory.
	CommandToExecute []string `json:"exec"`
}

// EnvironmentConfigJSON is the struct within your environment.json that
// describes a single environment (staging, prod, sandbox, etc.)
type EnvironmentConfigJSON struct {
	// MatchCriteria This allows for mapping a login to an SF instance to the
	// environment it represents in a way that will be consistent across multiple developers'
	// accounts and machines.
	MatchCriteria *EnvironmentMatch `json:"match"`

	// Variables is a map of placeholders and values that will be interpolated into the metadata,
	// wherever the token is found with a $ prefixed.  The values are optionally either strings or
	// objects containing `exec` commands, hence the type is RawMessage here so it we can choose the
	// appropriate way to unmarshal it dynamically.  It is either supposed to be a
	// ReplacementValueAsCommand or a string.
	Variables map[string]json.RawMessage `json:"vars"`

	// Human-readable name for this instance.  This does not come from the contents of the JSON
	// object, but rather the name of the key in the top-level EnvironmentsConfigJSON object that
	// contained it.
	Name string
}

// EnvironmentsConfigJSON is the root struct for JSON unmarshalling that an `environment.json` file
// in your source tree root.  It can describe your SF environments and other settings, particularly
// parameters that can be templated into your Salesforce metadata files.
type EnvironmentsConfigJSON struct {
	Environments map[string]EnvironmentConfigJSON `json:"environments"`
}

// GetEnvironmentConfigForActiveUser retrieves the a user-specified environment configuration for
// the active project, looked up by comparing the given username and instance URI with matchers
// specified in environments.json for each environment.  Returns nil if there's no per-project
// environment config set up.
func (project *project) GetEnvironmentConfigForActiveEnvironment(activeUsername string, activeInstanceURI string) (foundEnvironment *EnvironmentConfigJSON, err error) {
	if environmentJSON, present := project.EnumerateContents()["environments.json"]; present {
		// now, we want to implement our interpolation regime!
		environmentConfig := EnvironmentsConfigJSON{}
		if err = json.Unmarshal(environmentJSON, &environmentConfig); err != nil {
			err = fmt.Errorf("Problem parsing environments.json at offset %v: %s", err.(*json.SyntaxError).Offset, err.Error())
			return
		}

		// now, to determine the current environment.
		for name, env := range environmentConfig.Environments {
			if env.MatchCriteria == nil {
				fmt.Printf("WARN: No matchers specified for environment '%s' in your environments.json.  See README.\n", name)
				continue
			}
			instanceMatched := true
			loginMatched := true

			if env.MatchCriteria.InstanceRegex != nil {
				instanceMatched, err = regexp.MatchString(*env.MatchCriteria.InstanceRegex, activeInstanceURI)
				if err != nil {
					return
				}
			}

			if env.MatchCriteria.LoginRegex != nil {
				loginMatched, err = regexp.MatchString(*env.MatchCriteria.LoginRegex, activeUsername)
				if err != nil {
					return
				}
			}

			if loginMatched && instanceMatched {
				envCopy := env

				foundEnvironment = &envCopy
				foundEnvironment.Name = name
				return
			}
		}
		if foundEnvironment == nil {
			err = fmt.Errorf("None of the environments specified in your project config matched your active login: '%s'\n", activeUsername)
		}
	}
	return
}
