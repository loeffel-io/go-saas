package go_saas

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-saas/go-saas/http"
	"github.com/go-saas/go-saas/mailer"
	"github.com/go-saas/go-saas/model"
	"github.com/go-saas/go-saas/security"
	"github.com/go-saas/go-saas/struct"
	"github.com/jinzhu/gorm"
	h "net/http"
	"strconv"
	"sync"
	"time"
)

func (saas *Saas) teamInvitationsTeam(http go_saas_http.Http) error {
	http.GetRouter().GET(
		"/api/auth/team/team-invitation",
		http.GetAuthenticator().GetMiddleware().MiddlewareFunc(),
		http.EmailVerificationMiddleware(saas.GetConfig().GetConfiguration().GetEmailVerification()),
		http.TeamRoleMiddleware(go_saas_security.RoleTeamOwner),
		func(c *gin.Context) {
			var err error
			var userId = http.GetAuthenticator().GetAuthUserId(c)
			var teamId, _ = strconv.Atoi(c.GetHeader("Team"))
			var teamInvitations []*go_saas_model.TeamInvitation
			var team = &go_saas_model.Team{
				Model:   go_saas_model.Model{Id: uint(teamId)},
				UserId:  &userId,
				RWMutex: new(sync.RWMutex),
			}

			if teamInvitations, err = http.GetDatabase().GetTeamInvitationsTeam(http.GetDatabase().GetConnection(), team, teamInvitations); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			c.JSON(h.StatusOK, http.Response(nil, teamInvitations))
		},
	)

	return nil
}

func (saas *Saas) createTeamInvitationsTeam(http go_saas_http.Http) error {
	http.GetRouter().POST(
		"/api/auth/team/team-invitation",
		http.GetAuthenticator().GetMiddleware().MiddlewareFunc(),
		http.EmailVerificationMiddleware(saas.GetConfig().GetConfiguration().GetEmailVerification()),
		http.TeamRoleMiddleware(go_saas_security.RoleTeamOwner),
		func(c *gin.Context) {
			var err error
			var user *go_saas_model.User
			var userId = http.GetAuthenticator().GetAuthUserId(c)
			var teamId, _ = strconv.Atoi(c.GetHeader("Team"))
			var teamInvitationExpire = time.Now().Add(time.Hour * 24 * 7)
			var teamInvitationAccepted = false
			var userTeamInvite = &_struct.TeamInvitationTeamCreate{
				RWMutex: new(sync.RWMutex),
			}
			var team = &go_saas_model.Team{
				Model:   go_saas_model.Model{Id: uint(teamId)},
				RWMutex: new(sync.RWMutex),
			}

			if err = c.ShouldBind(userTeamInvite); err != nil {
				c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				return
			}

			if team, err = http.GetDatabase().GetTeam(http.GetDatabase().GetConnection(), team); err != nil {
				c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				return
			}

			for _, teamUser := range team.GetTeamUsers() {
				teamUser.RWMutex = new(sync.RWMutex)
				teamUser.GetUser().RWMutex = new(sync.RWMutex)

				if userId == teamUser.GetUser().GetId() {
					user = teamUser.GetUser()
				}
			}

			var teamInvitations = make([]*go_saas_model.TeamInvitation, len(userTeamInvite.GetInvitations()))
			var index = make(map[string]bool)
			for _, teamInvitation := range team.GetTeamInvitations() {
				teamInvitation.RWMutex = new(sync.RWMutex)
				index[*teamInvitation.GetEmail()] = true
			}

			for i, teamInvitation := range userTeamInvite.GetInvitations() {
				var mail go_saas_mailer.Mail
				var token string
				var userInvited = &go_saas_model.User{
					RWMutex: new(sync.RWMutex),
				}
				teamInvitation.RWMutex = new(sync.RWMutex)

				if _, exists := index[*teamInvitation.GetEmail()]; exists {
					c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(nil, nil))
					return
				}

				if token, err = http.GetSecurity().GenerateToken(32); err != nil {
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
					return
				}

				teamInvitations[i] = &go_saas_model.TeamInvitation{
					UserId:   &userId,
					Email:    teamInvitation.GetEmail(),
					Token:    &token,
					Expire:   &teamInvitationExpire,
					Accepted: &teamInvitationAccepted,
					RWMutex:  new(sync.RWMutex),
				}

				if userInvited, err = http.GetDatabase().GetUserByField(http.GetDatabase().GetConnection(), userInvited, "email", *teamInvitations[i].GetEmail()); err != nil && err != gorm.ErrRecordNotFound {
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
					return
				}

				if mail, err = http.GetMailer().GetMail("teamInvitation", map[string]interface{}{
					"user":           user,
					"userInvited":    userInvited,
					"teamName":       *team.GetName(),
					"teamInvitation": teamInvitations[i],
				}); err != nil {
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
					return
				}

				if err = http.GetMailer().Send(c, mail); err != nil {
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
					return
				}
			}

			if team, err = http.GetDatabase().AddTeamInvitations(http.GetDatabase().GetConnection(), team, teamInvitations); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			c.JSON(h.StatusOK, http.Response(nil, team))
		},
	)

	return nil
}

func (saas *Saas) resendTeamInvitationTeam(http go_saas_http.Http) error {
	http.GetRouter().POST(
		"/api/auth/team/team-invitation/resend",
		http.GetAuthenticator().GetMiddleware().MiddlewareFunc(),
		http.EmailVerificationMiddleware(saas.GetConfig().GetConfiguration().GetEmailVerification()),
		http.TeamRoleMiddleware(go_saas_security.RoleTeamOwner),
		func(c *gin.Context) {
			var err error
			var mail go_saas_mailer.Mail
			var teamId, _ = strconv.Atoi(c.GetHeader("Team"))
			var teamInvitationTeamResend = &_struct.TeamInvitationTeamResend{
				RWMutex: new(sync.RWMutex),
			}
			var userInvited = &go_saas_model.User{
				RWMutex: new(sync.RWMutex),
			}

			if err := c.ShouldBind(teamInvitationTeamResend); err != nil {
				c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				return
			}

			var tmpTeamId = uint(teamId)
			var teamInvitation = &go_saas_model.TeamInvitation{
				Model:   go_saas_model.Model{Id: *teamInvitationTeamResend.GetId()},
				TeamId:  &tmpTeamId,
				RWMutex: new(sync.RWMutex),
			}

			if teamInvitation, err = http.GetDatabase().GetTeamInvitationByField(http.GetDatabase().GetConnection(), teamInvitation, "team_id", fmt.Sprintf("%d", *teamInvitation.GetTeamId())); err != nil {
				switch err {
				case gorm.ErrRecordNotFound:
					c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				default:
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				}
				return
			}

			if userInvited, err = http.GetDatabase().GetUserByField(http.GetDatabase().GetConnection(), userInvited, "email", *teamInvitation.GetEmail()); err != nil && err != gorm.ErrRecordNotFound {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			teamInvitation.GetUser().RWMutex, teamInvitation.GetTeam().RWMutex = new(sync.RWMutex), new(sync.RWMutex)
			if mail, err = http.GetMailer().GetMail("teamInvitation", map[string]interface{}{
				"user":           teamInvitation.GetUser(),
				"userInvited":    userInvited,
				"teamName":       *teamInvitation.GetTeam().GetName(),
				"teamInvitation": teamInvitation,
			}); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			if err = http.GetMailer().Send(c, mail); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			c.JSON(h.StatusOK, http.Response(nil, teamInvitation))
		},
	)

	return nil
}

func (saas *Saas) deleteTeamInvitationTeam(http go_saas_http.Http) error {
	http.GetRouter().DELETE(
		"/api/auth/team/team-invitation",
		http.GetAuthenticator().GetMiddleware().MiddlewareFunc(),
		http.EmailVerificationMiddleware(saas.GetConfig().GetConfiguration().GetEmailVerification()),
		http.TeamRoleMiddleware(go_saas_security.RoleTeamOwner),
		func(c *gin.Context) {
			var err error
			var teamId, _ = strconv.Atoi(c.GetHeader("Team"))
			var teamInvitationTeamDelete = &_struct.TeamInvitationTeamDelete{
				RWMutex: new(sync.RWMutex),
			}

			if err := c.ShouldBind(teamInvitationTeamDelete); err != nil {
				c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				return
			}

			var tmpTeamId = uint(teamId)
			var teamInvitation = &go_saas_model.TeamInvitation{
				Model:   go_saas_model.Model{Id: *teamInvitationTeamDelete.GetId()},
				TeamId:  &tmpTeamId,
				RWMutex: new(sync.RWMutex),
			}

			if teamInvitation, err = http.GetDatabase().GetTeamInvitationByField(http.GetDatabase().GetConnection(), teamInvitation, "team_id", fmt.Sprintf("%d", *teamInvitation.GetTeamId())); err != nil {
				switch err {
				case gorm.ErrRecordNotFound:
					c.AbortWithStatusJSON(h.StatusBadRequest, http.Response(err, nil))
				default:
					c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				}
				return
			}

			if _, err = http.GetDatabase().DeleteTeamInvitation(http.GetDatabase().GetConnection(), teamInvitation); err != nil {
				c.AbortWithStatusJSON(h.StatusInternalServerError, http.Response(err, nil))
				return
			}

			c.JSON(h.StatusOK, http.Response(nil, nil))
		},
	)

	return nil
}