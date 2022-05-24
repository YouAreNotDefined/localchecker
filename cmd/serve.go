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
	Long:  `Start up the http server. File names can be omitted.`,
	Run:   serve,
}

const (
	HtmlType      = "text/html"
	JsType        = "application/javascript"
	JsonType      = "application/json"
	IncludeTagReg = `<!--#include ([a-z]+)="(\S+)" -->`
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
	var (
		reqURI string
		res    *Response
	)
	c := make(chan *Response)

	if err := isDirExist(r.RequestURI); err == nil {
		reqURI = fmt.Sprintf(`%sindex.html`, r.RequestURI)
	} else {
		http.Error(w, fmt.Sprintf("Error: %v", err), 500)
		handleErr(err)
	}

	ext := filepath.Ext(reqURI)
	mimeType := mime.TypeByExtension(ext)
	mediatype, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %v", err), 500)
		handleErr(err)
	}

	go getData(reqURI, c)
	res = <-c

	if res.Error != nil {
		http.Error(w, fmt.Sprintf("Error: %v", res.Error.Error()), 500)
		handleErr(res.Error)
	}

	switch mediatype {
	case HtmlType:
		go res.rewrite(c)
		routine := Routine{Response: <-c, Include: true, IncludeId: true}

		for routine.Include && routine.IncludeId {
			go routine.run(c)
			res = <-c
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
		pathMatched := [][][][]byte{}

		for _, v := range path {
			regPath := regexp.MustCompile(v.K)
			pathMatched = append(pathMatched, regPath.FindAllSubmatch(buf, 1))
		}

		if len(pathMatched) > 0 {
			for _, v := range path {
				str = strings.Replace(str, v.K, v.V, -1)
			}
		}
	}

	c <- &Response{Error: nil, Body: str}
}

func (res *Response) ReplaceIncludeTag(c chan *Response) {
	var (
		err error
		buf []byte
	)
	regInc := regexp.MustCompile(IncludeTagReg)
	str := res.Body
	resBuf := []byte(str)
	incMatched := regInc.FindAllSubmatch(resBuf, -1)

	if len(incMatched) > 0 {
		for _, v := range incMatched {
			incPath := string(v[2])
			buf, err = os.ReadFile(incPath)

			incTxt := string(buf)
			regString := fmt.Sprintf(`<!--#include ([a-z]+)="%s" -->`, incPath)
			reg := regexp.MustCompile(regString)
			str = reg.ReplaceAllString(str, incTxt)
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
	incMatched := [][][][]byte{}

	for _, v := range includeId {
		regString := makeIncludeTag(v.K)
		regInc := regexp.MustCompile(regString)
		incMatched = append(incMatched, regInc.FindAllSubmatch(resBuf, 1))
	}

	if len(includeId) > 0 && len(incMatched) > 0 {
		for _, v := range includeId {
			buf, err = os.ReadFile(v.V)

			handleErr(err)

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

func isDirExist(path string) error {
	currentDir, _ := os.Getwd()
	path = fmt.Sprintf(`%s%s`, currentDir, path)
	info, err := os.Stat(path)

	if err != nil {
		if !os.IsExist(err) {
			return nil
		} else {
			return err
		}
	} else {
		if !info.IsDir() {
			return nil
		}
	}
	return nil
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
