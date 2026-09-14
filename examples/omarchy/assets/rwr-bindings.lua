-- Load this user-owned module from your managed bindings.lua, after other
-- overrides that should lose to these four bindings.
local cycle = os.getenv("HOME") .. "/.config/hypr/workspace-cycle"
local function quote(value)
  return "'" .. value:gsub("'", "'\"'\"'") .. "'"
end
for _, key in ipairs({ "LEFT", "RIGHT", "UP", "DOWN" }) do
  hl.unbind("CTRL + ALT + " .. key)
end
o.bind("CTRL + ALT + LEFT", "Previous workspace", quote(cycle) .. " previous")
o.bind("CTRL + ALT + RIGHT", "Next workspace", quote(cycle) .. " next")
o.bind("CTRL + ALT + UP", "Exposé", hl.dsp.event("expose.window-overview:toggle"))
o.bind("CTRL + ALT + DOWN", "Workspace switcher", "omarchy-shell shell toggle io.github.woogy7.workspaces")
