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

type Necessary struct {
	include   bool
	includeId bool
}

func serve(cmd *cobra.Command, args []string) {
	port := config.Port
	http.HandleFunc("/", Handler)
	log.Printf("Server listening on http://localhost:%s/", port)
	log.Print(http.ListenAndServe(":"+port, nil))
}

func Handler(w http.ResponseWriter, r *http.Request) {
	rImg := regexp.MustCompile(`.png|.jpg|.svg|.gif|.webp|.ico`)
	rOther := regexp.MustCompile(`.css|.js|.json`)
	reqUri := r.RequestURI
	c := make(chan Res)
	var res Res

	if rImg.MatchString(reqUri) {
		go getData(reqUri, c)
		res = <-c

	} else if rOther.MatchString(reqUri) {
		if strings.Contains(reqUri, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		} else if strings.Contains(reqUri, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		} else if strings.Contains(reqUri, ".json") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}

		go getData(reqUri, c)
		go rewrite(<-c, c)
		res = <-c

	} else {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		go getData(reqUri, c)
		res = execute(<-c, c)
		isContain := isNecessary(res)

		if isContain.include || isContain.includeId {
			res = execute(res, c)
			isContain = isNecessary(res)
		}
		if isContain.include || isContain.includeId {
			res = execute(res, c)
		}
	}

	if res.Error != nil {
		fmt.Printf("error: %v\n", res.Error)
		http.Error(w, "file not found", 404)
	}

	fmt.Fprint(w, res.Response)
}

func execute(data Res, c chan Res) Res {
	go rewrite(data, c)
	go includeIdReplace(<-c, c)
	go includeReplace(<-c, c)
	go rewrite(<-c, c)

	return <-c
}

func rewrite(txt Res, c chan Res) {
	path := config.Path
	str := txt.Response
	resErr := txt.Error
	res := [][][][]byte{}
	buf := []byte(str)

	if len(path) > 0 && resErr == nil {
		for _, v := range path {
			regPath := regexp.MustCompile(v.K)
			res = append(res, regPath.FindAllSubmatch(buf, 1))
		}

		if len(res) > 0 {
			for _, v := range path {
				str = strings.Replace(str, v.K, v.V, -1)
			}
		}
	}

	c <- Res{Error: resErr, Response: str}
}

func includeReplace(txt Res, c chan Res) {
	regInc := regexp.MustCompile(`<!--#include ([a-z]+)="(\S+)" -->`)
	str := txt.Response
	resErr := txt.Error
	resBuf := []byte(str)
	res := regInc.FindAllSubmatch(resBuf, -1)

	if len(res) > 0 && resErr == nil {
		for _, v := range res {
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

func includeIdReplace(txt Res, c chan Res) {
	includeId := config.IncludeId
	str := txt.Response
	resBuf := []byte(str)
	resErr := txt.Error
	res := [][][][]byte{}

	if resErr == nil {
		for _, v := range includeId {
			regString := fmt.Sprintf(`<("[^"]*"|'[^']*'|[^'">])*id="%s"("[^"]*"|'[^']*'|[^'">])*></("[^"]*"|'[^']*'|[^'">])*>`, v.K)
			regInc := regexp.MustCompile(regString)
			res = append(res, regInc.FindAllSubmatch(resBuf, 1))
		}

		if len(includeId) > 0 && len(res) > 0 {
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

func getData(reqURI string, c chan Res) {
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

	c <- Res{Error: err, Response: txt}
}

func isNecessary(txt Res) Necessary {
	var necessary Necessary
	necessary.include = false
	necessary.includeId = false

	resErr := txt.Error
	resStr := txt.Response
	resBuf := []byte(resStr)
	regInc := regexp.MustCompile(`<!--#include ([a-z]+)="(\S+)" -->`)
	res := regInc.FindAllSubmatch(resBuf, -1)

	includeId := config.IncludeId
	resId := [][][][]byte{}

	for _, v := range includeId {
		regString := fmt.Sprintf(`<("[^"]*"|'[^']*'|[^'">])*id="%s"("[^"]*"|'[^']*'|[^'">])*></("[^"]*"|'[^']*'|[^'">])*>`, v.K)
		regIncId := regexp.MustCompile(regString)
		resId = append(resId, regIncId.FindAllSubmatch(resBuf, 1))
	}

	if resErr == nil {
		if len(res) > 0 {
			necessary.include = true
		} else if len(resId) > 0 {
			necessary.includeId = true
		}
	}

	return necessary
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
