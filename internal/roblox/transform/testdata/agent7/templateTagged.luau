-- goblox, a roblox-ts fork. goblox v0.0.0-dev, roblox-ts v3.0.0
local tagged0 = tag({ "plain" })
local tagged1 = tag({ "a ", " b ", " c" }, n, s)
local tagged2 = tag({ "", "" }, n)
local _exp = f()
n = 3
local tagged3 = obj.tag({ "m ", " n ", "" }, _exp, n)
local taggedEsc = tag({ "a	b ", " A" }, n)
return {
	tagged0 = tagged0,
	tagged1 = tagged1,
	tagged2 = tagged2,
	tagged3 = tagged3,
	taggedEsc = taggedEsc,
}
