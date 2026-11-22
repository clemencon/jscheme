;;; JS strings are Scheme strings whose characters stand for UTF-16 code
;;; units. Surrogate code units (#xD800-#xDFFF) cannot be Scheme chars, so
;;; they are kept in the range #x10F800-#x10FFFF.

(define surrogate-offset (- #x10F800 #xD800))

(define (unit->char u)
  (if (fx<= #xD800 u #xDFFF)
      (integer->char (fx+ u surrogate-offset))
      (integer->char u)))

(define (char->unit c)
  (let ([i (char->integer c)])
    (if (fx>= i #x10F800) (fx- i surrogate-offset) i)))

;; A Scheme string of code points (from the command line, or decoded UTF-8)
;; to a JS string of code units.
(define (codepoints->js s)
  (if (let loop ([i 0]) (and (< i (string-length s)) (or (> (char->integer (string-ref s i)) #xFFFF) (loop (+ i 1)))))
      (let ([out (open-output-string)])
        (string-for-each
         (lambda (c)
           (let ([cp (char->integer c)])
             (if (> cp #xFFFF)
                 (let ([v (- cp #x10000)])
                   (write-char (unit->char (+ #xD800 (ash v -10))) out)
                   (write-char (unit->char (+ #xDC00 (logand v #x3FF))) out))
                 (write-char c out))))
         s)
        (get-output-string out))
      s))

;; Decode UTF-8 bytes leniently (invalid sequences become U+FFFD).
(define (utf8->js bv)
  (let ([n (bytevector-length bv)] [out (open-output-string)])
    (define (b k) (bytevector-u8-ref bv k))
    (define (cont? k) (and (< k n) (= (logand (b k) #xC0) #x80)))
    (define (emit cp)
      (if (> cp #xFFFF)
          (let ([v (- cp #x10000)])
            (write-char (unit->char (+ #xD800 (ash v -10))) out)
            (write-char (unit->char (+ #xDC00 (logand v #x3FF))) out))
          (write-char (unit->char cp) out)))
    (let loop ([i 0])
      (when (< i n)
        (let ([x (b i)])
          (cond
           [(< x #x80) (write-char (integer->char x) out) (loop (+ i 1))]
           [(and (= (logand x #xE0) #xC0) (cont? (+ i 1)))
            (emit (logor (ash (logand x #x1F) 6) (logand (b (+ i 1)) #x3F))) (loop (+ i 2))]
           [(and (= (logand x #xF0) #xE0) (cont? (+ i 1)) (cont? (+ i 2)))
            (emit (logor (ash (logand x #x0F) 12) (ash (logand (b (+ i 1)) #x3F) 6) (logand (b (+ i 2)) #x3F)))
            (loop (+ i 3))]
           [(and (= (logand x #xF8) #xF0) (cont? (+ i 1)) (cont? (+ i 2)) (cont? (+ i 3)))
            (emit (logor (ash (logand x #x07) 18) (ash (logand (b (+ i 1)) #x3F) 12)
                         (ash (logand (b (+ i 2)) #x3F) 6) (logand (b (+ i 3)) #x3F)))
            (loop (+ i 4))]
           [else (emit #xFFFD) (loop (+ i 1))]))))
    (get-output-string out)))

;; A JS string to a Scheme string of code points for output: surrogate
;; pairs are combined, lone surrogates become U+FFFD.
(define (js->codepoints s)
  (let ([n (string-length s)])
    (if (let loop ([i 0]) (and (< i n) (or (fx>= (char->integer (string-ref s i)) #x10F800) (loop (+ i 1)))))
        (let ([out (open-output-string)])
          (let loop ([i 0])
            (when (< i n)
              (let ([u (char->unit (string-ref s i))])
                (cond
                 [(and (<= #xD800 u #xDBFF) (< (+ i 1) n)
                       (<= #xDC00 (char->unit (string-ref s (+ i 1))) #xDFFF))
                  (write-char (integer->char (+ #x10000 (ash (- u #xD800) 10)
                                                (- (char->unit (string-ref s (+ i 1))) #xDC00)))
                              out)
                  (loop (+ i 2))]
                 [(<= #xD800 u #xDFFF) (write-char #\xFFFD out) (loop (+ i 1))]
                 [else (write-char (string-ref s i) out) (loop (+ i 1))]))))
          (get-output-string out))
        s)))
