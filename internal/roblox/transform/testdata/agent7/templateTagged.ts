declare function tag(strings: ReadonlyArray<string>, ...values: Array<unknown>): string;
declare function f(): string;
declare let n: number;
declare let s: string;
declare const obj: { tag: typeof tag };

export const tagged0 = tag`plain`;
export const tagged1 = tag`a ${n} b ${s} c`;
export const tagged2 = tag`${n}`;
export const tagged3 = obj.tag`m ${f()} n ${(n = 3)}`;
export const taggedEsc = tag`a\tb ${n} \u0041`;
