package url_helpers

import "net/url"

func CleanURL(value string) (ret string) {
	u, err := url.Parse(value)
	if err != nil {
		return
	}
	u.User = nil
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}
