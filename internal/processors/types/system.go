package types

// ToolInfo represents information about a tool.
type ToolInfo struct {
	Exists bool   // Whether the tool exists
	Bin    string // Path to the tool binary
}

type ToolList struct {
	Git    ToolInfo
	Pip    ToolInfo
	Gem    ToolInfo
	Npm    ToolInfo
	Yarn   ToolInfo
	Pnpm   ToolInfo
	Bun    ToolInfo
	Cargo  ToolInfo
	Docker ToolInfo
	Curl   ToolInfo
	Wget   ToolInfo
	Make   ToolInfo
	GCC    ToolInfo
	Clang  ToolInfo
	Python ToolInfo
	Ruby   ToolInfo
	Java   ToolInfo
}
