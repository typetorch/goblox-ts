export class Greeter {
	private count = 0;

	constructor(private readonly name: string) {}

	greet(who: string) {
		this.count++;
		return `${this.name} greets ${who} (${this.count})`;
	}
}

export const doubled = [1, 2, 3].map((n) => n * 2);
