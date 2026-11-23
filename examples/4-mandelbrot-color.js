// The Mandelbrot set in 24-bit colour, with smooth shading.
// Run: bin/jsrust examples/4-mandelbrot-color.js   (needs a true-colour terminal)
//
// Like 3-mandelbrot.js, but:
//  - each character cell holds two pixels, using the upper half block
//    U+2580 (foreground = top pixel, background = bottom pixel),
//    which gives roughly square pixels and twice the vertical resolution;
//  - the escape count is made continuous ("smooth colouring"),
//    so the colour bands blend into each other instead of forming steps.

var W = 100;             // width in pixels, which is also in characters
var MAX_ITER = 300;

// The region to draw. Try for example:
//   seahorse valley:  CX = -0.745, CY = 0.11,  SPAN = 0.06
//   elephant valley:  CX = 0.282,  CY = 0.01,  SPAN = 0.04
var CX = -0.65, CY = 0;  // centre
var SPAN = 3.2;          // width of the view in the complex plane

var H = Math.round(W * 0.6 / 2) * 2;          // even, for whole character rows
var PIXEL = SPAN / W;                         // complex units per pixel
var ESC = "\x1b[";

// --- Escape time ------------------------------------------------------

// z escapes once |z| > 2, but smooth colouring is only accurate when z is well past that,
// so we keep iterating until |z| > BAILOUT.
var BAILOUT = 256;
var LOG2 = Math.log(2);

// Smooth iteration count for c = cr + ci*i, or -1 inside the set.
function smoothEscape(cr, ci) {
    var zr = 0, zi = 0, zr2 = 0, zi2 = 0, n = 0;
    while (n < MAX_ITER && zr2 + zi2 <= BAILOUT * BAILOUT) {
        zi = 2 * zr * zi + ci;
        zr = zr2 - zi2 + cr;
        zr2 = zr * zr;
        zi2 = zi * zi;
        n++;
    }
    if (n >= MAX_ITER) return -1;
    // log2(log|z|) grows by 1 per iteration once z escapes;
    // subtracting it turns the integer count into a continuous one.
    var logZ = Math.log(zr2 + zi2) / 2;
    return n + 1 - Math.log(logZ / LOG2) / LOG2;
}

// --- Colour -----------------------------------------------------------

// Colour stops at positions 0..1,
// running deep blue -> white -> gold -> black and back to the starting blue,
// so the gradient wraps seamlessly.
// (ES1 has no array or object literals, hence one array per component.)
var STOP_POS = new Array(0,      0.16,  0.42,  0.6425, 0.8575, 1);
var STOP_R   = new Array(0,      32,    237,   255,    0,      0);
var STOP_G   = new Array(7,      107,   255,   170,    2,      7);
var STOP_B   = new Array(100,    203,   255,   0,      0,      100);

// The colour at t in [0, 1), blended linearly between the two nearest stops,
// as an "r;g;b" string ready for an escape sequence.
function gradient(t) {
    var i = 0;
    while (t > STOP_POS[i + 1]) i++;
    var f = (t - STOP_POS[i]) / (STOP_POS[i + 1] - STOP_POS[i]);
    var r = Math.floor(STOP_R[i] + f * (STOP_R[i + 1] - STOP_R[i]));
    var g = Math.floor(STOP_G[i] + f * (STOP_G[i + 1] - STOP_G[i]));
    var b = Math.floor(STOP_B[i] + f * (STOP_B[i + 1] - STOP_B[i]));
    return r + ";" + g + ";" + b;
}

var BANDS = 0.3;         // how quickly the colours cycle; try 0.1 to 0.5

// The colour for a smooth escape count mu (-1 inside the set).
function colour(mu) {
    if (mu < 0) return "0;0;0";                     // inside: black
    // The square root compresses the fast-escaping points far outside,
    // and the -1 starts them at deep blue.
    var t = (Math.sqrt(mu) - 1) * BANDS;
    if (t < 0) t = 0;
    return gradient(t - Math.floor(t));             // wrap into [0, 1)
}

// --- Drawing ----------------------------------------------------------

// The colour of pixel (x, y), with (0, 0) at the top left.
function pixelColour(x, y) {
    var cr = CX + (x - W / 2 + 0.5) * PIXEL;
    var ci = CY - (y - H / 2 + 0.5) * PIXEL;
    return colour(smoothEscape(cr, ci));
}

for (var row = 0; row < H / 2; row++) {
    var out = new Array();
    for (var x = 0; x < W; x++) {
        out[out.length] = ESC + "38;2;" + pixelColour(x, 2 * row) + "m" +
                          ESC + "48;2;" + pixelColour(x, 2 * row + 1) + "m▀";
    }
    out[out.length] = ESC + "0m";
    print(out.join(""));
}
