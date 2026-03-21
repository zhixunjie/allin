package service

import (
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/allin/server/base/biz/dao"
	"github.com/allin/server/base/biz/model"
	"github.com/allin/server/contrib/auth"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type UserSvc struct{}

func NewUserSvc() *UserSvc { return &UserSvc{} }

// Register validates input, creates user with 10 000 chip welcome bonus, issues JWT.
func (svc *UserSvc) Register(username, password, displayName string) (*model.User, string, error) {
	username = strings.TrimSpace(username)
	displayName = strings.TrimSpace(displayName)

	if err := validateRegister(username, password, displayName); err != nil {
		return nil, "", err
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, "", errors.New("server error")
	}

	u := &model.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: hash,
		DisplayName:  displayName,
		ChipBalance:  10000,
		CreatedAt:    time.Now(),
	}
	if err := dao.UserDao.Create(u); err != nil {
		return nil, "", err
	}

	token, err := auth.IssueToken(viper.GetString("jwt.secret"), u.ID, u.DisplayName)
	if err != nil {
		return nil, "", errors.New("server error")
	}
	return u, token, nil
}

// Login verifies credentials and issues JWT.
func (svc *UserSvc) Login(username, password string) (*model.User, string, error) {
	u, err := dao.UserDao.GetByUsername(username)
	if err != nil {
		return nil, "", errors.New("invalid username or password")
	}
	if !auth.CheckPassword(password, u.PasswordHash) {
		return nil, "", errors.New("invalid username or password")
	}

	token, err := auth.IssueToken(viper.GetString("jwt.secret"), u.ID, u.DisplayName)
	if err != nil {
		return nil, "", errors.New("server error")
	}
	return u, token, nil
}

// GetByID returns the user with the given ID.
func (svc *UserSvc) GetByID(id string) (*model.User, error) {
	return dao.UserDao.GetByID(id)
}

func validateRegister(username, password, displayName string) error {
	if len(username) < 3 || len(username) > 32 {
		return errors.New("username must be 3-32 characters")
	}
	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return errors.New("username may only contain letters, digits and underscores")
		}
	}
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if len(displayName) < 1 || len(displayName) > 32 {
		return errors.New("display_name must be 1-32 characters")
	}
	return nil
}
