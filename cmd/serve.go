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
		replaceIncludeTag, replaceIncludeId := res.needsReplace()

		for replaceIncludeTag || replaceIncludeId {
			if replaceIncludeTag {
				go res.ReplaceIncludeTag(c)
				res = <-c
			}
			if replaceIncludeId {
				go res.ReplaceIncludeId(c)
				res = <-c
			}

			go res.rewrite(c)
			res = <-c
			replaceIncludeTag, replaceIncludeId = res.needsReplace()
		}
	case JsType, JsonType:
		go res.rewrite(c)
		res = <-c
	}

	w.Header().Set("Content-Type", mimeType)
	fmt.Fprint(w, res.Body)
}

func (res *Response) rewrite(c chan *Response) {
	path := config.Path
	str := res.Body
	buf := []byte(str)

	if len(path) > 0 {
		matchPathCount := 0

		for _, v := range path {
			regPath := regexp.MustCompile(v.K)
			matchPathCount += len(regPath.FindAll(buf, -1))
		}
		if matchPathCount > 0 {
			for _, v := range path {
				str = strings.Replace(str, v.K, v.V, -1)
			}
		}
	}
	c <- &Response{Error: res.Error, Body: str}
}

func (res *Response) ReplaceIncludeTag(c chan *Response) {
	var (
		err error
		buf []byte
	)
	regInc := regexp.MustCompile(IncludeTagRegex)
	str := res.Body
	resBuf := []byte(str)
	matchIncludeTags := regInc.FindAllSubmatch(resBuf, -1)

	if len(matchIncludeTags) > 0 {
		for _, v := range matchIncludeTags {
			incPath := string(v[2])
			buf, err = os.ReadFile(incPath)
			handleErr(err)

			if err == nil {
				incTxt := string(buf)
				regString := fmt.Sprintf(`<!--#include ([a-z]+)="%s" -->`, incPath)
				reg := regexp.MustCompile(regString)
				str = reg.ReplaceAllString(str, incTxt)
			}
		}
	}

	c <- &Response{Error: err, Body: str}
}

func (res *Response) ReplaceIncludeId(c chan *Response) {
	var (
		err error
		buf []byte
	)
	includeId := config.IncludeId
	str := res.Body
	resBuf := []byte(str)
	matchCountId := 0

	if len(includeId) > 0 {
		for _, v := range includeId {
			regString := makeIncludeTag(v.K)
			regInc := regexp.MustCompile(regString)
			matchCountId += len(regInc.FindAll(resBuf, -1))
		}

		if matchCountId > 0 {
			for _, v := range includeId {
				buf, err = os.ReadFile(v.V)
				handleErr(err)

				if err == nil {
					incTxt := string(buf)
					regBody := regexp.MustCompile(`<body>([\s\S]*)</body>`)
					resBody := regBody.FindSubmatch(buf)

					if len(resBody) > 0 {
						incTxt = string(resBody[1])
					}
					incTag := makeIncludeTag(v.K)
					reg := regexp.MustCompile(incTag)
					str = reg.ReplaceAllString(str, incTxt)
				}
			}
		}
	}
	c <- &Response{Error: err, Body: str}
}

func getData(reqURI string, c chan *Response) {
	alternate := config.Alternate
	reqURI = fmt.Sprintf(`.%s`, reqURI)

	if len(alternate) > 0 {
		for _, v := range alternate {
			if strings.Contains(reqURI, v.K) {
				reqURI = strings.Replace(reqURI, v.K, v.V, 1)
				break
			}
		}
	}

	buf, err := os.ReadFile(reqURI)
	txt := string(buf)
	c <- &Response{Error: err, Body: txt}
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
	resStr := res.Body
	resBuf := []byte(resStr)
	includeId := config.IncludeId
	matchCountId := 0
	matchCountInc := 0

	if len(includeId) > 0 {
		for _, v := range includeId {
			regString := makeIncludeTag(v.K)
			regIncId := regexp.MustCompile(regString)
			matchCountId += len(regIncId.FindAll(resBuf, -1))
			if isNotFileExist(v.V) {
				matchCountId -= 1
			}
		}
	}

	regInc := regexp.MustCompile(IncludeTagRegex)
	matchIncludeTags := regInc.FindAllSubmatch(resBuf, -1)
	matchCountInc += len(matchIncludeTags)
	if matchCountInc > 0 {
		for _, v := range matchIncludeTags {
			if isNotFileExist(string(v[2])) {
				matchCountInc -= 1
			}
		}
	}

	// return need to do ReplaceIncludeTag(), ReplaceIncludeId()
	if matchCountInc == 0 && matchCountId == 0 {
		return false, false
	} else if matchCountInc == 0 {
		return false, true
	} else if matchCountId == 0 {
		return true, false
	} else {
		return true, true
	}
}

func isNotFileExist(path string) bool {
	_, err := os.Stat(path)
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
