package cmd

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start up the http server. File names can be omitted.",
	Long:  "Start up the http server. File names can be omitted.",
	Run:   serve,
}

const (
	HtmlType        = "text/html"
	CssType         = "text/css"
	JsType          = "application/javascript"
	JsonType        = "application/json"
	IncludeTagRegex = `<!--#include ([a-z]+)="(\S+)" -->`
)

type Response struct {
	Error error
	Body  string
}

func serve(cmd *cobra.Command, args []string) {
	port := config.Port

	http.HandleFunc("/", handler)
	log.Printf("Server listening on http://localhost:%s/", port)
	log.Print(http.ListenAndServe(":"+port, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	var res *Response
	reqURI := r.RequestURI
	c := make(chan *Response)
	defer close(c)

	if isDir, err := isDirExist(r.RequestURI); isDir {
		reqURI = fmt.Sprintf(`%sindex.html`, r.RequestURI)
	} else {
		handleErr(err)
	}

	ext := filepath.Ext(reqURI)
	mimeType := mime.TypeByExtension(ext)
	mediatype, _, err := mime.ParseMediaType(mimeType)
	handleErr(err)

	go getData(reqURI, c)
	res = <-c
	handleErr(res.Error)

	switch mediatype {
	case HtmlType:
		go res.rewrite(c)
		res = <-c
		needsReplaceIncTag, needsReplaceIncId := res.needsReplace()

		for needsReplaceIncTag || needsReplaceIncId {
			if needsReplaceIncTag {
				go res.ReplaceIncludeTag(c)
				res = <-c
			}
			if needsReplaceIncId {
				go res.ReplaceIncludeId(c)
				res = <-c
			}

			go res.rewrite(c)
			res = <-c
			needsReplaceIncTag, needsReplaceIncId = res.needsReplace()
		}
	case CssType, JsType, JsonType:
		go res.rewrite(c)
		res = <-c
	}

	w.Header().Set("Content-Type", mimeType)
	fmt.Fprint(w, res.Body)
}

func (res *Response) rewrite(c chan *Response) {
	pathes := config.Path
	stringBody := res.Body
	bufferedBody := []byte(stringBody)

	if len(pathes) > 0 {
		matchPathCount := 0

		for _, v := range pathes {
			pathRep := regexp.MustCompile(v.K)
			matchPathCount += len(pathRep.FindAll(bufferedBody, -1))
		}
		if matchPathCount > 0 {
			for _, v := range pathes {
				stringBody = strings.Replace(stringBody, v.K, v.V, -1)
			}
		}
	}
	c <- &Response{Error: res.Error, Body: stringBody}
}

func (res *Response) ReplaceIncludeTag(c chan *Response) {
	var (
		err          error
		bufferedFile []byte
	)
	includeTagRep := regexp.MustCompile(IncludeTagRegex)
	stringBody := res.Body
	bufferedBody := []byte(stringBody)
	matchIncludeTags := includeTagRep.FindAllSubmatch(bufferedBody, -1)

	if len(matchIncludeTags) > 0 {
		for _, v := range matchIncludeTags {
			includeTagPath := string(v[2])
			bufferedFile, err = os.ReadFile(includeTagPath)
			handleErr(err)

			if err == nil {
				incTxt := string(bufferedFile)
				includeTagReg := fmt.Sprintf(`<!--#include ([a-z]+)="%s" -->`, includeTagPath)
				rep := regexp.MustCompile(includeTagReg)
				stringBody = rep.ReplaceAllString(stringBody, incTxt)
			}
		}
	}
	c <- &Response{Error: err, Body: stringBody}
}

func (res *Response) ReplaceIncludeId(c chan *Response) {
	var (
		err          error
		bufferedFile []byte
	)
	includeIds := config.IncludeId
	stringBody := res.Body
	bufferedBody := []byte(stringBody)
	matchIncludeIdCount := 0

	if len(includeIds) > 0 {
		for _, v := range includeIds {
			includeIdTagReg := makeIncludeTag(v.K)
			includeIdTagRep := regexp.MustCompile(includeIdTagReg)
			matchIncludeIdCount += len(includeIdTagRep.FindAll(bufferedBody, -1))
		}
		if matchIncludeIdCount > 0 {
			for _, v := range includeIds {
				bufferedFile, err = os.ReadFile(v.V)
				handleErr(err)

				if err == nil {
					stringFile := string(bufferedFile)
					bodyRep := regexp.MustCompile(`<body>([\s\S]*)</body>`)
					fileBody := bodyRep.FindSubmatch(bufferedFile)

					if len(fileBody) > 0 {
						stringFile = string(fileBody[1])
					}
					includeIdTagReg := makeIncludeTag(v.K)
					includeIdTagRep := regexp.MustCompile(includeIdTagReg)
					stringBody = includeIdTagRep.ReplaceAllString(stringBody, stringFile)
				}
			}
		}
	}
	c <- &Response{Error: err, Body: stringBody}
}

func getData(reqURI string, c chan *Response) {
	alternates := config.Alternate
	reqURI = fmt.Sprintf(`.%s`, reqURI)

	if len(alternates) > 0 {
		for _, v := range alternates {
			if strings.Contains(reqURI, v.K) {
				reqURI = strings.Replace(reqURI, v.K, v.V, 1)
				break
			}
		}
	}
	bufferedFile, err := os.ReadFile(reqURI)
	c <- &Response{Error: err, Body: string(bufferedFile)}
}

func isDirExist(path string) (bool, error) {
	currentDir, _ := os.Getwd()
	path = fmt.Sprintf(`%s%s`, currentDir, path)
	info, err := os.Stat(path)

	if err != nil {
		if !os.IsExist(err) {
			return false, nil
		} else {
			return false, err
		}
	} else {
		if !info.IsDir() {
			return false, nil
		}
	}
	return true, nil
}

func (res *Response) needsReplace() (bool, bool) {
	stringBody := res.Body
	bufferedBody := []byte(stringBody)
	includeIds := config.IncludeId
	matchIncludeIdCount := 0
	matchIncludeCount := 0

	if len(includeIds) > 0 {
		for _, v := range includeIds {
			includeIdTagReg := makeIncludeTag(v.K)
			includeIdTagRep := regexp.MustCompile(includeIdTagReg)
			matchIncludeIdCount += len(includeIdTagRep.FindAll(bufferedBody, -1))
			if isNotFileExist(v.V) {
				matchIncludeIdCount -= 1
			}
		}
	}
	includeTagReg := regexp.MustCompile(IncludeTagRegex)
	matchIncludeTags := includeTagReg.FindAllSubmatch(bufferedBody, -1)
	matchIncludeCount += len(matchIncludeTags)
	if matchIncludeCount > 0 {
		for _, v := range matchIncludeTags {
			if isNotFileExist(string(v[2])) {
				matchIncludeCount -= 1
			}
		}
	}

	// return need to do ReplaceIncludeTag(), ReplaceIncludeId()
	if matchIncludeCount == 0 && matchIncludeIdCount == 0 {
		return false, false
	} else if matchIncludeCount == 0 {
		return false, true
	} else if matchIncludeIdCount == 0 {
		return true, false
	} else {
		return true, true
	}
}

func isNotFileExist(URI string) bool {
	_, err := os.Stat(URI)
	return os.IsNotExist(err)
}

func handleErr(err error) {
	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

func makeIncludeTag(value string) string {
	return fmt.Sprintf(`<("[^"]*"|'[^']*'|[^'">])*id="%s"("[^"]*"|'[^']*'|[^'">])*></("[^"]*"|'[^']*'|[^'">])*>`, value)
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
