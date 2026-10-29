declare function tag(strings: ReadonlyArray<string>, ...values: Array<unknown>): string;
declare function f(): string;
declare let n: number;
declare let s: string;
declare const obj: { tag: typeof tag };

export const plain = `hello world`;
export const empty = ``;
export const escapes = `tab\there \`tick\` \u0041 \uD83D\uDE00 \$ {braces} \\u0041 line\
continued`;
export const multi = `line1
line2`;
