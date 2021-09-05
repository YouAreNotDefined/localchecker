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
		go res.rewrite(<-c, c)
		go res.includeIdReplace(<-c, c)
		go res.includeReplace(<-c, c)
	}

	return <-c
}

func (r *Routine) isNecessary(txt Res) *Routine {
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
		if len(res) == 0 {
			r.Include = false
		} else if len(resId) == 0 {
			r.IncludeId = false
		}
	}

	return &Routine{Include: r.Include, IncludeId: r.IncludeId}
}
