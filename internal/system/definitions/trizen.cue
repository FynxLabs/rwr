package providers

providers: "trizen": #ArchAURHelper & {
	#aur: "trizen"
	elevated: true
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
