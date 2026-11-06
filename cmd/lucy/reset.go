package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"lucy/internal/auth"
)

// resetPassword runs on the server, so shell access is the proof of ownership.
// It works while Lucy is running and ends every session.
func resetPassword(*slog.Logger) error {
	password, err := askPassword()
	if err != nil {
		return err
	}
	if !auth.ValidPassword(password) {
		return fmt.Errorf("şifre en az %d karakter olmalı", auth.MinPasswordLength)
	}
	hash, err := auth.Hash(password)
	if err != nil {
		return err
	}
	db, err := openStore()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.ReplacePassword(context.Background(), hash, time.Now()); err != nil {
		return err
	}
	fmt.Println("Şifre değişti, tüm oturumlar kapatıldı.")
	return nil
}

func askPassword() (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return "", err
		}
		return strings.TrimRight(line, "\r\n"), nil
	}
	first, err := prompt(fd, "Yeni şifre: ")
	if err != nil {
		return "", err
	}
	second, err := prompt(fd, "Şifre (tekrar): ")
	if err != nil {
		return "", err
	}
	if first != second {
		return "", errors.New("iki şifre eşleşmiyor")
	}
	return first, nil
}

func prompt(fd int, label string) (string, error) {
	fmt.Print(label)
	data, err := term.ReadPassword(fd)
	fmt.Println()
	return string(data), err
}
