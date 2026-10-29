declare function tag(strings: ReadonlyArray<string>, ...values: Array<unknown>): string;
declare function f(): string;
declare let n: number;
declare let s: string;
declare const obj: { tag: typeof tag };

export const sub1 = `a ${n} b`;
export const sub2 = `${n}${s}`;
export const sub3 = `x${f()}y${(n = 5)}z`;
export const subEsc = `{${n}} \u0041${s}\t`;
export const subTable = `${[1, 2]}`;
