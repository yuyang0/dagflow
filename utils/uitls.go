package utils

func Invoke(fn func() error) error {
	return fn()
}
