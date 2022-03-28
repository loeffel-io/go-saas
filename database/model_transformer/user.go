package makeless_go_model_transformer

import (
	"github.com/makeless/makeless-go/database/model"
	"github.com/makeless/makeless-go/proto/basic"
)

type UserTransformer interface {
	CreateUserRequestToUser(createUserRequest *makeless.CreateUserRequest, token string) (*makeless_go_model.User, error)
	UserToUser(user *makeless_go_model.User) (*makeless.User, error)
}