// Embedded assets: fonts, the icon and the changelog. (c) 2026 Ironite
package main

import (
	"bytes"
	"embed"
	"image"
	"image/png"
)

//go:embed assets/fonts assets/game CHANGELOG.md
var assetFS embed.FS

func assetBytes(name string) []byte {
	b, err := assetFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return b
}

func assetPNG(name string) image.Image {
	im, err := png.Decode(bytes.NewReader(assetBytes("assets/game/" + name)))
	if err != nil {
		panic(name + ": " + err.Error())
	}
	return im
}

func changelogText() string {
	b, _ := assetFS.ReadFile("CHANGELOG.md")
	return string(b)
}
