package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
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

type Res struct {
	Error    error
	Response string
}

func serve(cmd *cobra.Command, args []string) {
	port := config.Port

	http.HandleFunc("/", Handler)
	log.Printf("Server listening on http://localhost:%s/", port)
	log.Print(http.ListenAndServe(":"+port, nil))
}

func Handler(w http.ResponseWriter, r *http.Request) {
	reqUri := r.RequestURI
	mime := mimeTypeForFile(reqUri)
	c := make(chan *Res)

	var res *Res

	go getData(reqUri, c)
	res = <-c

	if res.Error != nil {
		http.Error(w, res.Error.Error(), 404)
		log.Fatal(res.Error)
	}

	switch mime.Category {
	case "html":
		go res.rewrite(c)
		routine := Routine{Response: <-c, Include: true, IncludeId: true}

		for routine.Include && routine.IncludeId {
			go routine.run(c)
			res = <-c
		}

	case "asset":
		go res.rewrite(c)
		res = <-c
	}

	mimeType := fmt.Sprintf(`%s; charset=utf-8`, mime.ContentType)

	w.Header().Set("Content-Type", mimeType)
	fmt.Fprint(w, res.Response)
}

func (res *Res) rewrite(c chan *Res) {
	path := config.Path
	str := res.Response
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

	c <- &Res{Error: res.Error, Response: str}
}

func (res *Res) includeReplace(c chan *Res) {
	regInc := regexp.MustCompile(`<!--#include ([a-z]+)="(\S+)" -->`)
	str := res.Response
	resBuf := []byte(str)
	incMatched := regInc.FindAllSubmatch(resBuf, -1)

	if len(incMatched) > 0 {
		for _, v := range incMatched {
			incPath := string(v[2])
			buf, err := os.ReadFile(incPath)

			handleErr(err)

			incTxt := string(buf)
			regString := fmt.Sprintf(`<!--#include ([a-z]+)="%s" -->`, incPath)
			reg := regexp.MustCompile(regString)
			str = reg.ReplaceAllString(str, incTxt)
		}
	}

	c <- &Res{Error: res.Error, Response: str}
}

func (res *Res) includeIdReplace(c chan *Res) {
	includeId := config.IncludeId
	str := res.Response
	resBuf := []byte(str)
	incMatched := [][][][]byte{}

	for _, v := range includeId {
		regString := fmt.Sprintf(`<("[^"]*"|'[^']*'|[^'">])*id="%s"("[^"]*"|'[^']*'|[^'">])*></("[^"]*"|'[^']*'|[^'">])*>`, v.K)
		regInc := regexp.MustCompile(regString)
		incMatched = append(incMatched, regInc.FindAllSubmatch(resBuf, 1))
	}

	if len(includeId) > 0 && len(incMatched) > 0 {
		for _, v := range includeId {
			buf, err := os.ReadFile(v.V)

			handleErr(err)

			incTxt := string(buf)
			regBody := regexp.MustCompile(`<body>([\s\S]*)</body>`)
			resBody := regBody.FindSubmatch(buf)

			if len(resBody) > 0 {
				incTxt = string(resBody[1])
			}

			incTag := fmt.Sprintf(`<("[^"]*"|'[^']*'|[^'">])*id="%s"("[^"]*"|'[^']*'|[^'">])*></("[^"]*"|'[^']*'|[^'">])*>`, v.K)
			reg := regexp.MustCompile(incTag)
			str = reg.ReplaceAllString(str, incTxt)
		}
	}

	c <- &Res{Error: res.Error, Response: str}
}

func getData(reqURI string, c chan *Res) {
	alternate := config.Alternate

	if reqURI == "/" {
		reqURI = "." + reqURI + "index.html"
	} else {
		reqURI = "." + reqURI
	}

	if len(alternate) > 0 {
		for _, v := range alternate {
			if strings.Contains(reqURI, v.K) {
				reqURI = strings.Replace(reqURI, v.K, v.V, 1)
				break
			}
		}
	}

	buf, err := os.ReadFile(reqURI)

	handleErr(err)

	txt := string(buf)

	c <- &Res{Error: err, Response: txt}
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
