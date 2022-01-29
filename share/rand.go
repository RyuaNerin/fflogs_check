package share

import "math/rand"

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func FillRandomString(s []byte) {
	for i := range s {
		s[i] = charset[rand.Intn(len(charset))]
	}
}
