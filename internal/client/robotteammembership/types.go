package robotteammembership

// Robot scope types.
const (
	RobotMembershipTypeTeam string = "teams"
)

// A RelationshipList represents JSON API relationships.
// https://jsonapi.org/format/#document-resource-object-relationships
type RelationshipList struct {
	Data []ResourceIdentifier `json:"data"`
}

type ResourceIdentifier struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type DeleteParameters struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}
