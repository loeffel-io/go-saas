package makeless_go

import (
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/makeless/makeless-go/http"
	"github.com/makeless/makeless-go/mailer"
	"github.com/makeless/makeless-go/model"
	"github.com/makeless/makeless-go/security"
	"github.com/makeless/makeless-go/struct"
	"gorm.io/gorm"
	h "net/http"
	"sync"
)

func (makeless *Makeless) teamInvitation(http makeless_go_http.Http) error {
	http.GetRouter().GetEngine().GET(
		"/api/team-invitation",
		func(c *gin.Context) {
			var err error
			var token = c.Query("token")
			var teamInvitation = &makeless_go_model.TeamInvitation{
				RWMutex: new(sync.RWMutex),
			}

			if teamInvitation, err = http.GetDatabase().GetTeamInvitationByField(http.GetDatabase().GetConnection().WithContext(c), teamInvitation, "token", token); err != nil {
				switch errors.Is(err, gorm.ErrRecordNotFound) {
				case true:
					c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				default:
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				}
				return
			}

			c.JSON(h.StatusOK, http.Response(nil, teamInvitation))
		},
	)

	return nil
}

func (makeless *Makeless) teamInvitations(http makeless_go_http.Http) error {
	http.GetRouter().GetEngine().GET(
		"/api/auth/team-invitation",
		http.GetAuthenticator().GetMiddleware().MiddlewareFunc(),
		http.EmailVerificationMiddleware(makeless.GetConfig().GetConfiguration().GetEmailVerification()),
		func(c *gin.Context) {
			var err error
			var userId = http.GetAuthenticator().GetAuthUserId(c)
			var teamInvitations []*makeless_go_model.TeamInvitation
			var user = &makeless_go_model.User{
				Model:   makeless_go_model.Model{Id: userId},
				RWMutex: new(sync.RWMutex),
			}

			if teamInvitations, err = http.GetDatabase().GetTeamInvitations(http.GetDatabase().GetConnection().WithContext(c), user, teamInvitations); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			c.JSON(h.StatusOK, http.Response(nil, teamInvitations))
		},
	)

	return nil
}

func (makeless *Makeless) registerTeamInvitation(http makeless_go_http.Http) error {
	http.GetRouter().GetEngine().POST(
		"/api/team-invitation/register",
		func(c *gin.Context) {
			var err error
			var mail makeless_go_mailer.Mail
			var userExists bool
			var token = c.Query("token")
			var verified = false
			var tx = http.GetDatabase().GetConnection().WithContext(c).Begin(new(sql.TxOptions))
			var register = &_struct.Register{
				RWMutex: new(sync.RWMutex),
			}
			var teamInvitation = &makeless_go_model.TeamInvitation{
				RWMutex: new(sync.RWMutex),
			}

			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
					panic(r)
				}
			}()

			if err = tx.Error; err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if err := c.ShouldBind(register); err != nil {
				c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				return
			}

			if teamInvitation, err = http.GetDatabase().GetTeamInvitationByField(http.GetDatabase().GetConnection().WithContext(c), teamInvitation, "token", token); err != nil {
				switch errors.Is(err, gorm.ErrRecordNotFound) {
				case true:
					c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				default:
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				}
				return
			}

			if token, err = http.GetSecurity().GenerateToken(32); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			var user = &makeless_go_model.User{
				Name:     register.GetName(),
				Password: register.GetPassword(),
				Email:    register.GetEmail(),
				EmailVerification: &makeless_go_model.EmailVerification{
					Token:    &token,
					Verified: &verified,
					RWMutex:  new(sync.RWMutex),
				},
				RWMutex: new(sync.RWMutex),
			}

			if userExists, err = http.GetSecurity().UserExists(http.GetSecurity().GetDatabase().GetConnection(), "email", *user.Email); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if userExists {
				c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(makeless_go_security.UserAlreadyExists, nil))
				return
			}

			if user, err = http.GetSecurity().Register(tx, user); err != nil {
				tx.Rollback()
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if user.GetEmailVerification().RWMutex == nil {
				user.GetEmailVerification().RWMutex = new(sync.RWMutex)
			}

			var tmpUserId = user.GetId()
			var teamUser = &makeless_go_model.TeamUser{
				UserId:  &tmpUserId,
				TeamId:  teamInvitation.GetTeamId(),
				Role:    &makeless_go_security.RoleTeamUser,
				RWMutex: new(sync.RWMutex),
			}

			if err = http.GetDatabase().AddTeamUsers(tx, []*makeless_go_model.TeamUser{teamUser}, teamInvitation.GetTeam()); err != nil {
				tx.Rollback()
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if _, err = http.GetDatabase().AcceptTeamInvitation(tx, teamInvitation); err != nil {
				tx.Rollback()
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if mail, err = http.GetMailer().GetMail(
				"emailVerification", map[string]interface{}{
					"user": user,
				},
				makeless.GetConfig().GetConfiguration().GetLocale(),
			); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if err = http.GetMailer().SendQueue(mail); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if err = tx.Commit().Error; err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			c.JSON(h.StatusOK, http.Response(nil, user))
		},
	)

	return nil
}

func (makeless *Makeless) acceptTeamInvitation(http makeless_go_http.Http) error {
	http.GetRouter().GetEngine().PATCH(
		"/api/auth/team-invitation/accept",
		http.GetAuthenticator().GetMiddleware().MiddlewareFunc(),
		http.EmailVerificationMiddleware(makeless.GetConfig().GetConfiguration().GetEmailVerification()),
		func(c *gin.Context) {
			var err error
			var userId = http.GetAuthenticator().GetAuthUserId(c)
			var userEmail = http.GetAuthenticator().GetAuthEmail(c)
			var tx = http.GetDatabase().GetConnection().WithContext(c).Begin(new(sql.TxOptions))
			var user = &makeless_go_model.User{
				Model:   makeless_go_model.Model{Id: userId},
				RWMutex: new(sync.RWMutex),
			}
			var teamInvitationAccept = &_struct.TeamInvitationAccept{
				RWMutex: new(sync.RWMutex),
			}

			defer func() {
				if r := recover(); r != nil {
					tx.Rollback()
					panic(r)
				}
			}()

			if err = tx.Error; err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if err := c.ShouldBind(teamInvitationAccept); err != nil {
				c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				return
			}

			var teamInvitation = &makeless_go_model.TeamInvitation{
				Model:   makeless_go_model.Model{Id: *teamInvitationAccept.GetId()},
				Email:   &userEmail,
				RWMutex: new(sync.RWMutex),
			}

			if teamInvitation, err = http.GetDatabase().GetTeamInvitationByField(http.GetDatabase().GetConnection().WithContext(c), teamInvitation, "email", *teamInvitation.GetEmail()); err != nil {
				switch errors.Is(err, gorm.ErrRecordNotFound) {
				case true:
					c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				default:
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				}
				return
			}

			if user, err = http.GetDatabase().GetUser(http.GetDatabase().GetConnection().WithContext(c), user); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if teamInvitation, err = http.GetDatabase().AcceptTeamInvitation(tx, teamInvitation); err != nil {
				tx.Rollback()
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			var team = &makeless_go_model.Team{
				Model:   makeless_go_model.Model{Id: *teamInvitation.GetTeamId()},
				RWMutex: new(sync.RWMutex),
			}

			var teamUser = &makeless_go_model.TeamUser{
				UserId:  &userId,
				TeamId:  teamInvitation.GetTeamId(),
				Team:    teamInvitation.GetTeam(),
				User:    user,
				Role:    &makeless_go_security.RoleTeamUser,
				RWMutex: new(sync.RWMutex),
			}

			if err = http.GetDatabase().AddTeamUsers(tx, []*makeless_go_model.TeamUser{teamUser}, team); err != nil {
				tx.Rollback()
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if err = tx.Commit().Error; err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			c.JSON(h.StatusOK, http.Response(nil, team))
		},
	)

	return nil
}

func (makeless *Makeless) deleteTeamInvitation(http makeless_go_http.Http) error {
	http.GetRouter().GetEngine().DELETE(
		"/api/auth/team-invitation",
		http.GetAuthenticator().GetMiddleware().MiddlewareFunc(),
		http.EmailVerificationMiddleware(makeless.GetConfig().GetConfiguration().GetEmailVerification()),
		func(c *gin.Context) {
			var err error
			var userEmail = http.GetAuthenticator().GetAuthEmail(c)
			var teamInvitationDelete = &_struct.TeamInvitationDelete{
				RWMutex: new(sync.RWMutex),
			}

			if err := c.ShouldBind(teamInvitationDelete); err != nil {
				c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				return
			}

			var teamInvitation = &makeless_go_model.TeamInvitation{
				Model:   makeless_go_model.Model{Id: *teamInvitationDelete.GetId()},
				Email:   &userEmail,
				RWMutex: new(sync.RWMutex),
			}

			if teamInvitation, err = http.GetDatabase().GetTeamInvitationByField(http.GetDatabase().GetConnection().WithContext(c), teamInvitation, "email", *teamInvitation.GetEmail()); err != nil {
				switch errors.Is(err, gorm.ErrRecordNotFound) {
				case true:
					c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				default:
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				}
				return
			}

			if _, err = http.GetDatabase().DeleteTeamInvitation(http.GetDatabase().GetConnection().WithContext(c), teamInvitation); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			c.JSON(h.StatusOK, http.Response(nil, nil))
		},
	)

	return nil
}
