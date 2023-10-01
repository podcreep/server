package util

// GetVersionStr returns the version of the server as a string.
func GetVersionStr() string {
	// TODO: get this automatically somehow?
	return "0.0.1"
}

// GetUserAgent returns a string that can be used as the User-Agent string in requests we make.
func GetUserAgent() string {
	return "Podcreep/" + GetVersionStr()
}
