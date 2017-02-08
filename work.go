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

type Data struct {
	Works []Work
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
