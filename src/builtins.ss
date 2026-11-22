;;; Built-in objects (§15) and the host print function.

(define object-prototype (make-obj #f 'Object))
(define function-prototype (make-obj object-prototype 'Function))
(define array-prototype (make-obj object-prototype 'Array))
(define string-prototype (make-obj object-prototype 'String))
(define boolean-prototype (make-obj object-prototype 'Boolean))
(define number-prototype (make-obj object-prototype 'Number))
(define date-prototype (make-obj object-prototype 'Date))
(define global-object (make-obj object-prototype 'Global))
(define math-object (make-obj object-prototype 'Math))

(define (arg args i)
  (let loop ([args args] [i i])
    (cond [(null? args) js-undefined]
          [(fx= i 0) (car args)]
          [else (loop (cdr args) (fx- i 1))])))

(define (make-native name len proc)
  (let ([f (make-obj function-prototype 'Function)])
    (jsobj-call-set! f proc)
    (jsobj-info-set! f name)
    (add-prop! f "length" (fixnum->flonum len) (fxior RO DD DE))
    f))

(define (defmethod! o name len proc)
  (let ([f (make-native name len proc)])
    (add-prop! o name f DE)
    f))

(define (defconst! o name v)
  (add-prop! o name v (fxior RO DD DE)))

(define (make-string-object s)
  (let ([o (make-obj string-prototype 'String)])
    (jsobj-value-set! o s)
    (add-prop! o "length" (fixnum->flonum (string-length s)) (fxior RO DD DE))
    o))
(define (make-number-object n)
  (let ([o (make-obj number-prototype 'Number)])
    (jsobj-value-set! o n)
    o))
(define (make-boolean-object b)
  (let ([o (make-obj boolean-prototype 'Boolean)])
    (jsobj-value-set! o b)
    o))

(define (make-array-object elems)
  (let ([a (make-obj array-prototype 'Array)])
    (add-prop! a "length" (fixnum->flonum (length elems)) (fxior DE DD))
    (let loop ([i 0] [es elems])
      (unless (null? es)
        (add-prop! a (number->string i) (car es) 0)
        (loop (fx+ i 1) (cdr es))))
    a))

(define (check-class this class what)
  (unless (and (jsobj? this) (eq? (jsobj-class this) class))
    (throw-error "~a called on an object that is not a ~a" what class)))

;; Output

(define stdout (standard-output-port (buffer-mode block)))

(define (write-js-string s)
  (put-bytevector stdout (string->utf8 (js->codepoints s))))

(define (flush-stdout) (flush-output-port stdout))

;; ---------------------------------------------------------------------
;; Global object (§15.1)

(define (add-global! name v) (add-prop! global-object name v DE))

(add-global! "NaN" +nan.0)
(add-global! "Infinity" +inf.0)
;; Not in ES1; an extension allowed by §2.
(add-global! "undefined" js-undefined)

(add-global! "print"
  (make-native "print" 0
    (lambda (this args)
      (let loop ([as args] [first #t])
        (unless (null? as)
          (unless first (put-u8 stdout 32))
          (write-js-string (to-string (car as)))
          (loop (cdr as) #f)))
      (put-u8 stdout 10)
      js-undefined)))

(add-global! "eval"
  (make-native "eval" 1
    (lambda (this args)
      (let ([x (arg args 0)] [c *caller-ctx*])
        (if (string? x)
            (eval-in-context x c 0)
            x)))))

(define (radix-digit c r)
  (let ([d (cond [(char<=? #\0 c #\9) (- (char->integer c) 48)]
                 [(char<=? #\a c #\z) (- (char->integer c) 87)]
                 [(char<=? #\A c #\Z) (- (char->integer c) 55)]
                 [else 99])])
    (and (< d r) d)))

(add-global! "parseInt"
  (make-native "parseInt" 2
    (lambda (this args)
      (let* ([s (to-string (arg args 0))]
             [n (string-length s)]
             [i (let loop ([i 0]) (if (and (< i n) (str-ws? (string-ref s i))) (loop (+ i 1)) i))]
             [sign (if (and (< i n) (char=? (string-ref s i) #\-)) -1.0 1.0)]
             [i (if (and (< i n) (memv (string-ref s i) '(#\+ #\-))) (+ i 1) i)]
             [rest (substring s i n)]
             [rn (string-length rest)]
             [radix (to-int32 (arg args 1))])
        (define (has-0x?) (and (>= rn 2) (char=? (string-ref rest 0) #\0) (memv (string-ref rest 1) '(#\x #\X))))
        (let-values ([(R S)
                      (cond
                       [(and (not (= radix 0)) (or (< radix 2) (> radix 36))) (values #f #f)]
                       [(not (= radix 0))
                        (values radix (if (and (= radix 16) (has-0x?)) (substring rest 2 rn) rest))]
                       [(or (= rn 0) (not (char=? (string-ref rest 0) #\0))) (values 10 rest)]
                       [(has-0x?) (values 16 (substring rest 2 rn))]
                       [else (values 8 rest)])])
          (if (not R)
              +nan.0
              (let loop ([j 0] [v 0])
                (let ([d (and (< j (string-length S)) (radix-digit (string-ref S j) R))])
                  (if d
                      (loop (+ j 1) (+ (* v R) d))
                      (if (= j 0) +nan.0 (fl* sign (inexact v))))))))))))

(add-global! "parseFloat"
  (make-native "parseFloat" 1
    (lambda (this args)
      (let* ([s (to-string (arg args 0))]
             [n (string-length s)]
             [i (let loop ([i 0]) (if (and (< i n) (str-ws? (string-ref s i))) (loop (+ i 1)) i))])
        (let-values ([(end v) (scan-decimal s i)])
          (if end v +nan.0))))))

(define hex-chars "0123456789ABCDEF")
(define (hex-string n width)
  (let loop ([n n] [w width] [acc '()])
    (if (= w 0) (list->string acc)
        (loop (quotient n 16) (- w 1) (cons (string-ref hex-chars (remainder n 16)) acc)))))

(add-global! "escape"
  (make-native "escape" 1
    (lambda (this args)
      (let ([s (to-string (arg args 0))] [out (open-output-string)])
        (string-for-each
         (lambda (c)
           (let ([u (char->unit c)])
             (cond
              [(or (ascii-letter? c) (digit-char? c) (memv c '(#\@ #\* #\_ #\+ #\- #\. #\/)))
               (write-char c out)]
              [(< u 256) (write-char #\% out) (put-string out (hex-string u 2))]
              [else (put-string out "%u") (put-string out (hex-string u 4))])))
         s)
        (get-output-string out)))))

(add-global! "unescape"
  (make-native "unescape" 1
    (lambda (this args)
      (let* ([s (to-string (arg args 0))] [n (string-length s)] [out (open-output-string)])
        (define (hex k) (and (< k n) (hex-digit-value (string-ref s k))))
        (let loop ([k 0])
          (when (< k n)
            (let ([c (string-ref s k)])
              (cond
               [(and (char=? c #\%) (<= (+ k 6) n) (char=? (string-ref s (+ k 1)) #\u)
                     (hex (+ k 2)) (hex (+ k 3)) (hex (+ k 4)) (hex (+ k 5)))
                (write-char (unit->char (+ (* 4096 (hex (+ k 2))) (* 256 (hex (+ k 3)))
                                           (* 16 (hex (+ k 4))) (hex (+ k 5))))
                            out)
                (loop (+ k 6))]
               [(and (char=? c #\%) (<= (+ k 3) n) (hex (+ k 1)) (hex (+ k 2)))
                (write-char (unit->char (+ (* 16 (hex (+ k 1))) (hex (+ k 2)))) out)
                (loop (+ k 3))]
               [else (write-char c out) (loop (+ k 1))]))))
        (get-output-string out)))))

(add-global! "isNaN"
  (make-native "isNaN" 1 (lambda (this args) (nan? (to-number (arg args 0))))))
(add-global! "isFinite"
  (make-native "isFinite" 1 (lambda (this args) (finite? (to-number (arg args 0))))))

;; ---------------------------------------------------------------------
;; Object (§15.2)

(define object-constructor
  (make-native "Object" 1
    (lambda (this args)
      (let ([v (arg args 0)])
        (if (or (undefined? v) (js-null? v))
            (make-obj object-prototype 'Object)
            (to-object v))))))
(jsobj-construct-set! object-constructor
  (lambda (args)
    (let ([v (arg args 0)])
      (if (or (undefined? v) (js-null? v))
          (make-obj object-prototype 'Object)
          (to-object v)))))
(add-global! "Object" object-constructor)
(add-prop! object-constructor "prototype" object-prototype (fxior RO DD DE))
(add-prop! object-prototype "constructor" object-constructor DE)

(define (class-name o)
  (let ([c (jsobj-class o)])
    (case c
      [(activation) "Object"]
      [else (symbol->string c)])))

(defmethod! object-prototype "toString" 0
  (lambda (this args)
    (let ([o (to-object this)])
      (string-append "[object " (class-name o) "]"))))
(defmethod! object-prototype "valueOf" 0
  (lambda (this args) (to-object this)))

;; ---------------------------------------------------------------------
;; Function (§15.3)

(define (join-strings strs sep)
  (if (null? strs)
      ""
      (let ([out (open-output-string)])
        (put-string out (car strs))
        (for-each (lambda (s) (put-string out sep) (put-string out s)) (cdr strs))
        (get-output-string out))))

(define (function-constructor-proc args)
  (let* ([n (length args)]
         [pstrs (let loop ([as args] [k (- n 1)])
                  (if (<= k 0) '()
                      (let ([s (to-string (car as))])
                        (cons s (loop (cdr as) (- k 1))))))]
         [P (join-strings pstrs ",")]
         [body (if (= n 0) "" (to-string (list-ref args (- n 1))))])
    (let* ([params (guard (e [(syntax-err? e) (throw-error "SyntaxError in parameters: ~a" (syntax-err-msg e))])
                     (parse-js P 'params))]
           [stmts (guard (e [(syntax-err? e) (throw-error "SyntaxError in function body: ~a" (syntax-err-msg e))])
                    (parse-js body 'function-body))])
      (make-script-function
       (make-fninfo "anonymous" params (compile-stmt-list stmts) (collect-vars stmts)
                    (string-append "function anonymous(" P "\n) {\n" body "\n}"))))))

(define function-constructor
  (make-native "Function" 1 (lambda (this args) (function-constructor-proc args))))
(jsobj-construct-set! function-constructor function-constructor-proc)
(add-global! "Function" function-constructor)
(add-prop! function-constructor "prototype" function-prototype (fxior RO DD DE))

;; Function.prototype is a function that returns undefined
(jsobj-call-set! function-prototype (lambda (this args) js-undefined))
(jsobj-info-set! function-prototype "anonymous")
(add-prop! function-prototype "length" 0.0 (fxior RO DD DE))
(add-prop! function-prototype "constructor" function-constructor DE)
(defmethod! function-prototype "toString" 0
  (lambda (this args)
    (unless (and (jsobj? this) (jsobj-call this))
      (throw-error "Function.prototype.toString called on an object that is not a function"))
    (let ([info (jsobj-info this)])
      (if (fninfo? info)
          (fninfo-source info)
          (string-append "function " (if (string? info) info "anonymous") "() { }")))))

;; ---------------------------------------------------------------------
;; Array (§15.4)

(define (array-constructor-proc args)
  (if (and (= (length args) 1) (flonum? (car args)))
      (let ([a (make-obj array-prototype 'Array)])
        (add-prop! a "length" (inexact (to-uint32 (car args))) (fxior DE DD))
        a)
      (make-array-object args)))

(define array-constructor
  (make-native "Array" 1 (lambda (this args) (array-constructor-proc args))))
(jsobj-construct-set! array-constructor array-constructor-proc)
(add-global! "Array" array-constructor)
(add-prop! array-constructor "prototype" array-prototype (fxior RO DD DE))
(add-prop! array-prototype "length" 0.0 (fxior DE DD))
(add-prop! array-prototype "constructor" array-constructor DE)

(define (index-name k) (number->string k))

(define (array-join this sep-arg)
  (let* ([o (to-object this)]
         [len (to-uint32 (js-get o "length"))]
         [sep (if (undefined? sep-arg) "," (to-string sep-arg))])
    (if (= len 0)
        ""
        (let ([out (open-output-string)])
          (let loop ([k 0])
            (when (< k len)
              (when (> k 0) (put-string out sep))
              (let ([v (js-get o (index-name k))])
                (unless (or (undefined? v) (js-null? v))
                  (put-string out (to-string v))))
              (loop (+ k 1))))
          (get-output-string out)))))

(defmethod! array-prototype "toString" 0
  (lambda (this args) (array-join this js-undefined)))
(defmethod! array-prototype "join" 1
  (lambda (this args) (array-join this (arg args 0))))
(defmethod! array-prototype "reverse" 0
  (lambda (this args)
    (let* ([o (to-object this)]
           [len (to-uint32 (js-get o "length"))]
           [mid (quotient len 2)])
      (let loop ([k 0])
        (when (< k mid)
          (let* ([lo (index-name k)] [hi (index-name (- len k 1))]
                 [lov (js-get o lo)] [hiv (js-get o hi)])
            (if (js-has-property? o hi) (js-put! o lo hiv) (js-delete! o lo))
            (if (js-has-property? o lo) (js-put! o hi lov) (js-delete! o hi))
            (loop (+ k 1)))))
      o)))

(defmethod! array-prototype "sort" 1
  (lambda (this args)
    (let* ([o (to-object this)]
           [len (to-uint32 (js-get o "length"))]
           [cmpfn (arg args 0)]
           [has-cmp (not (null? args))]
           [vals (let loop ([k 0] [acc '()])
                   (if (< k len) (loop (+ k 1) (cons (js-get o (index-name k)) acc)) (reverse acc)))])
      (define (sort-compare x y)
        (cond [(and (undefined? x) (undefined? y)) 0.0]
              [(undefined? x) 1.0]
              [(undefined? y) -1.0]
              [has-cmp (to-number (js-call cmpfn js-undefined (list x y)))]
              [else (let* ([sx (to-string x)] [sy (to-string y)])
                      (cond [(string<? sx sy) -1.0] [(string<? sy sx) 1.0] [else 0.0]))]))
      (let ([sorted (list-sort (lambda (x y) (fl< (sort-compare x y) 0.0)) vals)])
        (let loop ([k 0] [vs sorted])
          (unless (null? vs)
            (js-put! o (index-name k) (car vs))
            (loop (+ k 1) (cdr vs)))))
      o)))

;; ---------------------------------------------------------------------
;; String (§15.5)

(define string-constructor
  (make-native "String" 1
    (lambda (this args) (if (null? args) "" (to-string (car args))))))
(jsobj-construct-set! string-constructor
  (lambda (args) (make-string-object (if (null? args) "" (to-string (car args))))))
(add-global! "String" string-constructor)
(add-prop! string-constructor "prototype" string-prototype (fxior RO DD DE))
(jsobj-value-set! string-prototype "")
(add-prop! string-prototype "length" 0.0 (fxior RO DD DE))
(add-prop! string-prototype "constructor" string-constructor DE)

(defmethod! string-constructor "fromCharCode" 1
  (lambda (this args)
    (let ([out (open-output-string)])
      (for-each (lambda (a) (write-char (unit->char (to-uint16 a)) out)) args)
      (get-output-string out))))

(defmethod! string-prototype "toString" 0
  (lambda (this args)
    (check-class this 'String "String.prototype.toString")
    (jsobj-value this)))
(defmethod! string-prototype "valueOf" 0
  (lambda (this args)
    (check-class this 'String "String.prototype.valueOf")
    (jsobj-value this)))

(define (clamp-index x len)
  ;; x: flonum integer (or infinite)
  (cond [(fl< x 0.0) 0] [(fl> x (fixnum->flonum len)) len] [else (flonum->fixnum x)]))

(defmethod! string-prototype "charAt" 1
  (lambda (this args)
    (let* ([s (to-string this)] [pos (to-integer (arg args 0))] [n (string-length s)])
      (if (or (fl< pos 0.0) (fl>= pos (fixnum->flonum n)))
          ""
          (string (string-ref s (flonum->fixnum pos)))))))
(defmethod! string-prototype "charCodeAt" 1
  (lambda (this args)
    (let* ([s (to-string this)] [pos (to-integer (arg args 0))] [n (string-length s)])
      (if (or (fl< pos 0.0) (fl>= pos (fixnum->flonum n)))
          +nan.0
          (fixnum->flonum (char->unit (string-ref s (flonum->fixnum pos))))))))

(define (substring-at? s k sub)
  (let ([m (string-length sub)])
    (and (<= (+ k m) (string-length s))
         (let loop ([j 0])
           (or (= j m)
               (and (char=? (string-ref s (+ k j)) (string-ref sub j))
                    (loop (+ j 1))))))))

(defmethod! string-prototype "indexOf" 2
  (lambda (this args)
    (let* ([s (to-string this)]
           [sub (to-string (arg args 0))]
           [pos (to-integer (arg args 1))]
           [n (string-length s)]
           [start (clamp-index pos n)])
      (let loop ([k start])
        (cond [(> (+ k (string-length sub)) n) -1.0]
              [(substring-at? s k sub) (fixnum->flonum k)]
              [else (loop (+ k 1))])))))

(defmethod! string-prototype "lastIndexOf" 2
  (lambda (this args)
    (let* ([s (to-string this)]
           [sub (to-string (arg args 0))]
           [p (to-number (arg args 1))]
           [pos (if (nan? p) +inf.0 (to-integer p))]
           [n (string-length s)]
           [start (clamp-index pos n)])
      (let loop ([k (min start (- n (string-length sub)))])
        (cond [(< k 0) -1.0]
              [(substring-at? s k sub) (fixnum->flonum k)]
              [else (loop (- k 1))])))))

(defmethod! string-prototype "split" 1
  (lambda (this args)
    (let ([s (to-string this)])
      (if (null? args)
          (make-array-object (list s))
          (let* ([sep (to-string (car args))] [n (string-length s)] [m (string-length sep)])
            (make-array-object
             (if (= m 0)
                 (let loop ([p 0]) (if (= p n) '() (cons (string (string-ref s p)) (loop (+ p 1)))))
                 (let loop ([p 0] [k 0])
                   (cond [(> (+ k m) n) (list (substring s p n))]
                         [(substring-at? s k sep) (cons (substring s p k) (loop (+ k m) (+ k m)))]
                         [else (loop p (+ k 1))])))))))))

(defmethod! string-prototype "substring" 2
  (lambda (this args)
    (let* ([s (to-string this)] [n (string-length s)]
           [start (clamp-index (to-integer-nan0 (arg args 0)) n)]
           [end (if (< (length args) 2) n (clamp-index (to-integer-nan0 (arg args 1)) n))])
      (substring s (min start end) (max start end)))))

(define (to-integer-nan0 v) (to-integer v))

(define (map-units f s)
  (let ([out (make-string (string-length s))])
    (let loop ([i 0])
      (when (< i (string-length s))
        (let ([c (string-ref s i)])
          (string-set! out i (if (fx>= (char->integer c) #x10F800) c (f c))))
        (loop (+ i 1))))
    out))

(define (simple-downcase c)
  (let ([d (char-downcase c)]) (if (fx> (char->integer d) #xFFFF) c d)))
(define (simple-upcase c)
  (let ([d (char-upcase c)]) (if (fx> (char->integer d) #xFFFF) c d)))

(defmethod! string-prototype "toLowerCase" 0
  (lambda (this args) (map-units simple-downcase (to-string this))))
(defmethod! string-prototype "toUpperCase" 0
  (lambda (this args) (map-units simple-upcase (to-string this))))

;; ---------------------------------------------------------------------
;; Boolean (§15.6)

(define boolean-constructor
  (make-native "Boolean" 1 (lambda (this args) (to-boolean (arg args 0)))))
(jsobj-construct-set! boolean-constructor
  (lambda (args) (make-boolean-object (to-boolean (arg args 0)))))
(add-global! "Boolean" boolean-constructor)
(add-prop! boolean-constructor "prototype" boolean-prototype (fxior RO DD DE))
(jsobj-value-set! boolean-prototype #f)
(add-prop! boolean-prototype "constructor" boolean-constructor DE)
(defmethod! boolean-prototype "toString" 0
  (lambda (this args)
    (check-class this 'Boolean "Boolean.prototype.toString")
    (if (jsobj-value this) "true" "false")))
(defmethod! boolean-prototype "valueOf" 0
  (lambda (this args)
    (check-class this 'Boolean "Boolean.prototype.valueOf")
    (jsobj-value this)))

;; ---------------------------------------------------------------------
;; Number (§15.7)

(define number-constructor
  (make-native "Number" 1 (lambda (this args) (if (null? args) 0.0 (to-number (car args))))))
(jsobj-construct-set! number-constructor
  (lambda (args) (make-number-object (if (null? args) 0.0 (to-number (car args))))))
(add-global! "Number" number-constructor)
(add-prop! number-constructor "prototype" number-prototype (fxior RO DD DE))
(defconst! number-constructor "MAX_VALUE" 1.7976931348623157e308)
(defconst! number-constructor "MIN_VALUE" 5e-324)
(defconst! number-constructor "NaN" +nan.0)
(defconst! number-constructor "NEGATIVE_INFINITY" -inf.0)
(defconst! number-constructor "POSITIVE_INFINITY" +inf.0)
(jsobj-value-set! number-prototype 0.0)
(add-prop! number-prototype "constructor" number-constructor DE)

(define (number->radix-string x r)
  (cond
   [(nan? x) "NaN"]
   [(infinite? x) (if (fl> x 0.0) "Infinity" "-Infinity")]
   [(fl< x 0.0) (string-append "-" (number->radix-string (fl- x) r))]
   [else
    (let* ([ip (exact (flfloor x))]
           [fp (fl- x (flfloor x))]
           [is (string-downcase (number->string ip r))])
      (if (fl= fp 0.0)
          is
          (let loop ([f fp] [k 0] [acc '()])
            (if (or (fl= f 0.0) (= k 20))
                (string-append is "." (list->string (reverse acc)))
                (let* ([v (fl* f (fixnum->flonum r))] [d (flonum->fixnum (flfloor v))])
                  (loop (fl- v (flfloor v)) (+ k 1)
                        (cons (string-ref "0123456789abcdefghijklmnopqrstuvwxyz" d) acc)))))))]))

(defmethod! number-prototype "toString" 1
  (lambda (this args)
    (check-class this 'Number "Number.prototype.toString")
    (let ([x (jsobj-value this)] [r (arg args 0)])
      (if (undefined? r)
          (number->js-string x)
          (let ([radix (to-integer r)])
            (if (or (fl= radix 10.0) (fl< radix 2.0) (fl> radix 36.0))
                (number->js-string x)
                (number->radix-string x (flonum->fixnum radix))))))))
(defmethod! number-prototype "valueOf" 0
  (lambda (this args)
    (check-class this 'Number "Number.prototype.valueOf")
    (jsobj-value this)))

;; ---------------------------------------------------------------------
;; Math (§15.8)

(add-global! "Math" math-object)
(defconst! math-object "E" (exp 1.0))
(defconst! math-object "LN10" (log 10.0))
(defconst! math-object "LN2" (log 2.0))
(defconst! math-object "LOG2E" (/ 1.0 (log 2.0)))
(defconst! math-object "LOG10E" (/ 1.0 (log 10.0)))
(defconst! math-object "PI" (* 4.0 (atan 1.0)))
(defconst! math-object "SQRT1_2" (sqrt 0.5))
(defconst! math-object "SQRT2" (sqrt 2.0))

(define-syntax defmath1
  (syntax-rules ()
    [(_ name (x) body ...)
     (defmethod! math-object name 1
       (lambda (this args) (let ([x (to-number (arg args 0))]) body ...)))]))

(defmath1 "abs" (x) (flabs x))
(defmath1 "acos" (x) (if (or (nan? x) (fl> (flabs x) 1.0)) +nan.0 (flacos x)))
(defmath1 "asin" (x) (if (or (nan? x) (fl> (flabs x) 1.0)) +nan.0 (flasin x)))
(defmath1 "atan" (x) (flatan x))
(defmath1 "ceil" (x) (flceiling x))
(defmath1 "cos" (x) (flcos x))
(defmath1 "exp" (x) (flexp x))
(defmath1 "floor" (x) (flfloor x))
(defmath1 "log" (x) (if (fl< x 0.0) +nan.0 (fllog x)))
(defmath1 "sin" (x) (flsin x))
(defmath1 "sqrt" (x) (if (fl< x 0.0) +nan.0 (flsqrt x)))
(defmath1 "tan" (x) (fltan x))
(defmath1 "round" (x)
  (cond [(or (nan? x) (infinite? x) (fl= x 0.0)) x]
        [(and (fl< x 0.0) (fl>= x -0.5)) -0.0]
        [(fl>= (flabs x) 4503599627370496.0) x]
        [else (flfloor (fl+ x 0.5))]))

(defmethod! math-object "atan2" 2
  (lambda (this args)
    (let* ([y (to-number (arg args 0))] [x (to-number (arg args 1))])
      (flatan y x))))

(defmethod! math-object "max" 2
  (lambda (this args)
    (let* ([x (to-number (arg args 0))] [y (to-number (arg args 1))])
      (cond [(or (nan? x) (nan? y)) +nan.0]
            [(fl> x y) x]
            [(fl> y x) y]
            [(and (fl= x 0.0) (fl= y 0.0)) (if (and (eqv? x -0.0) (eqv? y -0.0)) -0.0 0.0)]
            [else x]))))
(defmethod! math-object "min" 2
  (lambda (this args)
    (let* ([x (to-number (arg args 0))] [y (to-number (arg args 1))])
      (cond [(or (nan? x) (nan? y)) +nan.0]
            [(fl< x y) x]
            [(fl< y x) y]
            [(and (fl= x 0.0) (fl= y 0.0)) (if (or (eqv? x -0.0) (eqv? y -0.0)) -0.0 0.0)]
            [else x]))))

(define (js-pow x y)
  (cond
   [(nan? y) +nan.0]
   [(fl= y 0.0) 1.0]
   [(nan? x) +nan.0]
   [(infinite? y)
    (let ([ax (flabs x)])
      (cond [(fl= ax 1.0) +nan.0]
            [(fl> ax 1.0) (if (fl> y 0.0) +inf.0 0.0)]
            [else (if (fl> y 0.0) 0.0 +inf.0)]))]
   [else (flexpt x y)]))

(defmethod! math-object "pow" 2
  (lambda (this args)
    (let* ([x (to-number (arg args 0))] [y (to-number (arg args 1))])
      (js-pow x y))))

(random-seed (let ([t (current-time)]) (+ 1 (mod (+ (time-second t) (time-nanosecond t)) 4294967087))))
(defmethod! math-object "random" 0
  (lambda (this args) (random 1.0)))
