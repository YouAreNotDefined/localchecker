package optimize

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/YouAreNotDefined/localchecker/internal/structure"
)

func Rewrite(txt structure.Res, c chan structure.Res) {
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

	c <- structure.Res{Error: resErr, Response: str}
}

func IncludeReplace(txt structure.Res, c chan structure.Res) {
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

	c <- structure.Res{Error: resErr, Response: str}
}

func IncludeIdReplace(txt structure.Res, c chan structure.Res) {
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

	c <- structure.Res{Error: resErr, Response: str}
}
