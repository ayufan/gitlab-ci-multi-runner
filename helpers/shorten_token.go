package helpers

func ShortenToken(token string) string {
	if len(token) >= 8 {
		return token[0:8]
	}
	return token
}
