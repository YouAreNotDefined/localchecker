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
	c := make(chan Res)

	routine := Routine{Include: true, IncludeId: true}
	var res Res

	switch mime.Category {
	case "html":
		go res.getData(reqUri, c)
		go routine.executeRoutine(<-c, c)

	case "asset":
		go res.getData(reqUri, c)
		go res.rewrite(<-c, c)

	case "other":
		go res.getData(reqUri, c)
	}

	res = <-c

	if res.Error != nil {
		fmt.Printf("error: %v\n", res.Error)
		http.Error(w, "file not found", 404)
	}

	mimeType := fmt.Sprintf(`%s; charset=utf-8`, mime.ContentType)

	w.Header().Set("Content-Type", mimeType)
	fmt.Fprint(w, res.Response)
}

func (r *Res) rewrite(res Res, c chan Res) {
	path := config.Path
	str := res.Response
	resErr := res.Error
	buf := []byte(str)

	if len(path) > 0 && resErr == nil {
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

	c <- Res{Error: resErr, Response: str}
}

func (r *Res) includeReplace(res Res, c chan Res) {
	regInc := regexp.MustCompile(`<!--#include ([a-z]+)="(\S+)" -->`)
	str := res.Response
	resErr := res.Error
	resBuf := []byte(str)
	incMatched := regInc.FindAllSubmatch(resBuf, -1)

	if len(incMatched) > 0 && resErr == nil {
		for _, v := range incMatched {
			incPath := string(v[2])
			buf, err := os.ReadFile(incPath)
			resErr = err

			if err == nil {
				incTxt := string(buf)
				regString := fmt.Sprintf(`<!--#include ([a-z]+)="%s" -->`, incPath)
				reg := regexp.MustCompile(regString)
				str = reg.ReplaceAllString(str, incTxt)
			}
		}
	}

	c <- Res{Error: resErr, Response: str}
}

func (r *Res) includeIdReplace(res Res, c chan Res) {
	includeId := config.IncludeId
	str := res.Response
	resErr := res.Error
	resBuf := []byte(str)

	if resErr == nil {
		incMatched := [][][][]byte{}

		for _, v := range includeId {
			regString := fmt.Sprintf(`<("[^"]*"|'[^']*'|[^'">])*id="%s"("[^"]*"|'[^']*'|[^'">])*></("[^"]*"|'[^']*'|[^'">])*>`, v.K)
			regInc := regexp.MustCompile(regString)
			incMatched = append(incMatched, regInc.FindAllSubmatch(resBuf, 1))
		}

		if len(includeId) > 0 && len(incMatched) > 0 {
			for _, v := range includeId {
				buf, err := os.ReadFile(v.V)
				resErr = err

				if resErr == nil {
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
		}
	}

	c <- Res{Error: resErr, Response: str}
}

func (r *Res) getData(reqURI string, c chan Res) *Res {
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
	txt := string(buf)

	return &Res{Error: err, Response: txt}
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
