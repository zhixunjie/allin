package service

import (
	"errors"
	"strings"
	"time"
	"unicode"

	"github.com/allin/server/base/biz/dao"
	"github.com/allin/server/base/biz/model"
	"github.com/allin/server/contrib/auth"
	"github.com/spf13/viper"
)

type UserSvc struct{}

func NewUserSvc() *UserSvc { return &UserSvc{} }

// Register 校验注册参数，创建用户（初始筹码 10000），返回用户信息和 JWT。
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

// Login 验证账号密码，成功后签发 JWT。
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

// GetByID 根据用户 ID 查询用户信息。
func (svc *UserSvc) GetByID(id int64) (*model.User, error) {
	return dao.UserDao.GetByID(id)
}

// ClaimFreeChips 在余额不足 1000 时补发 10000 筹码（原子操作，防并发重复领取）。
// 返回补发后的新余额。
func (svc *UserSvc) ClaimFreeChips(userID int64) (int64, error) {
	const threshold = int64(1000)
	const award = int64(10000)
	ok, err := dao.UserDao.ClaimChipsIfBelow(userID, threshold, award, "claim_free")
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, errors.New("sufficient balance")
	}
	u, err := dao.UserDao.GetByID(userID)
	if err != nil {
		return 0, err
	}
	return u.ChipBalance, nil
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
