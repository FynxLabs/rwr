package providers

providers: "pacman": #PacmanFamily & {
	elevated: true
	detection: binary: "pacman"
	commands: {
 "install": "-Sy --noconfirm",
 "update": "-Syu --noconfirm",
 "remove": "-R --noconfirm",
 "list": "-Q",
 "search": "-Ss",
 "clean": "-Sc --noconfirm"
}
}
