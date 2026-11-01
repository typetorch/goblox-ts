-- goblox, a roblox-ts fork. goblox v0.0.0-dev, roblox-ts v3.0.0
local plain = `hello world`
local empty = ``
local escapes = `tab\there \`tick\` \u{0041} \u{1f600} $ \{braces\} \\u0041 linecontinued`
local multi = `line1\
line2`
return {
	plain = plain,
	empty = empty,
	escapes = escapes,
	multi = multi,
}
