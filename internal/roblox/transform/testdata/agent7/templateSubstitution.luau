-- Compiled with roblox-ts v3.0.0
local sub1 = `a {n} b`
local sub2 = `{n}{s}`
local _exp = f()
n = 5
local sub3 = `x{_exp}y{n}z`
local subEsc = `\{{n}\} \u{0041}{s}\t`
local subTable = `{({ 1, 2 })}`
return {
	sub1 = sub1,
	sub2 = sub2,
	sub3 = sub3,
	subEsc = subEsc,
	subTable = subTable,
}
