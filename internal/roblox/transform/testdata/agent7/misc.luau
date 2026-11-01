-- goblox, a roblox-ts fork. goblox v0.0.0-dev, roblox-ts v3.0.0
local TS = require(game:GetService("ReplicatedStorage"):WaitForChild("include"):WaitForChild("RuntimeLib"))
local c1 = if n > 0 then "pos" else "neg"
local _result
if n > 0 then
	n = 1
	_result = n
else
	_result = f()
end
local c2 = _result
if n > 0 then
	f()
else
	n = 2
	local _ = n
end
local t1 = maybe + 1
local t2 = n
local t3 = n
local waiter = TS.async(function()
	return TS.await(p)
end)
local function gen()
	return TS.generator(function()
		coroutine.yield()
		coroutine.yield(1)
		local _next = it.next
		local _currentInput
		local _returnValue
		while true do
			local _result_1 = _next(_currentInput)
			if _result_1.done then
				_returnValue = _result_1.value
				break
			end
			_currentInput = coroutine.yield(_result_1.value)
		end
		local r = _returnValue
		local _next_1 = it.next
		local _currentInput_1
		while true do
			local _result_1 = _next_1(_currentInput_1)
			if _result_1.done then
				break
			end
			_currentInput_1 = coroutine.yield(_result_1.value)
		end
		return r
	end)
end
return {
	waiter = waiter,
	gen = gen,
	c1 = c1,
	c2 = c2,
	t1 = t1,
	t2 = t2,
	t3 = t3,
}
