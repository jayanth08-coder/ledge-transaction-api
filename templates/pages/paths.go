package pages

import "github.com/a-h/templ"

func accountPath(id string) templ.SafeURL {
	return templ.URL("/accounts/" + id)
}
