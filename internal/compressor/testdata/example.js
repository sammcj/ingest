// This is a JavaScript comment
import { something } from 'module';

export class MyJSClass {
    constructor(name) {
        this.name = name;
    }

    greet(message) {
        // Method body
        console.log(message + ", " + this.name + "!");
        if (this.name.length > 3) {
            console.log("Long name in JS");
        }
        return true;
    }
}

export function myJSFunction(x, y) {
    // Function body
    const result = x + y;
    console.log("JS Result is " + result);
    return result;
}

// Arrow function with block body
const myArrowFunc = (a, b) => {
    // Arrow function body
    return a * b;
};

// Arrow function with expression body
const myExpressionArrow = (x) => x * x;

function* myGenerator() {
    yield 1;
    yield 2;
}

// Another export type
export const myVar = 42;

// Anonymous default export function
// Commented out for testing separately
// export default function() {
//     console.log("Anon default func body");
//     return "done";
// }

// Anonymous default export class
export default class {
    constructor() {
        this.x = 1;
    }
}

// Create a second test file for the anonymous function default export
