package client

// CLIConfig is format for the up configuration file.
type CLIConfig struct {
	Upbound Upbound `json:"upbound"`
}

// Upbound contains configuration information for Upbound.
type Upbound struct {
	// Default indicates the default profile.
	Default string `json:"default"`

	// Profiles contain sets of credentials for communicating with Upbound. Key
	// is name of the profile.
	Profiles map[string]Profile `json:"profiles,omitempty"`
}

// A Profile is a set of credentials
type Profile struct {
	// ID is either a username, email, or token.
	ID string `json:"id"`

	// Type is the type of the profile.
	Type ProfileType `json:"type"`

	// Session is a session token used to authenticate to Upbound.
	Session string `json:"session,omitempty"`

	// Account is the default account to use when this profile is selected.
	Account string `json:"account,omitempty"`

	// BaseConfig represent persisted settings for this profile.
	// For example:
	// * flags
	// * environment variables
	BaseConfig map[string]string `json:"base,omitempty"`
}

// ProfileType is a type of Upbound profile.
type ProfileType string

// Types of profiles.
const (
	UserProfileType  ProfileType = "user"
	TokenProfileType ProfileType = "token"
)
