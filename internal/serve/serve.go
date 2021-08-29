package serve

import (
	"fmt"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"

	"github.com/YouAreNotDefined/localchecker/internal/get"
	"github.com/YouAreNotDefined/localchecker/internal/optimize"
	"github.com/YouAreNotDefined/localchecker/internal/structure"
	"github.com/spf13/cobra"
)

func Listen(cmd *cobra.Command, args []string) {
	port := config.Port
	http.HandleFunc("/", Handler)
	log.Printf("Server listening on http://localhost:%s/", port)
	log.Print(http.ListenAndServe(":"+port, nil))
}

func Handler(w http.ResponseWriter, r *http.Request) {
	reqUri := r.RequestURI
	mime := MimeTypeForFile(reqUri)
	mimeType := fmt.Sprintf(`%s; charset=utf-8`, mime.contentType)
	c := make(chan structure.Res)
	var res structure.Res

	switch mime.category {
	case "html":
		go get.File(reqUri, c)
		res = <-c
	case "asset":
		go get.File(reqUri, c)
		go optimize.Rewrite(<-c, c)
		res = <-c
	case "other":
		go get.File(reqUri, c)
		res = ExecuteRoutine(<-c, c)
		isContain := IsNecessary(res)
		if isContain.include || isContain.includeId {
			res = ExecuteRoutine(res, c)
			isContain = IsNecessary(res)
		}
		if isContain.include || isContain.includeId {
			res = ExecuteRoutine(res, c)
		}
	}

	if res.Error != nil {
		fmt.Printf("error: %v\n", res.Error)
		http.Error(w, "file not found", 404)
	}

	w.Header().Set("Content-Type", mimeType)
	fmt.Fprint(w, res.Response)
}

func ExecuteRoutine(data structure.Res, c chan structure.Res) structure.Res {
	go optimize.Rewrite(data, c)
	go optimize.IncludeIdReplace(<-c, c)
	go optimize.IncludeReplace(<-c, c)
	go optimize.Rewrite(<-c, c)
	return <-c
}

func MimeTypeForFile(file string) structure.Mime {
	ext := filepath.Ext(file)
	switch ext {
	case ".htm", ".html":
		return structure.Mime{contentType: "text/html", category: "html"}
	case ".css":
		return structure.Mime{contentType: "text/css", category: "asset"}
	case ".js":
		return structure.Mime{contentType: "application/javascript", category: "asset"}
	case ".json":
		return structure.Mime{contentType: "application/json", category: "asset"}
	default:
		return structure.Mime{contentType: mime.TypeByExtension(ext), category: "other"}
	}
}

func IsNecessary(txt structure.Res) structure.Necessary {
	isNecessaryInc := false
	isNecessaryIncId := false
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
			isNecessaryInc = true
		} else if len(resId) > 0 {
			isNecessaryIncId = true
		}
	}

	return structure.Necessary{include: isNecessaryInc, includeId: isNecessaryIncId}
}
