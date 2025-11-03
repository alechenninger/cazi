package claims

// Standard claims for requester context.
// These follow common identity and authorization claim conventions.
var (
	// Sub is the subject identifier (typically a user ID)
	Sub = TopLevel[string]("sub")

	// PreferredUsername is the human-readable username
	PreferredUsername = TopLevel[string]("preferred_username")

	// Email is the email address
	Email = TopLevel[string]("email")

	// Roles is the list of roles assigned to the subject
	Roles = TopLevel[[]string]("roles")

	// Groups is the list of groups the subject belongs to
	Groups = TopLevel[[]string]("groups")
)
