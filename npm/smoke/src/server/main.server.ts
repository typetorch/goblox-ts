import { Greeter, doubled } from "../shared/greet";

const greeter = new Greeter("goblox");
for (const n of doubled) {
	print(greeter.greet(`player ${n}`));
}
game.GetService("Players").PlayerAdded.Connect((player) => print(greeter.greet(player.Name)));
