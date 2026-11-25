package accounts

import (
	"strconv"

	model "github.com/jterrazz/studio-station/apps/engine/accounts"
	"github.com/jterrazz/studio-station/apps/engine/catalog"
	"github.com/jterrazz/studio-station/apps/engine/page"
	"github.com/jterrazz/studio-station/apps/engine/pages"
)

func done(command, label string, said ...string) (string, []page.Row) {
	r := pages.Done(label, catalog.StateOK, "")
	r.Detail = page.Texts(said...)
	return command, []page.Row{r}
}

func outcome(thisMac page.Context, command string, rows []page.Row) page.Page {
	return pages.Outcome(command, thisMac, nil, rows, "passwords")
}

func Added(label, mac string, thisMac page.Context) page.Page {
	c, rows := done("accounts add", label, "kept in "+mac+"'s vault", "station accounts grant "+strconv.Quote(label)+" --on <machine> gives it to a VM")
	return outcome(thisMac, c, rows)
}

func Granted(label, vm string, thisMac page.Context) page.Page {
	c, rows := done("accounts grant", label, "granted to "+vm, "until it is removed")
	return outcome(thisMac, c, rows)
}

func Revoked(label, vm string, r model.Revoked, thisMac page.Context) page.Page {
	c, rows := done("accounts remove", label, "no longer granted to "+vm, r.Sentence)
	return outcome(thisMac, c, rows)
}

func Forgotten(label, mac string, ended int, thisMac page.Context) page.Page {
	grants := "no grant ended"
	switch {
	case ended == 1:
		grants = "1 grant ended"
	case ended > 1:
		grants = strconv.Itoa(ended) + " grants ended"
	}
	c, rows := done("accounts remove", label, "out of "+mac+"'s vault", grants)
	return outcome(thisMac, c, rows)
}
