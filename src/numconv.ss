;;; numconv.ss -- conversions between numbers and strings.

;;; Number -> String (9.8.1)

;; Shortest round-tripping decimal digits of a positive finite flonum:
;; returns (values digit-string n) where the value is 0.d1d2... * 10^n.
(define (flonum-digits x)
  (let* ([s (number->string x)]
         [s (let ([bar (string-index-of s #\|)]) (if bar (substring s 0 bar) s))]
         [epos (string-index-of s #\e)]
         [mant (if epos (substring s 0 epos) s)]
         [ex (if epos (string->number (substring s (fx+ epos 1) (string-length s))) 0)]
         [dot (string-index-of mant #\.)]
         [ipart (if dot (substring mant 0 dot) mant)]
         [fpart (if dot (substring mant (fx+ dot 1) (string-length mant)) "")]
         [digits (string-append ipart fpart)]
         [n (+ (string-length ipart) ex)])
    ;; strip leading zeros
    (let lead ([i 0] [n n])
      (if (and (fx< i (string-length digits)) (char=? (string-ref digits i) #\0))
          (lead (fx+ i 1) (- n 1))
          (let trail ([j (string-length digits)])
            (if (and (fx> j i) (char=? (string-ref digits (fx- j 1)) #\0))
                (trail (fx- j 1))
                (values (substring digits i j) n)))))))

(define (string-index-of s c)
  (let ([n (string-length s)])
    (let loop ([i 0])
      (cond [(fx= i n) #f] [(char=? (string-ref s i) c) i] [else (loop (fx+ i 1))]))))

(define (zeros k) (make-string (max k 0) #\0))

(define (js-number->string x)
  (cond
    [(flnan? x) "NaN"]
    [(fl= x 0.0) "0"]
    [(flinfinite? x) (if (fl> x 0.0) "Infinity" "-Infinity")]
    [(fl< x 0.0) (string-append "-" (js-number->string (fl- x)))]
    [(and (fl< x 9007199254740992.0) (fl= x (flfloor x)))
     (number->string (flonum->fixnum x))]
    [else
     (let-values ([(s n) (flonum-digits x)])
       (let ([k (string-length s)])
         (cond
           [(and (<= k n) (<= n 21)) (string-append s (zeros (- n k)))]
           [(and (< 0 n) (<= n 21))
            (string-append (substring s 0 n) "." (substring s n k))]
           [(and (< -6 n) (<= n 0))
            (string-append "0." (zeros (- n)) s)]
           [else
            (let* ([e (- n 1)]
                   [es (string-append (if (< e 0) "-" "+") (number->string (abs e)))])
              (if (= k 1)
                  (string-append s "e" es)
                  (string-append (substring s 0 1) "." (substring s 1 k) "e" es)))])))]))

;; Number.prototype.toString(radix) for radix other than 10.
(define (number->radix-string x radix)
  (cond
    [(flnan? x) "NaN"]
    [(flinfinite? x) (if (fl> x 0.0) "Infinity" "-Infinity")]
    [(fl= x 0.0) "0"]
    [(fl< x 0.0) (string-append "-" (number->radix-string (fl- x) radix))]
    [else
     (let* ([ip (flfloor x)]
            [is (string-downcase (number->string (exact ip) radix))]
            [frac (fl- x ip)])
       (if (fl= frac 0.0)
           is
           (let-values ([(p get) (open-string-output-port)])
             (put-string p is)
             (put-char p #\.)
             (let loop ([f frac] [k 0])
               (when (and (fl> f 0.0) (fx< k 52))
                 (let* ([v (fl* f (fixnum->flonum radix))]
                        [d (flonum->fixnum (flfloor v))])
                   (put-char p (string-ref "0123456789abcdefghijklmnopqrstuvwxyz" d))
                   (loop (fl- v (fixnum->flonum d)) (fx+ k 1)))))
             (get))))]))

;;; Decimal digits -> Number

;; The number value for the decimal integer `digits` (a string of decimal
;; digits, possibly empty) times 10^exp10, correctly rounded.
(define (decimal->flonum digits exp10)
  (let* ([n (string-length digits)]
         [start (let loop ([i 0]) (if (and (fx< i n) (char=? (string-ref digits i) #\0)) (loop (fx+ i 1)) i))])
    (if (fx= start n)
        0.0
        (let* ([sig (substring digits start n)]
               [point (+ (string-length sig) exp10)])
          (cond
            [(> point 400) +inf.0]
            [(< point -400) 0.0]
            [else
             (let ([m (string->number sig 10)])
               (inexact (if (>= exp10 0) (* m (expt 10 exp10)) (/ m (expt 10 (- exp10))))))])))))

(define (digit-char? c) (and (char? c) (char<=? #\0 c #\9)))

(define (hex-digit-value c)
  (cond
    [(not (char? c)) #f]
    [(char<=? #\0 c #\9) (fx- (char->integer c) 48)]
    [(char<=? #\a c #\f) (fx- (char->integer c) 87)]
    [(char<=? #\A c #\F) (fx- (char->integer c) 55)]
    [else #f]))

;; Scan the longest StrDecimalLiteral (unsigned, 9.3.1) in s starting at i.
;; Returns (values end value), or (values #f #f) when there is none.
(define (scan-str-decimal s i)
  (let ([n (string-length s)])
    (define (at j) (and (fx< j n) (string-ref s j)))
    (define (digits-end j) (if (digit-char? (at j)) (digits-end (fx+ j 1)) j))
    (if (and (fx<= (fx+ i 8) n) (string=? (substring s i (fx+ i 8)) "Infinity"))
        (values (fx+ i 8) +inf.0)
        (let* ([i1 (digits-end i)]
               [int (substring s i i1)]
               [has-dot (eqv? (at i1) #\.)]
               [i2 (if has-dot (digits-end (fx+ i1 1)) i1)]
               [frac (if has-dot (substring s (fx+ i1 1) i2) "")])
          (if (and (fx= (string-length int) 0) (fx= (string-length frac) 0))
              (values #f #f)
              ;; optional exponent
              (let* ([e (at i2)]
                     [exp-end+val
                      (if (memv e '(#\e #\E))
                          (let* ([j (fx+ i2 1)]
                                 [sign (at j)]
                                 [j (if (memv sign '(#\+ #\-)) (fx+ j 1) j)]
                                 [k (digits-end j)])
                            (if (fx> k j)
                                (cons k (* (if (eqv? sign #\-) -1 1)
                                           (string->number (substring s j k) 10)))
                                #f))
                          #f)]
                     [end (if exp-end+val (car exp-end+val) i2)]
                     [ex (if exp-end+val (cdr exp-end+val) 0)])
                (values end
                        (decimal->flonum (string-append int frac)
                                         (- ex (string-length frac))))))))))

(define (str-white? c)
  (memv c '(#\tab #\space #\page #\vtab #\return #\newline)))

(define (skip-str-white s i)
  (let ([n (string-length s)])
    (let loop ([i i]) (if (and (fx< i n) (str-white? (string-ref s i))) (loop (fx+ i 1)) i))))

;; ToNumber applied to a string (9.3.1).
(define (string->js-number s)
  (let* ([n (string-length s)]
         [i (skip-str-white s 0)]
         [j (let loop ([j n]) (if (and (fx> j i) (str-white? (string-ref s (fx- j 1)))) (loop (fx- j 1)) j))])
    (cond
      [(fx= i j) 0.0]
      [(and (fx>= (fx- j i) 2) (char=? (string-ref s i) #\0)
            (memv (string-ref s (fx+ i 1)) '(#\x #\X)))
       (let loop ([k (fx+ i 2)] [v 0])
         (if (fx= k j)
             (if (fx= k (fx+ i 2)) +nan.0 (inexact v))
             (let ([d (hex-digit-value (string-ref s k))])
               (if d (loop (fx+ k 1) (+ (* v 16) d)) +nan.0))))]
      [else
       (let* ([c (string-ref s i)]
              [neg (char=? c #\-)]
              [start (if (or neg (char=? c #\+)) (fx+ i 1) i)])
         (let-values ([(end v) (scan-str-decimal s start)])
           (if (and end (fx= end j))
               (if neg (fl- v) v)
               +nan.0)))])))
