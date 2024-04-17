package types

// ToolInfo represents information about a tool.
type ToolInfo struct {
	Exists bool   // Whether the tool exists
	Bin    string // Path to the tool binary
}

type ToolList struct {
	Git        ToolInfo
	Bun        ToolInfo
	Docker     ToolInfo
	Curl       ToolInfo
	Wget       ToolInfo
	Make       ToolInfo
	GCC        ToolInfo
	Clang      ToolInfo
	Python     ToolInfo
	Ruby       ToolInfo
	Java       ToolInfo
	Bash       ToolInfo
	Zsh        ToolInfo
	PowerShell ToolInfo
	Perl       ToolInfo
	Lua        ToolInfo
	Go         ToolInfo
	Rust       ToolInfo
	Gpg        ToolInfo
	Rpm        ToolInfo
	Dpkg       ToolInfo
	Cat        ToolInfo
	Ls         ToolInfo
	Lsof       ToolInfo
}
