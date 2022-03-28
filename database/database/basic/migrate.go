package makeless_go_database_basic

import (
	"github.com/makeless/makeless-go/database/model"
)

func (database *Database) Migrate() error {
	return database.GetConnection().AutoMigrate(
		new(makeless_go_model.User),
		new(makeless_go_model.EmailVerification),
		new(makeless_go_model.PasswordRequest),
		new(makeless_go_model.Team),
		new(makeless_go_model.TeamUser),
		new(makeless_go_model.TeamInvitation),
	)
}
