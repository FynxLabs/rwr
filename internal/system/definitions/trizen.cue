package providers

providers: "trizen": #ArchAURHelper & {
	#aur: "trizen"
	elevated: false
	commands: {
 "install": "-S --noconfirm --noedit",
 "update": "-Syua --noconfirm --noedit",
 "remove": "-Rns --noconfirm",
 "list": "-Qm",
 "listExplicit": "-Qme",
 "search": "-Ss",
 "clean": "-Sc --noconfirm"
}
}
