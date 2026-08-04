package providers

providers: "pamac": #ArchAURHelper & {
	#aur: "pamac-aur"
	elevated: true
	commands: {
 "install": "build --no-confirm",
 "update": "upgrade -a --no-confirm",
 "remove": "remove --no-confirm",
 "list": "list -i",
 "listExplicit": "list --explicitly-installed",
 "search": "search -a",
 "clean": "clean --no-confirm"
}
}
