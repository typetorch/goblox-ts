package luau

// SetParent is `node.parent = parent` for transformer code that relinks list nodes by hand (roblox-ts's
// transformForStatement addFinalizers).
func SetParent(node Node, parent Node) { node.setParent(parent) }
