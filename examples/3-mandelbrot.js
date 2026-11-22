// The Mandelbrot set, drawn in ASCII.
// Run: bin/jscheme examples/3-mandelbrot.js
//
// For each character cell we pick a point c in the complex plane and iterate z = z*z + c from z = 0.
// Points whose z stays bounded belong to the set (drawn '@'); the others are shaded by how fast z escapes.

var WIDTH = 78;
var HEIGHT = 36;
var MAX_ITER = 60;

var X_MIN = -2.2, X_MAX = 0.8;
var Y_MIN = -1.2, Y_MAX = 1.2;

// From fast escape (left) to slow escape (right);
// the last character is used only for points that never escaped.
var PALETTE = " .,-:;=+*#%@";

// Number of iterations before |z| > 2, or MAX_ITER if it never escapes.
function escapeTime(cr, ci) {
    var zr = 0, zi = 0, zr2 = 0, zi2 = 0;
    var n = 0;
    while (n < MAX_ITER && zr2 + zi2 <= 4) {
        zi = 2 * zr * zi + ci;
        zr = zr2 - zi2 + cr;
        zr2 = zr * zr;
        zi2 = zi * zi;
        n++;
    }
    return n;
}

// The character for an escape time n in 0..MAX_ITER.
// Most points escape within a few iterations,
// so the square root spreads those over more characters.
// It reaches 1, and so the last character, only at MAX_ITER.
function shade(n) {
    var k = Math.floor(Math.sqrt(n / MAX_ITER) * (PALETTE.length - 1));
    return PALETTE.charAt(k);
}

for (var row = 0; row < HEIGHT; row++) {
    var ci = Y_MAX - (Y_MAX - Y_MIN) * row / (HEIGHT - 1);
    var line = "";
    for (var col = 0; col < WIDTH; col++) {
        var cr = X_MIN + (X_MAX - X_MIN) * col / (WIDTH - 1);
        line += shade(escapeTime(cr, ci));
    }
    print(line);
}
