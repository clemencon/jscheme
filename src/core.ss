;;; core.ss -- JavaScript values, objects, properties and type conversions.
;;;
;;; Value representation:
;;;   undefined  -> (void)            null    -> '()
;;;   booleans   -> #t / #f           numbers -> flonums (always)
;;;   strings    -> Scheme strings of UTF-16 code units; a surrogate code unit
;;;                 u (D800-DFFF) is stored as the character u + #x102000,
;;;                 which lies in the otherwise unused range 10F800-10FFFF.
;;;   objects    -> jsobj records (and subtypes).

(define js-undefined (void))
(define js-null '())

(define (undefined? v) (eq? v js-undefined))

;;; ---------------------------------------------------------------------
;;; Unique markers

(define-record-type marker (fields name) (nongenerative) (sealed #t))
(define c-empty (make-marker 'empty))       ; normal completion, no value
(define c-brk (make-marker 'break))
(define c-cnt (make-marker 'continue))
(define c-ret (make-marker 'return))
(define absent (make-marker 'absent))       ; missing property / array hole
(define lazy-args (make-marker 'lazy-args)) ; arguments object not built yet

;; Value carried by break/continue ("after value V") or by return.
(define *cval* c-empty)

;;; ---------------------------------------------------------------------
;;; Errors

(define-record-type js-error (fields kind message eof?) (nongenerative) (sealed #t))

(define (throw-runtime fmt . args)
  (raise (make-js-error 'runtime (apply format fmt args) #f)))

(define (throw-syntax msg eof?)
  (raise (make-js-error 'syntax msg eof?)))

;;; ---------------------------------------------------------------------
;;; Code units

(define (unit->char u)
  (if (and (fx>= u #xD800) (fx<= u #xDFFF))
      (integer->char (fx+ u #x102000))
      (integer->char u)))

(define (char->unit c)
  (let ([n (char->integer c)])
    (if (fx>= n #x10F800) (fx- n #x102000) n)))

;; Expand characters above FFFF into surrogate pairs (mapped).
(define (string->units s)
  (let ([n (string-length s)])
    (let check ([i 0])
      (cond
        [(fx= i n) s]
        [(fx> (char->integer (string-ref s i)) #xFFFF)
         (let-values ([(p get) (open-string-output-port)])
           (do ([j 0 (fx+ j 1)]) ((fx= j n))
             (let ([c (char->integer (string-ref s j))])
               (if (fx> c #xFFFF)
                   (let ([v (fx- c #x10000)])
                     (write-char (unit->char (fx+ #xD800 (fxsrl v 10))) p)
                     (write-char (unit->char (fx+ #xDC00 (fxand v #x3FF))) p))
                   (write-char (string-ref s j) p))))
           (get))]
        [else (check (fx+ i 1))]))))

;; Convert a JS string to proper Unicode for output: surrogate pairs are
;; combined, lone surrogates become U+FFFD.
(define (units->output s)
  (let ([n (string-length s)])
    (let check ([i 0])
      (cond
        [(fx= i n) s]
        [(fx>= (char->integer (string-ref s i)) #x10F800)
         (let-values ([(p get) (open-string-output-port)])
           (let loop ([j 0])
             (when (fx< j n)
               (let ([u (char->unit (string-ref s j))])
                 (cond
                   [(and (fx>= u #xD800) (fx<= u #xDBFF) (fx< (fx+ j 1) n)
                         (let ([u2 (char->unit (string-ref s (fx+ j 1)))])
                           (and (fx>= u2 #xDC00) (fx<= u2 #xDFFF))))
                    (let ([u2 (char->unit (string-ref s (fx+ j 1)))])
                      (write-char (integer->char
                                   (fx+ #x10000 (fx+ (fxsll (fx- u #xD800) 10) (fx- u2 #xDC00))))
                                  p)
                      (loop (fx+ j 2)))]
                   [(and (fx>= u #xD800) (fx<= u #xDFFF))
                    (write-char #\xFFFD p) (loop (fx+ j 1))]
                   [else (write-char (string-ref s j) p) (loop (fx+ j 1))]))))
           (get))]
        [else (check (fx+ i 1))]))))

;; Compare two JS strings by code units: returns -1, 0 or 1.
(define (js-string-compare a b)
  (let ([na (string-length a)] [nb (string-length b)])
    (let loop ([i 0])
      (cond
        [(fx= i na) (if (fx= i nb) 0 -1)]
        [(fx= i nb) 1]
        [else
         (let ([ca (string-ref a i)] [cb (string-ref b i)])
           (if (char=? ca cb)
               (loop (fx+ i 1))
               (if (fx< (char->unit ca) (char->unit cb)) -1 1)))]))))

;;; ---------------------------------------------------------------------
;;; Properties

(define attr-ro 1)
(define attr-de 2)
(define attr-dd 4)
(define attr-builtin 2)                 ; DontEnum only
(define attr-fixed 7)                   ; ReadOnly DontEnum DontDelete

(define-record-type xprop
  (fields (mutable value) (mutable attrs) order)
  (nongenerative) (sealed #t))

(define prop-counter 0)
(define (new-xprop v attrs)
  (set! prop-counter (fx+ prop-counter 1))
  (make-xprop v attrs prop-counter))

;;; ---------------------------------------------------------------------
;;; Objects
;;;
;;; kind: 0 plain (including native functions), 1 array, 2 user function,
;;;       3 arguments object.

(define-record-type jsobj
  (fields (mutable proto) (mutable class) (mutable props) (mutable value) kind)
  (nongenerative))

(define-record-type jsarray
  (parent jsobj)
  (fields (mutable vec) (mutable len))
  (nongenerative) (sealed #t))

;; call: (lambda (this argv) ...)
;; ctor: #f (no [[Construct]]), #t (generic construct) or (lambda (argv) ...)
(define-record-type jsfunc
  (parent jsobj)
  (fields (mutable call) (mutable ctor) name (mutable act) src)
  (nongenerative) (sealed #t))

(define-record-type jsargs
  (parent jsobj)
  (fields frame map)
  (nongenerative) (sealed #t))

(define (new-object proto class)
  (make-jsobj proto class #f js-undefined 0))

;; Key canonicalisation: an array index (0 .. 2^32-2) is a fixnum, any other
;; property name is a symbol.

(define (key->string k)
  (if (fixnum? k) (number->string k) (symbol->string k)))

(define (index-string? s)
  (let ([n (string-length s)])
    (and (fx> n 0) (fx<= n 10)
         (let ([c0 (string-ref s 0)])
           (and (char<=? #\0 c0 #\9)
                (or (fx= n 1) (not (char=? c0 #\0)))))
         (let loop ([i 1])
           (or (fx= i n)
               (and (char<=? #\0 (string-ref s i) #\9) (loop (fx+ i 1))))))))

(define (string->key s)
  (if (index-string? s)
      (let ([v (string->number s)])
        (if (< v 4294967295) v (string->symbol s)))
      (string->symbol s)))

(define (number->key x)
  (if (and (fl>= x 0.0) (fl< x 4294967295.0) (fl= x (flfloor x)))
      (flonum->fixnum x)
      (string->symbol (js-number->string x))))

(define (to-key v)
  (cond
    [(flonum? v) (number->key v)]
    [(string? v) (string->key v)]
    [else (string->key (to-string v))]))

;;; Own property access

(define (plain-own-prop o key)
  (let ([t (jsobj-props o)])
    (and t (eq-hashtable-ref t key #f))))

(define (plain-get-own o key)
  (let ([t (jsobj-props o)])
    (if t
        (let ([p (eq-hashtable-ref t key #f)])
          (if p (xprop-value p) absent))
        absent)))

(define (get-own o key)
  (if (fx= (jsobj-kind o) 0)
      (plain-get-own o key)
      (special-get-own o key)))

(define (special-get-own o key)
  (case (jsobj-kind o)
    [(1)
     (cond
       [(fixnum? key)
        (let ([v (jsarray-vec o)])
          (if (fx< key (vector-length v))
              (vector-ref v key)
              (plain-get-own o key)))]
       [(eq? key 'length) (fixnum->flonum (jsarray-len o))]
       [else (plain-get-own o key)])]
    [(2)
     (if (eq? key 'arguments)
         (let ([a (jsfunc-act o)])
           (if a (materialize-arguments a) js-null))
         (plain-get-own o key))]
    [(3)
     (let ([i (and (fixnum? key) (args-mapped-slot o key))])
       (if i
           (vector-ref (jsargs-frame o) i)
           (plain-get-own o key)))]
    [else (plain-get-own o key)]))

(define (args-mapped-slot o k)
  (let ([m (jsargs-map o)])
    (and (fx< k (vector-length m)) (vector-ref m k))))

;; Attributes of an own property, or #f when there is none.
(define (own-attrs o key)
  (case (jsobj-kind o)
    [(1)
     (cond
       [(fixnum? key)
        (let ([v (jsarray-vec o)])
          (if (fx< key (vector-length v))
              (and (not (eq? (vector-ref v key) absent)) 0)
              (let ([p (plain-own-prop o key)]) (and p (xprop-attrs p)))))]
       [(eq? key 'length) (fxior attr-de attr-dd)]
       [else (let ([p (plain-own-prop o key)]) (and p (xprop-attrs p)))])]
    [(2)
     (if (eq? key 'arguments)
         attr-fixed
         (let ([p (plain-own-prop o key)]) (and p (xprop-attrs p))))]
    [else (let ([p (plain-own-prop o key)]) (and p (xprop-attrs p)))]))

(define (own-has? o key) (and (own-attrs o key) #t))

;;; [[Get]]

(define (obj-get* o key)                ; absent when not found
  (let loop ([o o])
    (if (jsobj? o)
        (let ([v (get-own o key)])
          (if (eq? v absent) (loop (jsobj-proto o)) v))
        absent)))

(define (obj-get o key)
  (let loop ([o o])
    (if (jsobj? o)
        (let ([v (get-own o key)])
          (if (eq? v absent) (loop (jsobj-proto o)) v))
        js-undefined)))

;;; [[HasProperty]]

(define (obj-has? o key)
  (let loop ([o o])
    (and (jsobj? o)
         (or (own-has? o key) (loop (jsobj-proto o))))))

;;; [[CanPut]] along the prototype chain

(define (proto-can-put? o key)
  (let loop ([o o])
    (if (jsobj? o)
        (let ([a (own-attrs o key)])
          (if a
              (fx= 0 (fxand a attr-ro))
              (loop (jsobj-proto o))))
        #t)))

;;; [[Put]]

(define (ensure-props o)
  (or (jsobj-props o)
      (let ([t (make-eq-hashtable)])
        (jsobj-props-set! o t)
        t)))

(define (add-own! o key v attrs)
  (eq-hashtable-set! (ensure-props o) key (new-xprop v attrs)))

;; Create or replace an own property with the given attributes.
(define (define-own! o key v attrs)
  (if (and (jsarray? o) (fixnum? key))
      (array-put! o key v)
      (let ([p (plain-own-prop o key)])
        (if p
            (begin (xprop-value-set! p v) (xprop-attrs-set! p attrs))
            (add-own! o key v attrs)))))

(define (plain-put! o key v)
  (let ([p (plain-own-prop o key)])
    (if p
        (when (fx= 0 (fxand (xprop-attrs p) attr-ro))
          (xprop-value-set! p v))
        (when (proto-can-put? (jsobj-proto o) key)
          (add-own! o key v 0)))))

(define (obj-put! o key v)
  (case (jsobj-kind o)
    [(0) (plain-put! o key v)]
    [(1) (array-put! o key v)]
    [(2) (unless (eq? key 'arguments) (plain-put! o key v))]
    [(3)
     (let ([i (and (fixnum? key) (args-mapped-slot o key))])
       (if i
           (vector-set! (jsargs-frame o) i v)
           (plain-put! o key v)))]
    [else (plain-put! o key v)]))

;;; [[Delete]]

(define (plain-delete! o key)
  (let ([p (plain-own-prop o key)])
    (cond
      [(not p) #t]
      [(fx= 0 (fxand (xprop-attrs p) attr-dd))
       (eq-hashtable-delete! (jsobj-props o) key)
       #t]
      [else #f])))

(define (obj-delete! o key)
  (case (jsobj-kind o)
    [(1)
     (cond
       [(fixnum? key)
        (let ([v (jsarray-vec o)])
          (if (fx< key (vector-length v))
              (begin (vector-set! v key absent) #t)
              (plain-delete! o key)))]
       [(eq? key 'length) #f]
       [else (plain-delete! o key)])]
    [(2) (if (eq? key 'arguments) #f (plain-delete! o key))]
    [(3)
     (let ([r (plain-delete! o key)])
       (when (and r (fixnum? key))
         (let ([m (jsargs-map o)])
           (when (fx< key (vector-length m)) (vector-set! m key #f))))
       r)]
    [else (plain-delete! o key)]))

;;; Arrays

(define array-sparse-limit 16)

(define (array-grow! a key)
  ;; make the vector big enough to hold index key
  (let* ([old (jsarray-vec a)]
         [cap (vector-length old)]
         [ncap (fxmax (fx+ key 1) (fx* 2 cap) 8)]
         [new (make-vector ncap absent)])
    (do ([i 0 (fx+ i 1)]) ((fx= i cap)) (vector-set! new i (vector-ref old i)))
    (jsarray-vec-set! a new)
    ;; migrate sparse elements that now fall inside the vector
    (let ([t (jsobj-props a)])
      (when t
        (let-values ([(keys vals) (hashtable-entries t)])
          (vector-for-each
           (lambda (k p)
             (when (and (fixnum? k) (fx< k ncap))
               (vector-set! new k (xprop-value p))
               (eq-hashtable-delete! t k)))
           keys vals))))))

(define (array-put! a key v)
  (cond
    [(fixnum? key)
     (let ([vec (jsarray-vec a)])
       (cond
         [(fx< key (vector-length vec)) (vector-set! vec key v)]
         [(fx< key (fx+ (fx* 2 (vector-length vec)) array-sparse-limit))
          (array-grow! a key)
          (vector-set! (jsarray-vec a) key v)]
         [else
          (let ([p (plain-own-prop a key)])
            (if p (xprop-value-set! p v) (add-own! a key v 0)))]))
     (when (fx>= key (jsarray-len a))
       (jsarray-len-set! a (fx+ key 1)))]
    [(eq? key 'length) (array-set-length! a (to-uint32 v))]
    [else (plain-put! a key v)]))

(define (array-set-length! a newlen)
  (let ([old (jsarray-len a)])
    (when (< newlen old)
      (let* ([vec (jsarray-vec a)] [cap (vector-length vec)])
        (do ([i newlen (fx+ i 1)]) ((fx>= i (fxmin cap old)))
          (vector-set! vec i absent))
        (let ([t (jsobj-props a)])
          (when t
            (vector-for-each
             (lambda (k) (when (and (fixnum? k) (fx>= k newlen)) (eq-hashtable-delete! t k)))
             (hashtable-keys t))))))
    (jsarray-len-set! a newlen)))

(define (new-array proto n)
  (make-jsarray proto 'Array #f js-undefined 1
                (make-vector (fxmin n 100000) absent) n))

;;; Enumeration (for-in)

(define (own-keys-ordered o)
  ;; all own keys as a list of (key . enumerable?) in a stable order:
  ;; array elements, then index keys ascending, then others by creation.
  (let* ([t (jsobj-props o)]
         [entries
          (if t
              (let-values ([(keys vals) (hashtable-entries t)])
                (let loop ([i (fx- (vector-length keys) 1)] [acc '()])
                  (if (fx< i -1) acc
                      (if (fx= i -1) acc
                          (loop (fx- i 1)
                                (cons (cons (vector-ref keys i) (vector-ref vals i)) acc))))))
              '())]
         [sorted (sort (lambda (a b)
                         (let ([ka (car a)] [kb (car b)])
                           (cond
                             [(and (fixnum? ka) (fixnum? kb)) (fx< ka kb)]
                             [(fixnum? ka) #t]
                             [(fixnum? kb) #f]
                             [else (fx< (xprop-order (cdr a)) (xprop-order (cdr b)))])))
                       entries)]
         [named (map (lambda (e) (cons (car e) (fx= 0 (fxand (xprop-attrs (cdr e)) attr-de))))
                     sorted)])
    (case (jsobj-kind o)
      [(1)
       (let ([vec (jsarray-vec o)])
         (let loop ([i (fx- (vector-length vec) 1)] [acc (cons (cons 'length #f) named)])
           (if (fx< i 0) acc
               (loop (fx- i 1)
                     (if (eq? (vector-ref vec i) absent) acc (cons (cons i #t) acc))))))]
      [(2) (cons (cons 'arguments #f) named)]
      [else named])))

(define (enumerable-keys o)
  (let ([seen (make-eq-hashtable)])
    (let loop ([o o] [acc '()])
      (if (jsobj? o)
          (loop (jsobj-proto o)
                (fold-left (lambda (acc e)
                             (if (eq-hashtable-contains? seen (car e))
                                 acc
                                 (begin
                                   (eq-hashtable-set! seen (car e) #t)
                                   (if (cdr e) (cons (car e) acc) acc))))
                           acc (own-keys-ordered o)))
          (reverse acc)))))

;;; ---------------------------------------------------------------------
;;; Type conversion

(define (js-typeof v)
  (cond
    [(undefined? v) "undefined"]
    [(null? v) "object"]
    [(boolean? v) "boolean"]
    [(flonum? v) "number"]
    [(string? v) "string"]
    [(jsfunc? v) "function"]
    [else "object"]))

(define (to-boolean v)
  (cond
    [(eq? v #t) #t]
    [(eq? v #f) #f]
    [(flonum? v) (not (or (fl= v 0.0) (flnan? v)))]
    [(string? v) (fx> (string-length v) 0)]
    [(jsobj? v) #t]
    [else #f]))

(define (to-number v)
  (cond
    [(flonum? v) v]
    [(string? v) (string->js-number v)]
    [(eq? v #t) 1.0]
    [(eq? v #f) 0.0]
    [(undefined? v) +nan.0]
    [(null? v) 0.0]
    [else (to-number (to-primitive v 'number))]))

(define (to-primitive v hint)
  (if (jsobj? v) (default-value v hint) v))

(define (try-method o name)
  (let ([f (obj-get o name)])
    (if (jsobj? f)
        (let ([r (js-call f o '#())])
          (if (jsobj? r) absent r))
        absent)))

(define (default-value o hint)
  (let ([hint (or hint (if (eq? (jsobj-class o) 'Date) 'string 'number))])
    (let* ([first (if (eq? hint 'string) 'toString 'valueOf)]
           [second (if (eq? hint 'string) 'valueOf 'toString)]
           [r (try-method o first)])
      (if (eq? r absent)
          (let ([r (try-method o second)])
            (if (eq? r absent)
                (throw-runtime "TypeError: cannot convert object to primitive value")
                r))
          r))))

(define (to-string v)
  (cond
    [(string? v) v]
    [(flonum? v) (js-number->string v)]
    [(eq? v #t) "true"]
    [(eq? v #f) "false"]
    [(undefined? v) "undefined"]
    [(null? v) "null"]
    [else (to-string (to-primitive v 'string))]))

(define (to-integer v)
  (let ([x (to-number v)])
    (cond
      [(flnan? x) 0.0]
      [(flinfinite? x) x]
      [else (fltruncate x)])))

(define two32 4294967296.0)
(define two31 2147483648.0)

(define (fl->uint32 x)
  (cond
    [(and (fl>= x 0.0) (fl< x two32)) (flonum->fixnum x)]
    [(or (flnan? x) (flinfinite? x)) 0]
    [(and (fl> x -9.0e15) (fl< x 9.0e15))
     (let ([i (flonum->fixnum (fltruncate x))])
       (fxmodulo i 4294967296))]
    [else (modulo (exact (fltruncate x)) 4294967296)]))

(define (fl->int32 x)
  (if (and (fl> x -2147483649.0) (fl< x two31))
      (flonum->fixnum x)
      (let ([u (fl->uint32 x)])
        (if (fx>= u 2147483648) (fx- u 4294967296) u))))

(define (to-int32 v) (fl->int32 (if (flonum? v) v (to-number v))))
(define (to-uint32 v) (fl->uint32 (if (flonum? v) v (to-number v))))
(define (to-uint16 v) (fxand (to-uint32 v) #xFFFF))

(define (check-coercible v)
  (when (or (undefined? v) (null? v))
    (throw-runtime "TypeError: ~a has no properties" (if (null? v) "null" "undefined"))))

(define (to-object v)
  (cond
    [(jsobj? v) v]
    [(string? v) (make-string-object v)]
    [(flonum? v) (make-wrapper number-prototype 'Number v)]
    [(boolean? v) (make-wrapper boolean-prototype 'Boolean v)]
    [else (throw-runtime "TypeError: cannot convert ~a to an object"
                         (if (null? v) "null" "undefined"))]))

(define (make-wrapper proto class v)
  (let ([o (new-object proto class)])
    (jsobj-value-set! o v)
    o))

(define (make-string-object s)
  (let ([o (make-wrapper string-prototype 'String s)])
    (add-own! o 'length (fixnum->flonum (string-length s)) attr-fixed)
    o))

;; [[Get]] on any value (base already checked to be coercible).
(define (prim-get b key)
  (cond
    [(string? b)
     (if (eq? key 'length)
         (fixnum->flonum (string-length b))
         (obj-get string-prototype key))]
    [(flonum? b) (obj-get number-prototype key)]
    [(boolean? b) (obj-get boolean-prototype key)]
    [else (check-coercible b) js-undefined]))

(define (value-get b key)
  (if (jsobj? b) (obj-get b key) (prim-get b key)))

;;; ---------------------------------------------------------------------
;;; Calling

(define (js-call f this argv)
  (if (jsfunc? f)
      ((jsfunc-call f) this argv)
      (throw-runtime "TypeError: ~a is not a function" (describe-value f))))

(define (js-construct f argv)
  (if (jsfunc? f)
      (let ([c (jsfunc-ctor f)])
        (cond
          [(procedure? c) (c argv)]
          [(eq? c #t)
           (let* ([p (obj-get f 'prototype)]
                  [o (new-object (if (jsobj? p) p object-prototype) 'Object)]
                  [r ((jsfunc-call f) o argv)])
             (if (jsobj? r) r o))]
          [else (throw-runtime "TypeError: ~a is not a constructor" (describe-value f))]))
      (throw-runtime "TypeError: ~a is not a constructor" (describe-value f))))

(define (describe-value v)
  (cond
    [(jsfunc? v) (format "function ~a" (or (jsfunc-name v) ""))]
    [(jsobj? v) "object"]
    [(string? v) (format "~s" (units->output v))]
    [else (to-string v)]))
