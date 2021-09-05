package cmd

import (
	"fmt"
	"regexp"
)

type Routine struct {
	Include   bool
	IncludeId bool
}

func (r *Routine) executeRoutine(res Res, c chan Res) Res {
	go res.rewrite(res, c)

	isContain := r.isNecessary(<-c)

	for isContain.Include || isContain.IncludeId {
		go res.rewrite(res, c)
		go res.includeIdReplace(<-c, c)
		go res.includeReplace(<-c, c)
	}

	return <-c
}

func (r *Routine) isNecessary(res Res) *Routine {
	resStr := res.Response
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

	if len(incMatched) == 0 {
		r.Include = false
	} else if len(idMatched) == 0 {
		r.IncludeId = false
	}

	return &Routine{Include: r.Include, IncludeId: r.IncludeId}
}
