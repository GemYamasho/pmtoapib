package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Collection struct {
	Info  CollectionInfo   `json:"info"`
	Items []CollectionItem `json:"item"`
}

type CollectionInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CollectionItem struct {
	Name    string                `json:"name"`
	Request CollectionItemRequest `json:"request"`
}

type CollectionItemRequest struct {
	Url         string          `json:"url"`
	Method      string          `json:"method"`
	Header      []RequestHeader `json:"header"`
	Body        RequestBody     `json:"body"`
	Description string          `json:"description"`
}

func (r CollectionItemRequest) ShortUrl() string {
	u, _ := url.Parse(r.Url)
	return u.Path
}

func (r CollectionItemRequest) UrlParameterList() []string {
	u, _ := url.Parse(r.Url)

	parameters := []string{}

	m, _ := url.ParseQuery(u.RawQuery)

	for key, _ := range m {
		parameters = append(parameters, key)
	}

	return parameters
}

func (r CollectionItemRequest) UrlParameterListString() string {
	return strings.Join(r.UrlParameterList(), ",")
}

func (r CollectionItemRequest) ResponseBodyIncludePath() string {
	dir, file := filepath.Split(r.ShortUrl())
	file = fmt.Sprintf("%v-%v", strings.ToLower(r.Method), file)
	return fmt.Sprintf("responses%v%v.json", dir, file)
}

func (r CollectionItemRequest) ResponseBodyIncludeString() template.HTML {
	return template.HTML(fmt.Sprintf("<!-- include(%v) -->", r.ResponseBodyIncludePath()))
}

type RequestHeader struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
	Disabled    bool   `json:"disabled"`
}

type RequestBody struct {
	Mode string `json:"mode"`
	Raw  string `json:"raw"`
}

func (b RequestBody) RawString() template.HTML {
	return template.HTML(b.Raw)
}

func getApibFileContent(c Collection) string {
	tpl :=
		`# Group {{ .Info.Name }}

{{ .Info.Description }}

{{ range .Items }}
## {{ .Name }} [{{ .Request.ShortUrl }}{{ if .Request.UrlParameterListString }}{?{{ .Request.UrlParameterListString }}}{{ end }}]

### {{ if .Request.Description }}{{ .Request.Description }}{{ else }}DESCRIPTION{{ end }} [{{ .Request.Method }}]
{{ if .Request.UrlParameterList }}
+ Parameters

    {{ range .Request.UrlParameterList }}+ {{ . }} (string, required) - DESCRIPTION
    {{ end }}{{ end }}
+ Request

    + Headers
            {{ range .Request.Header }}{{ if not .Disabled }}
            {{ .Key }}: {{ .Value }}{{ end }}{{ end }}
    {{ if .Request.Body.Raw }}
    + Request (application/json)

    	    {{ .Request.Body.RawString }}
    {{ end }}
+ Response 200 (application/json)

    + Header

            NAME: VALUE

    + Body

            {{ .Request.ResponseBodyIncludeString }}




{{ end }}
`

	t := template.New("Template")
	t, _ = t.Parse(tpl)

	var doc bytes.Buffer
	t.Execute(&doc, c)
	s := doc.String()

	return s
}

func getResponseFiles(c Collection) []string {
	var files []string

	for _, item := range c.Items {
		files = append(files, item.Request.ResponseBodyIncludePath())
	}

	return files
}

func writeToFile(path string, content string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(filepath.Dir(path), os.ModePerm)
		err := ioutil.WriteFile(path, []byte(content), 0644)
		if err == nil {
			fmt.Printf("Created %v\n", path)
		}
	}
}

func main() {

	var collectionPath string
	outputPath := "./"

	if len(os.Args) > 1 {
		collectionPath = os.Args[1]
	}

	if len(os.Args) > 2 {
		outputPath = os.Args[2]
	}

	if collectionPath == "" {
		fmt.Println("No collection file defined!")
		return
	}

	file, _ := ioutil.ReadFile(collectionPath)
	var c Collection
	json.Unmarshal(file, &c)

	apibFileName := strings.Replace(c.Info.Name, " ", "-", -1)

	if len(os.Args) > 3 {
		apibFileName = os.Args[3]
	}

	apibFile := getApibFileContent(c)

	writeToFile(fmt.Sprintf("%v/%v.apib", filepath.Clean(outputPath), apibFileName), apibFile)

	for _, path := range getResponseFiles(c) {
		writeToFile(fmt.Sprintf("%v/%v", filepath.Clean(outputPath), path), "{}")
	}
}