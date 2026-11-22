;;; Values, objects and type conversions (§8, §9).
;;;
;;; undefined -> js-undefined, null -> js-null, booleans -> #t/#f,
;;; numbers -> flonums, strings -> Scheme strings of code units,
;;; objects -> jsobj records.

(define js-undefined 'undefined)
(define js-null 'null)
(define (undefined? v) (eq? v js-undefined))
(define (js-null? v) (eq? v js-null))

(define-record-type js-error (fields msg))
(define (throw-error fmt . args)
  (raise (make-js-error (apply format fmt args))))

;; Property attributes (§8.6.1)
(define RO 1)
(define DE 2)
(define DD 4)

(define-record-type prop (fields (mutable value) (mutable attrs) (immutable seq)))
(define-record-type prop-alias (fields target))

(define-record-type jsobj
  (fields (mutable proto)        ; jsobj or #f
          (mutable class)        ; symbol
          (immutable props)      ; hashtable string -> prop
          (mutable value)        ; [[Value]]
          (mutable call)         ; #f or (lambda (this args) ...)
          (mutable construct)    ; #f or (lambda (args) ...)
          (mutable info)))       ; function information

(define prop-counter 0)

(define (make-obj proto class)
  (make-jsobj proto class (make-hashtable string-hash string=?) #f #f #f #f))

(define (own-prop o name) (hashtable-ref (jsobj-props o) name #f))

(define (prop-get p)
  (let ([v (prop-value p)])
    (if (prop-alias? v) (prop-value (prop-alias-target v)) v)))

(define (prop-set! p v)
  (let ([x (prop-value p)])
    (if (prop-alias? x) (prop-value-set! (prop-alias-target x) v) (prop-value-set! p v))))

(define (add-prop! o name v attrs)
  (set! prop-counter (fx+ prop-counter 1))
  (hashtable-set! (jsobj-props o) name (make-prop v attrs prop-counter)))

;; define or replace, used when building built-ins
(define (define-prop! o name v attrs)
  (let ([p (own-prop o name)])
    (if p
        (begin (prop-value-set! p v) (prop-attrs-set! p attrs))
        (add-prop! o name v attrs))))

(define (attr? p a) (fxlogtest (prop-attrs p) a))

;; [[Get]] (§8.6.2.1)
(define (js-get o name)
  (let loop ([o o])
    (if o
        (let ([p (hashtable-ref (jsobj-props o) name #f)])
          (if p (prop-get p) (loop (jsobj-proto o))))
        js-undefined)))

;; [[CanPut]] (§8.6.2.3)
(define (js-can-put? o name)
  (let loop ([o o])
    (if o
        (let ([p (hashtable-ref (jsobj-props o) name #f)])
          (if p (not (attr? p RO)) (loop (jsobj-proto o))))
        #t)))

;; [[Put]] (§8.6.2.2, §15.4.5.1)
(define (js-put! o name v)
  (if (eq? (jsobj-class o) 'Array)
      (array-put! o name v)
      (when (js-can-put? o name)
        (let ([p (own-prop o name)])
          (if p (prop-set! p v) (add-prop! o name v 0))))))

;; [[HasProperty]] (§8.6.2.4)
(define (js-has-property? o name)
  (let loop ([o o])
    (and o (or (and (hashtable-ref (jsobj-props o) name #f) #t)
               (loop (jsobj-proto o))))))

;; [[Delete]] (§8.6.2.5)
(define (js-delete! o name)
  (let ([p (own-prop o name)])
    (cond [(not p) #t]
          [(attr? p DD) #f]
          [else (hashtable-delete! (jsobj-props o) name) #t])))

;; Array index (§15.4): canonical numeric string below 2^32-1.
(define (array-index name)
  (let ([n (string-length name)])
    (and (fx> n 0) (fx<= n 10)
         (let loop ([i 0])
           (or (fx= i n)
               (and (char<=? #\0 (string-ref name i) #\9) (loop (fx+ i 1)))))
         (or (fx= n 1) (not (char=? (string-ref name 0) #\0)))
         (let ([v (string->number name)])
           (and (< v 4294967295) v)))))

(define (array-put! a name v)
  (when (js-can-put? a name)
    (let ([p (own-prop a name)])
      (if (and p (string=? name "length"))
          (let* ([newlen (to-uint32 v)]
                 [oldlen (to-uint32 (prop-get p))])
            (when (< newlen oldlen)
              (let ([keys (vector->list (hashtable-keys (jsobj-props a)))])
                (for-each (lambda (k)
                            (let ([i (array-index k)])
                              (when (and i (>= i newlen) (< i oldlen))
                                (hashtable-delete! (jsobj-props a) k))))
                          keys)))
            (prop-set! p (inexact newlen)))
          (begin
            (if p (prop-set! p v) (add-prop! a name v 0))
            (let ([i (array-index name)])
              (when i
                (let ([lp (own-prop a "length")])
                  (if lp
                      (unless (< i (to-uint32 (prop-get lp)))
                        (prop-set! lp (inexact (+ i 1))))
                      (add-prop! a "length" (inexact (+ i 1)) (fxior DE DD)))))))))))

;; ---------------------------------------------------------------------
;; Type tests

(define (js-object? v) (jsobj? v))
(define (js-callable? v) (and (jsobj? v) (jsobj-call v) #t))

(define (js-typeof v)
  (cond [(eq? v js-undefined) "undefined"]
        [(eq? v js-null) "object"]
        [(boolean? v) "boolean"]
        [(flonum? v) "number"]
        [(string? v) "string"]
        [(jsobj-call v) "function"]
        [else "object"]))

;; ---------------------------------------------------------------------
;; Calling

(define (js-call f this args)
  (if (and (jsobj? f) (jsobj-call f))
      ((jsobj-call f) this args)
      (throw-error "~a is not a function" (js-typeof f))))

(define (js-construct f args)
  (cond [(not (jsobj? f)) (throw-error "~a is not a constructor" (js-typeof f))]
        [(not (jsobj-construct f)) (throw-error "object is not a constructor")]
        [else ((jsobj-construct f) args)]))

;; ---------------------------------------------------------------------
;; Conversions (§9)

;; [[DefaultValue]] (§8.6.2.6)
(define (default-value o hint)
  (let ([hint (or hint (if (eq? (jsobj-class o) 'Date) 'string 'number))])
    (define (try name)
      (let ([f (js-get o name)])
        (if (jsobj? f)
            (let ([r (js-call f o '())])
              (if (jsobj? r) #f (box r)))
            #f)))
    (let ([r (if (eq? hint 'string)
                 (or (try "toString") (try "valueOf"))
                 (or (try "valueOf") (try "toString")))])
      (if r (unbox r) (throw-error "cannot convert object to primitive value")))))

(define (to-primitive v hint)
  (if (jsobj? v) (default-value v hint) v))

(define (to-boolean v)
  (cond [(boolean? v) v]
        [(flonum? v) (not (or (fl= v 0.0) (nan? v)))]
        [(string? v) (fx> (string-length v) 0)]
        [(jsobj? v) #t]
        [else #f]))

(define (to-number v)
  (cond [(flonum? v) v]
        [(string? v) (string->js-number v)]
        [(boolean? v) (if v 1.0 0.0)]
        [(eq? v js-undefined) +nan.0]
        [(eq? v js-null) 0.0]
        [else (to-number (to-primitive v 'number))]))

(define (to-string v)
  (cond [(string? v) v]
        [(flonum? v) (number->js-string v)]
        [(boolean? v) (if v "true" "false")]
        [(eq? v js-undefined) "undefined"]
        [(eq? v js-null) "null"]
        [else (to-string (to-primitive v 'string))]))


;; ToInteger, as a flonum
(define (to-integer v)
  (let ([x (to-number v)])
    (cond [(nan? x) 0.0]
          [(infinite? x) x]
          [else (fltruncate x)])))

;; ToInt32 / ToUint32 / ToUint16 as exact integers
(define (to-uint32 v)
  (let ([x (if (flonum? v) v (to-number v))])
    (if (or (nan? x) (infinite? x))
        0
        (if (and (fl>= x 0.0) (fl< x 4294967296.0))
            (flonum->fixnum x)
            (modulo (exact (fltruncate x)) 4294967296)))))

(define (to-int32 v)
  (let ([u (to-uint32 v)])
    (if (>= u 2147483648) (- u 4294967296) u)))

(define (to-uint16 v)
  (modulo (to-uint32 v) 65536))

;; ToObject (§9.9)
(define (to-object v)
  (cond [(jsobj? v) v]
        [(string? v) (make-string-object v)]
        [(flonum? v) (make-number-object v)]
        [(boolean? v) (make-boolean-object v)]
        [else (throw-error "cannot convert ~a to object" (to-string v))]))

;; ---------------------------------------------------------------------
;; Number to string (§9.8.1)

;; Shortest round-tripping digits and exponent n (value = 0.d1d2... * 10^n)
(define (flonum-digits x)
  (let* ([s (number->string x)]
         [s (let ([bar (let loop ([i 0]) (cond [(= i (string-length s)) #f]
                                               [(char=? (string-ref s i) #\|) i]
                                               [else (loop (+ i 1))]))])
              (if bar (substring s 0 bar) s))]
         [epos (let loop ([i 0]) (cond [(= i (string-length s)) #f]
                                       [(char=? (string-ref s i) #\e) i]
                                       [else (loop (+ i 1))]))]
         [mant (if epos (substring s 0 epos) s)]
         [ex (if epos (string->number (substring s (+ epos 1) (string-length s))) 0)]
         [dot (let loop ([i 0]) (cond [(= i (string-length mant)) #f]
                                      [(char=? (string-ref mant i) #\.) i]
                                      [else (loop (+ i 1))]))]
         [ip (if dot (substring mant 0 dot) mant)]
         [fp (if dot (substring mant (+ dot 1) (string-length mant)) "")]
         [digits (string-append ip fp)]
         [n (+ (string-length ip) ex)])
    ;; strip leading zeros
    (let loop ([digits digits] [n n])
      (if (and (> (string-length digits) 1) (char=? (string-ref digits 0) #\0))
          (loop (substring digits 1 (string-length digits)) (- n 1))
          ;; strip trailing zeros
          (let tloop ([digits digits])
            (if (and (> (string-length digits) 1)
                     (char=? (string-ref digits (- (string-length digits) 1)) #\0))
                (tloop (substring digits 0 (- (string-length digits) 1)))
                (values digits n)))))))

(define (number->js-string x)
  (cond
   [(nan? x) "NaN"]
   [(fl= x 0.0) "0"]
   [(fl< x 0.0) (string-append "-" (number->js-string (fl- x)))]
   [(infinite? x) "Infinity"]
   [(and (fl< x 9007199254740992.0) (fl= x (flfloor x))) (number->string (exact x))]
   [else
    (let-values ([(s n) (flonum-digits x)])
      (let ([k (string-length s)])
        (cond
         [(and (<= k n) (<= n 21))
          (string-append s (make-string (- n k) #\0))]
         [(and (< 0 n) (<= n 21))
          (string-append (substring s 0 n) "." (substring s n k))]
         [(and (< -6 n) (<= n 0))
          (string-append "0." (make-string (- n) #\0) s)]
         [else
          (let* ([e (- n 1)]
                 [es (string-append (if (< e 0) "-" "+") (number->string (abs e)))])
            (if (= k 1)
                (string-append s "e" es)
                (string-append (substring s 0 1) "." (substring s 1 k) "e" es)))])))]))

;; ---------------------------------------------------------------------
;; String to number (§9.3.1)

(define (str-ws? c) (memv c '(#\tab #\space #\page #\xB #\return #\newline)))

(define (trim-str-ws s)
  (let* ([n (string-length s)]
         [a (let loop ([i 0]) (if (and (< i n) (str-ws? (string-ref s i))) (loop (+ i 1)) i))]
         [b (let loop ([j n]) (if (and (> j a) (str-ws? (string-ref s (- j 1)))) (loop (- j 1)) j))])
    (substring s a b)))

;; Scan a StrDecimalLiteral (unsigned) at position i of s. Returns the end
;; position of the longest match and its value, or #f.
(define (scan-decimal s i)
  (let ([n (string-length s)])
    (define (ch k) (if (< k n) (string-ref s k) #\nul))
    (if (and (<= (+ i 8) n) (string=? (substring s i (+ i 8)) "Infinity"))
        (values (+ i 8) +inf.0)
        (let* ([int-end (let loop ([j i]) (if (digit-char? (ch j)) (loop (+ j 1)) j))]
               [has-int (> int-end i)])
          (let-values ([(frac-start frac-end)
                        (if (char=? (ch int-end) #\.)
                            (let loop ([j (+ int-end 1)])
                              (if (digit-char? (ch j)) (loop (+ j 1)) (values (+ int-end 1) j)))
                            (values int-end int-end))])
            (let ([has-frac (> frac-end frac-start)])
              (if (not (or has-int has-frac))
                  (values #f #f)
                  (let* ([mant-end (if (char=? (ch int-end) #\.) frac-end int-end)]
                         [exp-info
                          (if (memv (ch mant-end) '(#\e #\E))
                              (let* ([k (+ mant-end 1)]
                                     [sign (case (ch k) [(#\+) 1] [(#\-) -1] [else 0])]
                                     [ds (if (= sign 0) k (+ k 1))]
                                     [de (let loop ([j ds]) (if (digit-char? (ch j)) (loop (+ j 1)) j))])
                                (if (> de ds)
                                    (cons de (* (if (= sign -1) -1 1) (string->number (substring s ds de))))
                                    #f))
                              #f)]
                         [end (if exp-info (car exp-info) mant-end)]
                         [ex (if exp-info (cdr exp-info) 0)]
                         [ip (substring s i int-end)]
                         [fp (substring s frac-start frac-end)]
                         [m (string->number (string-append "0" ip fp))])
                    (values end (decimal->number m (- ex (string-length fp))))))))))))

(define (string->js-number str)
  (let* ([s (trim-str-ws str)] [n (string-length s)])
    (cond
     [(= n 0) 0.0]
     [(and (> n 2) (char=? (string-ref s 0) #\0) (memv (string-ref s 1) '(#\x #\X)))
      (let loop ([i 2] [v 0])
        (if (= i n)
            (inexact v)
            (let ([d (hex-digit-value (string-ref s i))])
              (if d (loop (+ i 1) (+ (* v 16) d)) +nan.0))))]
     [else
      (let* ([c (string-ref s 0)]
             [sign (case c [(#\-) -1.0] [else 1.0])]
             [start (if (memv c '(#\+ #\-)) 1 0)])
        (let-values ([(end v) (scan-decimal s start)])
          (if (and end (= end n))
              (fl* sign v)
              +nan.0)))])))
