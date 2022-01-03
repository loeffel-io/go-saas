package makeless_go

import (
	"fmt"
	"github.com/makeless/makeless-go/mailer"
	"github.com/makeless/makeless-go/mailer/basic"
	"github.com/makeless/makeless-go/model"
	"github.com/matcornic/hermes/v2"
	"sync"
)

func (makeless *Makeless) mailTeamInvitation(data map[string]interface{}) (makeless_go_mailer.Mail, error) {
	var err error
	var name, link, message, messageHtml string
	var user = data["user"].(*makeless_go_model.User)
	var userInvited = data["userInvited"].(*makeless_go_model.User)
	var teamName = data["teamName"].(string)
	var teamInvitation = data["teamInvitation"].(*makeless_go_model.TeamInvitation)
	var messages = map[string]map[string]string{
		"en": {
			"subject": fmt.Sprintf(
				"%s invited you to %s",
				*user.GetName(),
				teamName,
			),
			"intro": fmt.Sprintf(
				"%s has invited you to %s.",
				*user.GetName(),
				teamName,
			),
			"outro": fmt.Sprintf(
				"Note: This invitation was intended for %s. If you were not expecting this invitation, you can ignore this email.",
				*teamInvitation.GetEmail(),
			),
			"instruction": "You can accept or decline this invitation. This invitation will expire in 7 days.",
			"button":      "View invitation",
		},
	}

	switch userInvited.GetName() {
	case nil:
		link = fmt.Sprintf("%s/invitation?token=%s", makeless.GetConfig().GetConfiguration().GetMail().GetLink(), *teamInvitation.GetToken())
	default:
		name = *userInvited.GetName()
		link = fmt.Sprintf("%s/settings/team-invitation", makeless.GetConfig().GetConfiguration().GetMail().GetLink())
	}

	config := hermes.Hermes{
		Product: hermes.Product{
			Name:      makeless.GetConfig().GetConfiguration().GetMail().GetName(),
			Link:      makeless.GetConfig().GetConfiguration().GetMail().GetLink(),
			Logo:      makeless.GetConfig().GetConfiguration().GetMail().GetLogo(),
			Copyright: makeless.GetConfig().GetConfiguration().GetMail().GetTexts(makeless.GetConfig().GetConfiguration().GetLocale()).GetCopyright(),
		},
	}

	email := hermes.Email{
		Body: hermes.Body{
			Name:      name,
			Greeting:  makeless.GetConfig().GetConfiguration().GetMail().GetTexts(makeless.GetConfig().GetConfiguration().GetLocale()).GetGreeting(),
			Signature: makeless.GetConfig().GetConfiguration().GetMail().GetTexts(makeless.GetConfig().GetConfiguration().GetLocale()).GetSignature(),
			Intros:    []string{messages[makeless.GetConfig().GetConfiguration().GetLocale()]["intro"]},
			Actions: []hermes.Action{
				{
					Instructions: messages[makeless.GetConfig().GetConfiguration().GetLocale()]["instruction"],
					Button: hermes.Button{
						Text:      messages[makeless.GetConfig().GetConfiguration().GetLocale()]["button"],
						Link:      link,
						Color:     makeless.GetConfig().GetConfiguration().GetMail().GetButtonColor(),
						TextColor: makeless.GetConfig().GetConfiguration().GetMail().GetButtonTextColor(),
					},
				},
			},
			Outros: []string{messages[makeless.GetConfig().GetConfiguration().GetLocale()]["outro"]},
		},
	}

	if message, err = config.GeneratePlainText(email); err != nil {
		return nil, err
	}

	if messageHtml, err = config.GenerateHTML(email); err != nil {
		return nil, err
	}

	return &makeless_go_mailer_basic.Mail{
		To:          []string{*teamInvitation.GetEmail()},
		From:        makeless.GetConfig().GetConfiguration().GetMail().GetFrom(),
		Subject:     messages[makeless.GetConfig().GetConfiguration().GetLocale()]["subject"],
		Message:     []byte(message),
		HtmlMessage: []byte(messageHtml),
		RWMutex:     new(sync.RWMutex),
	}, nil
}
