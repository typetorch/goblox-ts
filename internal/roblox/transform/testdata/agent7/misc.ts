declare function f(): number;
declare let n: number;
declare let maybe: number | undefined;
declare const p: Promise<number>;
declare const it: Generator<number, string, number>;

export const c1 = n > 0 ? "pos" : "neg";
export const c2 = n > 0 ? (n = 1) : f();
n > 0 ? f() : (n = 2);
export const t1 = maybe! + 1;
export const t2 = (n as number) satisfies number;
export const t3 = <number>n;

export async function waiter() {
	return await p;
}

export function* gen() {
	yield;
	yield 1;
	const r = yield* it;
	yield* it;
	return r;
}
