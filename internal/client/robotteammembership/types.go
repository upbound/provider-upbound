package robotteammembership

// Robot scope types.
const (
	RobotScopeTypeTeam string = "team"
)

type CreateParameters struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type DeleteParameters struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
