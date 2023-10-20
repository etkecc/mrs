package utils

import (
	"bytes"
	"io"
	"strings"

	"github.com/h2non/bimg"
)

const (
	AvatarWidth  = 40
	AvatarHeight = 40
	AvatarMIME   = "image/webp"
)

var avatarConfig = bimg.Options{
	Width:     AvatarWidth,
	Height:    AvatarHeight,
	Type:      bimg.WEBP,
	Crop:      true,
	Enlarge:   true,
	Interlace: true,
}

// Server returns server name from the matrix ID (room id/alias, user ID, etc)
func ServerFrom(matrixID string) string {
	idx := strings.LastIndex(matrixID, ":")
	if idx == -1 {
		return ""
	}
	if idx+2 == len(matrixID) { // "wrongid:"
		return ""
	}
	return matrixID[idx+1:]
}

// Avatar resizes and converts avatar stream to webp
func Avatar(avatarStream io.Reader) (io.Reader, bool) {
	avatarRaw, err := io.ReadAll(avatarStream)
	if err != nil {
		return avatarStream, false
	}

	avatar, err := bimg.NewImage(avatarRaw).Process(avatarConfig)
	if err != nil {
		return bytes.NewReader(avatarRaw), false
	}
	return bytes.NewReader(avatar), true
}
