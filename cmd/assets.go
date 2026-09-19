package main

import (
	"embed"
	"html/template"
	"io/fs"
)

//go:embed static/* index.html restaurants.json
var embeddedFiles embed.FS

// getStaticFS "/static" 경로에 서빙할 fs.FS 반환
func getStaticFS() (fs.FS, error) {
	return fs.Sub(embeddedFiles, "static")
}

// getIndexTemplate index.html 템플릿 파싱 객체 반환
func getIndexTemplate() (*template.Template, error) {
	return template.ParseFS(embeddedFiles, "index.html")
}

// getSeedRestaurantsJSON 내장된 restaurants.json 데이터 반환
func getSeedRestaurantsJSON() []byte {
	data, err := embeddedFiles.ReadFile("restaurants.json")
	if err != nil {
		return nil
	}
	return data
}
