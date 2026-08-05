package providers

providers: "yay": #ArchAURHelper & {
	#aur: "yay"
	elevated: false
	commands: {
 "install": "-S --noconfirm --needed",
 "update": "-Syu --noconfirm",
 "remove": "-Rns --noconfirm",
 "list": "-Qm",
 "listExplicit": "-Qme",
 "search": "-Ss",
 "clean": "-Yc --noconfirm"
}
}
