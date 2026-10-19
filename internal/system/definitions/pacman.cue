package providers

providers: "pacman": #PacmanFamily & {
	elevated: true
	detection: binary: "pacman"
	commands: {
		"install":      "-S --needed --noconfirm"
		"update":       "-Syu --noconfirm"
		"remove":       "-R --noconfirm"
		"list":         "-Q"
		"listExplicit": "-Qe"
		"search":       "-Ss"
		"clean":        "-Sc --noconfirm"
	}
}
