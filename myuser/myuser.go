package myuser

import "os/user"

// 获取当前用户用户名
func GetCurrentUser() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return currentUser.Username, nil
}

// 获取当前用户家目录
func GetUserHomePath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return currentUser.HomeDir, nil
}
