package providers

providers: "paru": #ArchAURHelper & {
	#aur: "paru"
	elevated: false
	commands: {
 "install": "-S --noconfirm",
 "update": "-Sua --noconfirm",
 "remove": "-Rns --noconfirm",
 "list": "-Qm",
 "listExplicit": "-Qme",
 "search": "-Ss",
 "clean": "-Scc --noconfirm"
}
}
