package providers

providers: "apt": #DebianFamily & {
	elevated: true
	detection: binary: "apt"
	commands: {
 "install": "install -y",
 "update": "update",
 "remove": "remove -y",
 "list": "dpkg --get-selections",
 "listExplicit": "apt-mark showmanual",
 "search": "search",
 "clean": "clean"
}
	corePackages: {
 "openssl": [
  "openssl",
  "libssl-dev"
 ],
 "build-essentials": [
  "build-essential",
  "cmake",
  "pkg-config",
  "libfreetype6-dev",
  "libfontconfig1-dev",
  "libxcb-xfixes0-dev",
  "libxkbcommon-dev",
  "python3"
 ]
}
}
