// Factorial, recursively and iteratively.
// Run: bin/jsrust examples/1-factorial.js

// n! for a non-negative integer n.
function factorial(n) {
    return n <= 1 ? 1 : n * factorial(n - 1);
}

// The same, with a loop instead of recursion.
// Note the order of the products:
// the recursion finishes factorial(n - 1) before multiplying,
// so it works out ((1 * 2) * 3) * ... * n; this loop goes the other way,
// ((n * (n-1)) * (n-2)) * ... * 1.
function factorialLoop(n) {
    var result = 1;
    for (var k = n; k > 1; k--) result *= k;
    return result;
}

for (var n = 0; n <= 12; n++) print(n + "! =", factorial(n));

print();
print("25!  =", factorial(25));
print("170! =", factorial(170), "(recursive)");
print("170! =", factorialLoop(170), "(loop)");
print("171! =", factorial(171), "(too big for a double)");
