package providers

providers: "aura": #ArchAURHelper & {
	#aur: "aura-bin"
	#removePkg: "aura"
	elevated: true
	commands: {
 "install": "-A --noconfirm",
 "update": "-Au --noconfirm",
 "remove": "-R --noconfirm",
 "list": "-Qm",
 "listExplicit": "-Qme",
 "search": "-As",
 "clean": "-Cc --noconfirm"
}
}
