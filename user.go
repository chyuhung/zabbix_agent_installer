package main

import "os/user"

// Get the current user's name
func GetCurrentUser() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return currentUser.Username, nil
}

// Get the current user's home directory path
func GetUserHomePath() (string, error) {
	currentUser, err := user.Current()
	if err != nil {
		return "", err
	}
	return currentUser.HomeDir, nil
}
