// Fibonacci numbers, computed three ways.
// Run: bin/jsrust examples/2-fibonacci.js
//
// fib(0) = 0, fib(1) = 1, fib(n) = fib(n - 1) + fib(n - 2).

// Straight from the definition.
// Each call makes two more, so the work grows like fib(n) itself:
// fine for n = 20, hopeless for n = 50.
function fibRecursive(n) {
    if (n < 2) return n;
    return fibRecursive(n - 1) + fibRecursive(n - 2);
}

// The same recursion, but each result is remembered the first time it is computed,
// so every fib(k) is worked out only once.
var fibCache = new Object();
function fibMemo(n) {
    if (n < 2) return n;
    if (fibCache[n] == null) fibCache[n] = fibMemo(n - 1) + fibMemo(n - 2);
    return fibCache[n];
}

// Walk up the sequence keeping only the last two numbers.
function fibIterative(n) {
    var a = 0, b = 1;               // fib(i) and fib(i + 1)
    for (var i = 0; i < n; i++) {
        var next = a + b;
        a = b;
        b = next;
    }
    return a;
}

var line = "";
for (var i = 0; i <= 20; i++) line += fibRecursive(i) + " ";
print("recursive : fib(0..20) =", line);
print("memoised  : fib(50)    =", fibMemo(50));
print("iterative : fib(78)    =", fibIterative(78));
