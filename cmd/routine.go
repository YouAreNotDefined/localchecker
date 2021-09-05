package cmd

import (
	"fmt"
	"regexp"
)

type Routine struct {
	Response  *Res
	Include   bool
	IncludeId bool
}

func (r *Routine) run(c chan *Res) {
	go r.Response.rewrite(c)
	r.Response = <-c

	go r.Response.includeReplace(c)
	r.Response = <-c

	go r.Response.includeIdReplace(c)
	r.Response = <-c

	r.isNecessary()

	c <- r.Response
}

func (r *Routine) isNecessary() *Routine {
	resStr := r.Response.Response
	resBuf := []byte(resStr)
	includeId := config.IncludeId
	idMatched := [][][][]byte{}

	if len(includeId) > 0 {
		for _, v := range includeId {
			regString := fmt.Sprintf(`<("[^"]*"|'[^']*'|[^'">])*id="%s"("[^"]*"|'[^']*'|[^'">])*></("[^"]*"|'[^']*'|[^'">])*>`, v.K)
			regIncId := regexp.MustCompile(regString)
			idMatched = append(idMatched, regIncId.FindAllSubmatch(resBuf, 1))
		}
	}

	regInc := regexp.MustCompile(`<!--#include ([a-z]+)="(\S+)" -->`)
	incMatched := regInc.FindAllSubmatch(resBuf, -1)

	if len(incMatched) == 0 && len(idMatched) == 0 {
		r.Include = false
		r.IncludeId = false
	} else if len(incMatched) == 0 {
		r.Include = false
		r.IncludeId = true
	} else if len(idMatched) == 0 {
		r.IncludeId = false
		r.Include = true
	} else {
		r.Include = true
		r.IncludeId = true
	}

	return &Routine{Response: r.Response, Include: r.Include, IncludeId: r.IncludeId}
}
