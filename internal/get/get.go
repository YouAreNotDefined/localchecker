package get

import (
	"os"
	"strings"

	"github.com/YouAreNotDefined/localchecker/internal/structure"
)

func File(reqURI string, c chan structure.Res) {
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

	c <- structure.Res{Error: err, Response: txt}
}
