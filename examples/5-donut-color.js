// A spinning, iced and sprinkled donut in 24-bit colour.
// Run: bin/jsrust examples/5-donut-color.js   (needs a true-colour terminal)
//
// The donut is a torus:
// a circle of radius R1 (the tube) swept around an axis at distance R2.
// Its surface is covered by a fixed grid of sample points, and each frame
//  1. turns every sample by the current angles, projects it onto the
//     screen and keeps the nearest one per pixel (a z-buffer);
//  2. patches the odd pixel the grid missed;
//  3. shades each visible sample: a diffuse term for the light falling
//     on the surface plus a specular highlight, applied to the colour of
//     the material (dough, icing or a sprinkle) at that point;
//  4. writes the pixels out two per character cell, using the upper half
//     block U+2580 (foreground = top pixel, background = bottom pixel),
//     which gives roughly square pixels.
// Nothing is drawn behind the donut, so the terminal's background
// (and its transparency, if it has any) shows through.
//
// If you stop it with Ctrl-C the cursor stays hidden;
// `tput cnorm` (or `reset`) brings it back.

var COLS = 80;               // character cells across
var ROWS = 24;               // character cells down (2 pixels each)
var SECONDS = 20;
var FPS = 30;

var R1 = 1;                  // tube radius
var R2 = 2;                  // distance from the axis to the tube centre
var K2 = 5;                  // distance from the viewer to the torus

// Sample points on the surface are about SPACING apart,
// which keeps them about a pixel apart where the torus is nearest the viewer;
// the odd pixel they miss is patched up afterwards (pass 2).
var SPACING = 0.08;
var THETA_STEP = SPACING / R1;
var PHI_STEP = 0.01;         // finest phi step; most rings use a multiple
var SPRINKLES = 90;

var ESC = "\x1b[";
var W = COLS, H = ROWS * 2;  // in pixels
// Pixels are square, so scale to the shorter side:
// the torus then stays on screen whichever way it is turned.
var K1 = (W < H ? W : H) * K2 * 3 / (8 * (R1 + R2));

// --- Random numbers ---------------------------------------------------

// A small seeded generator (Park-Miller),
// so the sprinkles land in the same places every run.
// All products stay below 2^53, so they're exact.
var seed = 20261122;
function random() {
    seed = (seed * 16807) % 2147483647;
    return seed / 2147483647;
}

// --- The surface ------------------------------------------------------

// Sample (i, j) sits at angle theta = i * THETA_STEP around the tube
// and phi = j * PHI_STEP around the axis.
// Data for every sample is kept in flat arrays at index i * nPhi + j.
var nTheta = Math.ceil(2 * Math.PI / THETA_STEP);
var nPhi = Math.ceil(2 * Math.PI / PHI_STEP);
var cosT = new Array(nTheta), sinT = new Array(nTheta);
var cosP = new Array(nPhi), sinP = new Array(nPhi);
for (var i = 0; i < nTheta; i++) { cosT[i] = Math.cos(i * THETA_STEP); sinT[i] = Math.sin(i * THETA_STEP); }
for (var j = 0; j < nPhi; j++)   { cosP[j] = Math.cos(j * PHI_STEP);   sinP[j] = Math.sin(j * PHI_STEP); }

// Each theta gives a ring around the axis of radius R2 + R1*cos(theta).
// The rings on the inside of the hole are shorter, so they need fewer samples:
// ring i only uses every phiStride[i]-th phi.
var phiStride = new Array(nTheta);
for (var i = 0; i < nTheta; i++) {
    var stride = Math.floor(SPACING / ((R2 + R1 * cosT[i]) * PHI_STEP));
    phiStride[i] = stride < 1 ? 1 : stride;
}

// --- Materials --------------------------------------------------------

// Base colour and how glossy it is (0 = matte).
// Material 0 is dough, 1 is icing and the rest are the sprinkle colours.
var DOUGH = 0, ICING = 1, FIRST_SPRINKLE = 2;
var MAT_R    = new Array(205, 240, 250, 255,  60, 120, 255);
var MAT_G    = new Array(140,  90, 250, 210, 170, 220, 140);
var MAT_B    = new Array( 70, 160, 250,  40, 255, 120,  30);
var MAT_GLOSS = new Array(0.15, 0.8, 0.5, 0.5, 0.5, 0.5, 0.5);
var NMAT = MAT_R.length;

// The material of every sample:
// the tube's top half (sin theta > 0, before rotation) is iced and the rest is dough.
// The edge of the icing wobbles with phi, so it looks like it has dripped.
function iceDonut() {
    var material = new Array(nTheta * nPhi);
    for (var j = 0; j < nPhi; j++) {
        var phi = j * PHI_STEP;
        var edge = -0.1 + 0.18 * Math.sin(3 * phi + 1) + 0.12 * Math.sin(8 * phi);
        for (var i = 0; i < nTheta; i++)
            material[i * nPhi + j] = sinT[i] > edge ? ICING : DOUGH;
    }
    return material;
}

var ROD_LEN = 0.2, ROD_WIDTH = 0.08;   // half length and half width

// Paints `count` sprinkles onto the icing in `material`:
// short rods, each with a random direction and colour, kept away from the icing's edge.
// Distances are measured on the surface:
// one step in theta is R1*THETA_STEP long, one step in phi (R2 + R1 cos theta)*PHI_STEP.
function addSprinkles(material, count) {
    var placed = 0;
    while (placed < count) {
        var ci = Math.floor(random() * nTheta), cj = Math.floor(random() * nPhi);
        if (material[ci * nPhi + cj] != ICING || sinT[ci] < 0.15) continue;
        var m = FIRST_SPRINKLE + Math.floor(random() * (NMAT - FIRST_SPRINKLE));
        var angle = random() * Math.PI;
        var dirU = Math.cos(angle), dirV = Math.sin(angle);
        var ring = R2 + R1 * cosT[ci];     // radius of the circle phi sweeps
        var di = Math.ceil(ROD_LEN / (R1 * THETA_STEP));
        var dj = Math.ceil(ROD_LEN / (ring * PHI_STEP));
        for (var a = -di; a <= di; a++) {
            for (var b = -dj; b <= dj; b++) {
                var u = a * R1 * THETA_STEP, v = b * ring * PHI_STEP;
                var along = u * dirU + v * dirV, across = v * dirU - u * dirV;
                if (along * along <= ROD_LEN * ROD_LEN && across * across <= ROD_WIDTH * ROD_WIDTH) {
                    var i = (ci + a + nTheta) % nTheta, j = (cj + b + nPhi) % nPhi;
                    material[i * nPhi + j] = m;
                }
            }
        }
        placed++;
    }
}

var material = iceDonut();
addSprinkles(material, SPRINKLES);

// --- Colours ----------------------------------------------------------

// A shaded pixel is a material lit to one of DIFFUSE_LEVELS diffuse levels
// plus one of SPEC_LEVELS specular levels.
// That is few enough combinations to build every escape code up front,
// so drawing a pixel is a lookup at codeIndex(m, d, s).
var DIFFUSE_LEVELS = 32, SPEC_LEVELS = 16;
var AMBIENT = 0.12;

function codeIndex(m, d, s) {
    return (m * DIFFUSE_LEVELS + d) * SPEC_LEVELS + s;
}

function clamp255(v) { return v > 255 ? 255 : Math.floor(v); }

var fgCode = new Array(NMAT * DIFFUSE_LEVELS * SPEC_LEVELS);
var bgCode = new Array(NMAT * DIFFUSE_LEVELS * SPEC_LEVELS);
function buildColourCodes() {
    for (var m = 0; m < NMAT; m++) {
        for (var d = 0; d < DIFFUSE_LEVELS; d++) {
            var light = AMBIENT + (1 - AMBIENT) * d / (DIFFUSE_LEVELS - 1);
            for (var s = 0; s < SPEC_LEVELS; s++) {
                var shine = 255 * MAT_GLOSS[m] * s / (SPEC_LEVELS - 1);
                var rgb = clamp255(MAT_R[m] * light + shine) + ";" +
                          clamp255(MAT_G[m] * light + shine) + ";" +
                          clamp255(MAT_B[m] * light + shine) + "m";
                fgCode[codeIndex(m, d, s)] = ESC + "38;2;" + rgb;
                bgCode[codeIndex(m, d, s)] = ESC + "48;2;" + rgb;
            }
        }
    }
}
buildColourCodes();

// The terminal's own background, used wherever the donut isn't.
var DEFAULT_BG = ESC + "49m";

// --- One frame --------------------------------------------------------

// Per-pixel buffers, indexed x + W * y, passed from one pass to the next.
var SIZE = W * H;
var zbuf = new Array(SIZE);      // 1/z of the nearest sample, 0 for none
var hitI = new Array(SIZE);      // which sample (i, j) is nearest,
var hitJ = new Array(SIZE);      // or hitI = -1 for none
var pixel = new Array(SIZE);     // codeIndex of the pixel, or -1 for none

// The donut turned by angle A about the x axis and B about the z axis,
// as a string that draws it from the top-left corner of the terminal.
function renderFrame(A, B) {
    var cA = Math.cos(A), sA = Math.sin(A);
    var cB = Math.cos(B), sB = Math.sin(B);
    projectSamples(cA, sA, cB, sB);
    patchGaps();
    shadePixels(cA, sA, cB, sB);
    return pixelsToText();
}

// Pass 1:
// fill zbuf, hitI and hitJ with the nearest sample per pixel.
// The interpreter reads locals faster than globals,
// so the hot loops in this and the next passes work on local copies.
function projectSamples(cA, sA, cB, sB) {
    var w = W, h = H, k1 = K1, zb = zbuf, hI = hitI, hJ = hitJ;
    var cosp = cosP, sinp = sinP, np = nPhi;
    var halfW = W / 2, halfH = H / 2;

    for (var p = 0; p < SIZE; p++) { zb[p] = 0; hI[p] = -1; }

    for (var i = 0; i < nTheta; i++) {
        var ct = cosT[i], st = sinT[i];
        var circleX = R2 + R1 * ct;
        var circleY = R1 * st;

        // The sample is the tube's circle point (circleX, circleY, 0) turned about the y axis by phi,
        // then about x by A and about z by B.
        // Multiplied out, each coordinate is c1*cos(phi) + c2*sin(phi) + c3,
        // where only the c's depend on theta.
        var x1 = circleX * cB, x2 = circleX * sA * sB, x3 = -circleY * cA * sB;
        var y1 = circleX * sB, y2 = -circleX * sA * cB, y3 = circleY * cA * cB;
        var z2 = cA * circleX, z3 = K2 + circleY * sA;

        // Back-face culling.
        // A point faces the viewer when its normal N points back along the line of sight P, i.e. N.P < 0.
        // Rotations keep dot products,
        // so N.P = (R2*ct + R1) + K2 * nz,
        // where nz (the normal's z) is ct*cA*sp + st*sA.
        var f1 = R2 * ct + R1 + K2 * st * sA, f2 = K2 * ct * cA;

        var stride = phiStride[i];
        for (var j = 0; j < np; j += stride) {
            var sp = sinp[j];
            if (f1 + f2 * sp >= 0) continue;         // facing away
            var cp = cosp[j];
            var ooz = 1 / (z2 * sp + z3);            // "one over z"
            var kz = k1 * ooz;
            var xs = halfW + kz * (x1 * cp + x2 * sp + x3);
            var ys = halfH - kz * (y1 * cp + y2 * sp + y3);
            if (xs < 0 || xs >= w || ys < 0 || ys >= h) continue;
            // Both are positive here, so |0 rounds down like Math.floor.
            var idx = (xs | 0) + w * (ys | 0);
            if (ooz > zb[idx]) {                     // nearer than what's there
                zb[idx] = ooz;
                hI[idx] = i;
                hJ[idx] = j;
            }
        }
    }
}

// Pass 2:
// the samples are spaced so that they occasionally miss a pixel,
// leaving a speck of backdrop (or of the far side of the ring) showing through.
// Copy a neighbour into any pixel whose neighbours on both sides,
// left and right or above and below, are clearly nearer.
function patchGaps() {
    var w = W, h = H, zb = zbuf, hI = hitI, hJ = hitJ;
    for (var y = 1; y < h - 1; y++) {
        for (var x = 1; x < w - 1; x++) {
            var p = x + w * y, z = zb[p] * 1.08, q = -1;
            if (zb[p - 1] > z && zb[p + 1] > z) q = p - 1;
            else if (zb[p - w] > z && zb[p + w] > z) q = p - w;
            if (q >= 0) { zb[p] = zb[q]; hI[p] = hI[q]; hJ[p] = hJ[q]; }
        }
    }
}

// Pass 3:
// fill `pixel` by shading the sample that won each pixel.
// The surface normal is (cos theta, sin theta, 0) turned like the point,
// which gives
//   ny = ct * (sB*cp - sA*cB*sp) + st * cA*cB
//   nz = ct * cA*sp + st * sA.
// The light comes from up and towards the viewer, (0, 1, -1)/sqrt 2;
// the viewer looks along +z.
// The specular term uses the half-way vector between the two (Blinn-Phong),
// normalised (0, 0.383, -0.924).
function shadePixels(cA, sA, cB, sB) {
    var hI = hitI, hJ = hitJ, mat = material, np = nPhi;
    var cosp = cosP, sinp = sinP;
    var cosA_cosB = cA * cB, sinA_cosB = sA * cB;
    for (var p = 0; p < SIZE; p++) {
        var i = hI[p];
        if (i < 0) { pixel[p] = -1; continue; }
        var j = hJ[p];
        var ct = cosT[i], st = sinT[i], cp = cosp[j], sp = sinp[j];
        var ny = ct * (sB * cp - sinA_cosB * sp) + st * cosA_cosB;
        var nz = ct * cA * sp + st * sA;

        var diffuse = (ny - nz) * 0.7071;
        if (diffuse < 0) diffuse = 0;
        var spec = 0.3827 * ny - 0.9239 * nz;
        if (spec < 0) spec = 0;
        spec = spec * spec; spec = spec * spec; spec = spec * spec;
        spec = spec * spec; spec = spec * spec;          // ^32: a tight highlight

        var d = (diffuse * (DIFFUSE_LEVELS - 1) + 0.5) | 0;
        var s = (spec * (SPEC_LEVELS - 1) + 0.5) | 0;
        pixel[p] = codeIndex(mat[i * np + j], d, s);
    }
}

// Pass 4:
// turn `pixel` into text, two pixel rows per line.
// Only a character's foreground can be drawn without touching the background,
// so a cell with one backdrop pixel puts the donut pixel in the foreground:
// the upper half block if it is on top, the lower half block U+2584 if it is below.
// A cell that is all backdrop is a space.
// A colour code is only sent when it differs from the current one.
function pixelsToText() {
    var out = new Array();
    var n = 0;
    out[n++] = ESC + "H";                    // cursor to top-left
    for (var row = 0; row < ROWS; row++) {
        var top = 2 * row * W, bottom = top + W;
        // Current colours: an index into fgCode / bgCode,
        // or -1 for the terminal default, which is what each line starts with.
        var curFg = -1, curBg = -1;
        for (var x = 0; x < W; x++) {
            var upper = pixel[top + x], lower = pixel[bottom + x];
            var fg = upper, bg = lower, ch = "▀";
            if (upper < 0) {
                if (lower < 0) { fg = curFg; ch = " "; }   // any foreground will do
                else { fg = lower; bg = -1; ch = "▄"; }
            }
            if (fg != curFg) { out[n++] = fgCode[fg]; curFg = fg; }
            if (bg != curBg) { out[n++] = bg < 0 ? DEFAULT_BG : bgCode[bg]; curBg = bg; }
            out[n++] = ch;
        }
        out[n++] = ESC + "0m";
        if (row < ROWS - 1) out[n++] = "\n";
    }
    return out.join("");
}

// --- Main loop --------------------------------------------------------

function now() { return new Date().getTime(); }
function waitUntil(ms) { while (now() < ms) { } }   // ES1 has no sleep

print(ESC + "2J" + ESC + "?25l");       // clear screen, hide cursor

var start = now(), frames = 0, A = 1, B = 0;
var frameTime = 1000 / FPS;
while (now() - start < SECONDS * 1000) {
    var frameStart = now();
    print(renderFrame(A, B));
    frames++;
    // Advance by elapsed time,
    // so the spin speed doesn't depend on the frame rate the interpreter manages.
    waitUntil(frameStart + frameTime);
    var dt = (now() - frameStart) / 1000;
    A += 1.2 * dt;
    B += 0.6 * dt;
}

var elapsed = (now() - start) / 1000;
print(ESC + "0m" + ESC + "?25h");       // reset colours, show cursor
print(frames + " frames in " + Math.round(elapsed * 10) / 10 + "s (" +
      Math.round(frames / elapsed) + " fps)");
