package main

import "html/template"

type Image struct {
	Src, Alt string
}

type Spec struct {
	Label, Content template.HTML
}

type Work struct {
	Title, About template.HTML
	Web          template.URL
	Blank        bool
	Image        Image
	Specs        []Spec
}

type TplData struct {
	Works []Work
	Lang  string
}

type WorksMap map[string][]Work

// SetLang applies provided lang to d.
// If d is nil, a new one is created with provided lang.
func (d *TplData) SetLang(lang string) *TplData {
	if d == nil {
		return &TplData{Lang: lang}
	}
	d.Lang = lang
	return d
}

func NewSpec(label, content string) Spec {
	return Spec{
		Label:   template.HTML(label),
		Content: template.HTML(content),
	}
}

func NewWork(title, web, about, isrc, ialt string, specs ...Spec) Work {
	return Work{
		Title: template.HTML(title),
		Web:   template.URL(web),
		About: template.HTML(about),
		Image: Image{
			Src: isrc,
			Alt: ialt,
		},
		Specs: specs,
	}
}
