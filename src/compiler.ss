;;; compiler.ss -- turns the AST into Scheme closures (chapters 10 to 14).
;;;
;;; An expression compiles to (lambda (env) value).  A statement compiles to
;;; (lambda (env) completion) where the completion is a JS value (normal
;;; completion after that value), c-empty (normal completion), or one of
;;; c-brk / c-cnt / c-ret, with the carried value in *cval*.

(define-record-type jsenv (fields act withs this) (nongenerative) (sealed #t))

;; Static information about a function: slot layout and compiled body.
(define-record-type finfo
  (fields name params nparams slots argslot nslots (mutable body) src)
  (nongenerative) (sealed #t))

(define (finfo-slot info name)
  (and info (eq-hashtable-ref (finfo-slots info) name #f)))

;; A function activation.  The activation object of the spec is the frame
;; (fixed slots) plus `extras', a table of variables added by eval code.
(define-record-type jsact
  (fields frame func argv info (mutable extras) (mutable argsobj))
  (nongenerative) (sealed #t))

;; Nesting depth of user function calls.  The spec sets no limit (10.2.3);
;; this bound only turns runaway recursion into a runtime error instead of
;; exhausting memory (a million frames take well under 300 MB).
(define max-call-depth 1000000)
(define *depth* 0)

;; The environment of the innermost running code (for eval).
(define *cur-env* #f)
(define the-global #f)
(define global-env #f)

(define (throw-reference name)
  (throw-runtime "ReferenceError: ~a is not defined" name))

;;; ------------------------------------------------------------------
;;; Arguments object (10.1.8)

(define (materialize-arguments act)
  (or (jsact-argsobj act)
      (let* ([argv (jsact-argv act)]
             [na (vector-length argv)]
             [info (jsact-info act)]
             [np (finfo-nparams info)]
             [nm (fxmin na np)]
             [m (make-vector nm #f)]
             [o (make-jsargs object-prototype 'Object #f js-undefined 3 (jsact-frame act) m)])
        (do ([i 0 (fx+ i 1)]) ((fx= i nm)) (vector-set! m i i))
        (add-own! o 'callee (jsact-func act) attr-de)
        (add-own! o 'length (fixnum->flonum na) attr-de)
        (do ([i 0 (fx+ i 1)]) ((fx= i na)) (add-own! o i (vector-ref argv i) attr-de))
        (jsact-argsobj-set! act o)
        (let ([fr (jsact-frame act)] [s (finfo-argslot info)])
          (when (eq? (vector-ref fr s) lazy-args) (vector-set! fr s o)))
        o)))

;;; ------------------------------------------------------------------
;;; References
;;;
;;; kinds: 0 property of base (an object or a primitive), key is a key
;;;        1 frame slot, base is the frame vector, key the slot index
;;;        2 eval-created variable, base is the extras table, key the name
;;;        3 unresolved (null base), key the name

(define (resolve-id env name slot)
  (let loop ([ws (jsenv-withs env)])
    (if (pair? ws)
        (if (obj-has? (car ws) name) (values 0 (car ws)) (loop (cdr ws)))
        (let ([act (jsenv-act env)])
          (cond
            [(and act slot) (values 1 (jsact-frame act))]
            [(and act (jsact-extras act) (eq-hashtable-contains? (jsact-extras act) name))
             (values 2 (jsact-extras act))]
            [(obj-has? the-global name) (values 0 the-global)]
            [else (values 3 #f)])))))

(define (ref-get env kind base key)
  (case kind
    [(0) (value-get base key)]
    [(1) (let ([v (vector-ref base key)])
           (if (eq? v lazy-args) (materialize-arguments (jsenv-act env)) v))]
    [(2) (eq-hashtable-ref base key js-undefined)]
    [else (throw-reference key)]))

(define (ref-put! kind base key v)
  (case kind
    [(0) (when (jsobj? base) (obj-put! base key v))]
    [(1) (vector-set! base key v)]
    [(2) (eq-hashtable-set! base key v)]
    [else (obj-put! the-global key v)]))

(define (ref-delete! kind base key)
  (case kind
    [(0) (if (jsobj? base) (obj-delete! base key) #t)]
    [(1) #f]
    [(2) (eq-hashtable-delete! base key) #t]
    [else #t]))

;; compile a reference: returns (lambda (env) (values kind base key))
(define (compile-ref node info)
  (case (car node)
    [(group) (compile-ref (cadr node) info)]
    [(id)
     (let* ([name (cadr node)] [slot (finfo-slot info name)])
       (lambda (env)
         (let-values ([(kind base) (resolve-id env name slot)])
           (values kind base (if (fx= kind 1) slot name)))))]
    [(dot)
     (let ([b (compile-expr (cadr node) info)] [key (caddr node)])
       (lambda (env)
         (let ([bv (b env)])
           (check-coercible bv)
           (values 0 bv key))))]
    [(index)
     (let ([b (compile-expr (cadr node) info)] [k (compile-expr (caddr node) info)])
       (lambda (env)
         (let* ([bv (b env)] [kv (k env)])
           (check-coercible bv)
           (values 0 bv (to-key kv)))))]
    [else
     ;; not a reference: evaluate, then fail when the reference is used
     (let ([e (compile-expr node info)])
       (lambda (env)
         (e env)
         (throw-runtime "ReferenceError: invalid assignment target")))]))

;;; ------------------------------------------------------------------
;;; Operators

(define (js-add a b)
  (if (and (flonum? a) (flonum? b))
      (fl+ a b)
      (let* ([a (to-primitive a #f)] [b (to-primitive b #f)])
        (if (or (string? a) (string? b))
            (string-append (to-string a) (to-string b))
            (fl+ (to-number a) (to-number b))))))

(define (num2 op)
  (lambda (a b)
    (if (and (flonum? a) (flonum? b))
        (op a b)
        (let* ([x (to-number a)] [y (to-number b)]) (op x y)))))

(define c-fmod #f)

(define (js-mod x y)
  (cond
    [(or (flnan? x) (flnan? y) (flinfinite? x) (fl= y 0.0)) +nan.0]
    [(flinfinite? y) x]
    [(fl= x 0.0) x]
    [else (let ([r (c-fmod x y)])
            (if (fl= r 0.0) (if (fl< x 0.0) -0.0 0.0) r))]))

(define (int32-wrap n)
  (let ([u (fxand n #xFFFFFFFF)])
    (if (fx>= u #x80000000) (fx- u #x100000000) u)))

(define (js-shl a b)
  (let* ([x (to-int32 a)] [s (fxand (to-uint32 b) 31)])
    (fixnum->flonum (int32-wrap (fxsll (fxand x (fx- (fxsll 1 (fx- 32 s)) 1)) s)))))

(define (js-sar a b)
  (let* ([x (to-int32 a)] [s (fxand (to-uint32 b) 31)])
    (fixnum->flonum (fxsra x s))))

(define (js-shr a b)
  (let* ([x (to-uint32 a)] [s (fxand (to-uint32 b) 31)])
    (fixnum->flonum (fxsrl x s))))

(define (bitop op)
  (lambda (a b)
    (let* ([x (to-int32 a)] [y (to-int32 b)])
      (fixnum->flonum (op x y)))))

;; Abstract relational comparison (11.8.5): #t, #f, or 'undef.
(define (js-less x y)
  (if (and (flonum? x) (flonum? y))
      (if (or (flnan? x) (flnan? y)) 'undef (fl< x y))
      (let* ([px (to-primitive x 'number)] [py (to-primitive y 'number)])
        (if (and (string? px) (string? py))
            (fx< (js-string-compare px py) 0)
            (let* ([nx (to-number px)] [ny (to-number py)])
              (if (or (flnan? nx) (flnan? ny)) 'undef (fl< nx ny)))))))

(define (js-lt a b) (eq? (js-less a b) #t))
(define (js-gt a b) (eq? (js-less b a) #t))
(define (js-le a b) (eq? (js-less b a) #f))
(define (js-ge a b) (eq? (js-less a b) #f))

(define (js-equal? x y)
  (cond
    [(and (flonum? x) (flonum? y)) (fl= x y)]
    [(and (string? x) (string? y)) (string=? x y)]
    [(or (undefined? x) (null? x)) (or (undefined? y) (null? y))]
    [(or (undefined? y) (null? y)) #f]
    [(and (boolean? x) (boolean? y)) (eq? x y)]
    [(and (jsobj? x) (jsobj? y)) (eq? x y)]
    [(and (flonum? x) (string? y)) (fl= x (to-number y))]
    [(and (string? x) (flonum? y)) (fl= (to-number x) y)]
    [(boolean? x) (js-equal? (if x 1.0 0.0) y)]
    [(boolean? y) (js-equal? x (if y 1.0 0.0))]
    [(jsobj? y) (js-equal? x (to-primitive y #f))]
    [(jsobj? x) (js-equal? (to-primitive x #f) y)]
    [else #f]))

(define binop-table
  (list
   (cons '+ js-add)
   (cons '- (num2 fl-))
   (cons '* (num2 fl*))
   (cons '/ (num2 fl/))
   (cons '% (num2 js-mod))
   (cons '<< js-shl)
   (cons '>> js-sar)
   (cons '>>> js-shr)
   (cons '& (bitop fxand))
   (cons p-bitor (bitop fxior))
   (cons '^ (bitop fxxor))
   (cons '< js-lt)
   (cons '> js-gt)
   (cons '<= js-le)
   (cons '>= js-ge)
   (cons '== js-equal?)
   (cons '!= (lambda (a b) (not (js-equal? a b))))))

(define (binop-proc op) (cdr (assq op binop-table)))

;; the operator of a compound assignment such as += or >>>=
(define (compound-op op)
  (let ([s (symbol->string op)])
    (binop-proc (string->symbol (substring s 0 (fx- (string-length s) 1))))))

;;; ------------------------------------------------------------------
;;; Expressions

(define (compile-args args info)
  (let ([cs (list->vector (map (lambda (a) (compile-expr a info)) args))])
    (case (vector-length cs)
      [(0) (lambda (env) '#())]
      [(1) (let ([a (vector-ref cs 0)]) (lambda (env) (vector (a env))))]
      [(2) (let ([a (vector-ref cs 0)] [b (vector-ref cs 1)])
             (lambda (env) (let* ([x (a env)] [y (b env)]) (vector x y))))]
      [else
       (let ([n (vector-length cs)])
         (lambda (env)
           (let ([v (make-vector n)])
             (do ([i 0 (fx+ i 1)]) ((fx= i n) v)
               (vector-set! v i ((vector-ref cs i) env))))))])))

(define (compile-id-get name info)
  (let ([slot (finfo-slot info name)])
    (if slot
        (lambda (env)
          (if (null? (jsenv-withs env))
              (let ([v (vector-ref (jsact-frame (jsenv-act env)) slot)])
                (if (eq? v lazy-args) (materialize-arguments (jsenv-act env)) v))
              (let-values ([(kind base) (resolve-id env name slot)])
                (ref-get env kind base (if (fx= kind 1) slot name)))))
        (lambda (env)
          (if (and (null? (jsenv-withs env))
                   (let ([a (jsenv-act env)]) (or (not a) (not (jsact-extras a)))))
              (let ([v (obj-get* the-global name)])
                (if (eq? v absent) (throw-reference name) v))
              (let-values ([(kind base) (resolve-id env name #f)])
                (ref-get env kind base name)))))))

(define (compile-member-get b k)
  (lambda (env)
    (let* ([bv (b env)] [kv (k env)])
      (if (jsobj? bv)
          (if (and (jsarray? bv) (flonum? kv))
              (let ([key (number->key kv)])
                (if (fixnum? key)
                    (let ([vec (jsarray-vec bv)])
                      (if (fx< key (vector-length vec))
                          (let ([v (vector-ref vec key)])
                            (if (eq? v absent) (obj-get bv key) v))
                          (obj-get bv key)))
                    (obj-get bv key)))
              (obj-get bv (to-key kv)))
          (begin (check-coercible bv) (prim-get bv (to-key kv)))))))

(define (call-this-of kind base)
  (if (fx= kind 0) base the-global))

(define (compile-call node info)
  (let ([f (cadr node)] [args (compile-args (caddr node) info)])
    (case (car (strip-group f))
      [(id)
       (let* ([name (cadr (strip-group f))] [slot (finfo-slot info name)])
         (lambda (env)
           (let-values ([(kind base) (resolve-id env name slot)])
             (let* ([key (if (fx= kind 1) slot name)]
                    [argv (args env)]
                    [fv (ref-get env kind base key)])
               (if (jsfunc? fv)
                   ((jsfunc-call fv) (if (fx= kind 0) base the-global) argv)
                   (throw-runtime "TypeError: ~a is not a function" name))))))]
      [(dot index)
       (let ([r (compile-ref f info)])
         (lambda (env)
           (let-values ([(kind base key) (r env)])
             (let* ([argv (args env)]
                    [fv (value-get base key)])
               (if (jsfunc? fv)
                   ((jsfunc-call fv) base argv)
                   (throw-runtime "TypeError: ~a is not a function" (key->string key)))))))]
      [else
       (let ([fe (compile-expr f info)])
         (lambda (env)
           (let* ([fv (fe env)] [argv (args env)])
             (if (jsfunc? fv)
                 ((jsfunc-call fv) the-global argv)
                 (throw-runtime "TypeError: ~a is not a function" (describe-value fv))))))])))

(define (strip-group n) (if (eq? (car n) 'group) (strip-group (cadr n)) n))

(define (compile-assign node info)
  (let ([op (cadr node)] [lhs (strip-group (caddr node))] [rhs (compile-expr (cadddr node) info)])
    (cond
      [(and (eq? op '=) (eq? (car lhs) 'id) (finfo-slot info (cadr lhs)))
       => (lambda (slot)
            (let ([name (cadr lhs)])
              (lambda (env)
                (if (null? (jsenv-withs env))
                    (let ([v (rhs env)])
                      (vector-set! (jsact-frame (jsenv-act env)) slot v)
                      v)
                    (let-values ([(kind base) (resolve-id env name slot)])
                      (let ([v (rhs env)])
                        (ref-put! kind base (if (fx= kind 1) slot name) v)
                        v))))))]
      [(and (eq? op '=) (eq? (car lhs) 'index))
       (let ([b (compile-expr (cadr lhs) info)] [k (compile-expr (caddr lhs) info)])
         (lambda (env)
           (let* ([bv (b env)] [kv (k env)])
             (check-coercible bv)
             (let* ([key (to-key kv)] [v (rhs env)])
               (when (jsobj? bv) (obj-put! bv key v))
               v))))]
      [(eq? op '=)
       (let ([r (compile-ref lhs info)])
         (lambda (env)
           (let-values ([(kind base key) (r env)])
             (let ([v (rhs env)])
               (ref-put! kind base key v)
               v))))]
      [else
       (let ([r (compile-ref lhs info)] [proc (compound-op op)])
         (lambda (env)
           (let-values ([(kind base key) (r env)])
             (let* ([old (ref-get env kind base key)]
                    [v (proc old (rhs env))])
               (ref-put! kind base key v)
               v))))])))

(define (compile-incdec node info prefix?)
  (let ([delta (if (eq? (cadr node) '++) 1.0 -1.0)]
        [r (compile-ref (caddr node) info)])
    (lambda (env)
      (let-values ([(kind base key) (r env)])
        (let* ([old (to-number (ref-get env kind base key))]
               [new (fl+ old delta)])
          (ref-put! kind base key new)
          (if prefix? new old))))))

(define (compile-unary node info)
  (let ([op (cadr node)] [arg (caddr node)])
    (case op
      [(++ --) (compile-incdec node info #t)]
      [(delete)
       (let ([a (strip-group arg)])
         (if (memq (car a) '(id dot index))
             (let ([r (compile-ref a info)])
               (lambda (env)
                 (let-values ([(kind base key) (r env)])
                   (ref-delete! kind base key))))
             (let ([e (compile-expr arg info)])
               (lambda (env)
                 (e env)
                 (throw-runtime "ReferenceError: delete of a value that is not a reference")))))]
      [(typeof)
       (let ([a (strip-group arg)])
         (if (eq? (car a) 'id)
             (let* ([name (cadr a)] [slot (finfo-slot info name)])
               (lambda (env)
                 (let-values ([(kind base) (resolve-id env name slot)])
                   (if (fx= kind 3)
                       "undefined"
                       (js-typeof (ref-get env kind base (if (fx= kind 1) slot name)))))))
             (let ([e (compile-expr arg info)])
               (lambda (env) (js-typeof (e env))))))]
      [else
       (let ([e (compile-expr arg info)])
         (case op
           [(void) (lambda (env) (e env) js-undefined)]
           [(+) (lambda (env) (to-number (e env)))]
           [(-) (lambda (env) (fl- (to-number (e env))))]
           [(~) (lambda (env) (fixnum->flonum (fxlognot (to-int32 (e env)))))]
           [(!) (lambda (env) (not (to-boolean (e env))))]
           [else (error 'compile "bad unary" op)]))])))

(define (compile-expr node info)
  (case (car node)
    [(num) (let ([v (cadr node)]) (lambda (env) v))]
    [(str) (let ([v (cadr node)]) (lambda (env) v))]
    [(null) (lambda (env) js-null)]
    [(bool) (let ([v (cadr node)]) (lambda (env) v))]
    [(this) (lambda (env) (jsenv-this env))]
    [(id) (compile-id-get (cadr node) info)]
    [(group) (compile-expr (cadr node) info)]
    [(dot)
     (let ([b (compile-expr (cadr node) info)] [key (caddr node)])
       (lambda (env)
         (let ([bv (b env)])
           (if (jsobj? bv) (obj-get bv key) (prim-get bv key)))))]
    [(index) (compile-member-get (compile-expr (cadr node) info) (compile-expr (caddr node) info))]
    [(call) (compile-call node info)]
    [(new)
     (let ([c (compile-expr (cadr node) info)] [args (compile-args (caddr node) info)])
       (lambda (env)
         (let* ([cv (c env)] [argv (args env)])
           (js-construct cv argv))))]
    [(postfix) (compile-incdec node info #f)]
    [(unary) (compile-unary node info)]
    [(binary)
     (let ([proc (binop-proc (cadr node))]
           [a (compile-expr (caddr node) info)]
           [b (compile-expr (cadddr node) info)])
       (case (cadr node)
         [(+) (lambda (env) (let* ([x (a env)] [y (b env)])
                              (if (and (flonum? x) (flonum? y)) (fl+ x y) (js-add x y))))]
         [(-) (lambda (env) (let* ([x (a env)] [y (b env)])
                              (if (and (flonum? x) (flonum? y)) (fl- x y) (proc x y))))]
         [(*) (lambda (env) (let* ([x (a env)] [y (b env)])
                              (if (and (flonum? x) (flonum? y)) (fl* x y) (proc x y))))]
         [(<) (lambda (env) (let* ([x (a env)] [y (b env)])
                              (if (and (flonum? x) (flonum? y)) (fl< x y) (proc x y))))]
         [(>) (lambda (env) (let* ([x (a env)] [y (b env)])
                              (if (and (flonum? x) (flonum? y)) (fl> x y) (proc x y))))]
         [(<=) (lambda (env) (let* ([x (a env)] [y (b env)])
                               (if (and (flonum? x) (flonum? y)) (fl<= x y) (proc x y))))]
         [(>=) (lambda (env) (let* ([x (a env)] [y (b env)])
                               (if (and (flonum? x) (flonum? y)) (fl>= x y) (proc x y))))]
         [else (lambda (env) (let* ([x (a env)] [y (b env)]) (proc x y)))]))]
    [(and)
     (let ([a (compile-expr (cadr node) info)] [b (compile-expr (caddr node) info)])
       (lambda (env) (let ([x (a env)]) (if (to-boolean x) (b env) x))))]
    [(or)
     (let ([a (compile-expr (cadr node) info)] [b (compile-expr (caddr node) info)])
       (lambda (env) (let ([x (a env)]) (if (to-boolean x) x (b env)))))]
    [(cond)
     (let ([c (compile-expr (cadr node) info)]
           [a (compile-expr (caddr node) info)]
           [b (compile-expr (cadddr node) info)])
       (lambda (env) (if (to-boolean (c env)) (a env) (b env))))]
    [(assign) (compile-assign node info)]
    [(comma)
     (let ([a (compile-expr (cadr node) info)] [b (compile-expr (caddr node) info)])
       (lambda (env) (a env) (b env)))]
    [else (error 'compile "unknown expression" node)]))

;;; ------------------------------------------------------------------
;;; Statements

(define (run-list v env)
  (let ([n (vector-length v)])
    (let loop ([i 0] [val c-empty])
      (if (fx= i n)
          val
          (let ([r ((vector-ref v i) env)])
            (cond
              [(eq? r c-empty) (loop (fx+ i 1) val)]
              [(marker? r)
               (when (eq? *cval* c-empty) (set! *cval* val))
               r]
              [else (loop (fx+ i 1) r)]))))))

(define (compile-stmt-list stmts info)
  (let ([cs (map (lambda (s) (compile-stmt s info)) stmts)])
    (cond
      [(null? cs) (lambda (env) c-empty)]
      [(null? (cdr cs)) (car cs)]
      [else (let ([v (list->vector cs)]) (lambda (env) (run-list v env)))])))

;; Run a loop body; returns (values action C) where action is
;; 'next, 'break, or 'return (C is then the completion to return).
(define-syntax loop-step
  (syntax-rules ()
    [(_ r C next brk ret)
     (cond
       [(eq? r c-empty) (next C)]
       [(not (marker? r)) (next r)]
       [(eq? r c-cnt) (next (if (eq? *cval* c-empty) C *cval*))]
       [(eq? r c-brk) (brk (if (eq? *cval* c-empty) C *cval*))]
       [else (ret r)])]))

(define (compile-var-decls decls info)
  (let ([cs (filter values
                    (map (lambda (d)
                           (and (cdr d)
                                (compile-assign (list 'assign '= (list 'id (car d)) (cdr d)) info)))
                         decls))])
    (lambda (env)
      (for-each (lambda (c) (c env)) cs)
      c-empty)))

(define (compile-for-in lhs-ref obj body)
  (lambda (env)
    (let* ([o (to-object (obj env))])
      (let loop ([keys (enumerable-keys o)] [C c-empty])
        (cond
          [(null? keys) C]
          [(not (obj-has? o (car keys))) (loop (cdr keys) C)]
          [else
           (let-values ([(kind base key) (lhs-ref env)])
             (ref-put! kind base key (key->string (car keys))))
           (let ([r (body env)])
             (loop-step r C (lambda (C) (loop (cdr keys) C)) values values))])))))

(define (compile-stmt node info)
  (case (car node)
    [(block) (compile-stmt-list (cadr node) info)]
    [(var) (compile-var-decls (cadr node) info)]
    [(empty) (lambda (env) c-empty)]
    [(expr) (compile-expr (cadr node) info)]
    [(if)
     (let ([c (compile-expr (cadr node) info)]
           [a (compile-stmt (caddr node) info)]
           [b (if (cadddr node) (compile-stmt (cadddr node) info) (lambda (env) c-empty))])
       (lambda (env) (if (to-boolean (c env)) (a env) (b env))))]
    [(while)
     (let ([c (compile-expr (cadr node) info)] [body (compile-stmt (caddr node) info)])
       (lambda (env)
         (let loop ([C c-empty])
           (if (to-boolean (c env))
               (let ([r (body env)])
                 (loop-step r C loop values values))
               C))))]
    [(for)
     (let* ([init (if (cadr node) (compile-stmt (cadr node) info) (lambda (env) c-empty))]
            [test (if (caddr node) (compile-expr (caddr node) info) (lambda (env) #t))]
            [update (if (cadddr node) (compile-expr (cadddr node) info) (lambda (env) #f))]
            [body (compile-stmt (list-ref node 4) info)])
       (lambda (env)
         (init env)
         (let loop ([C c-empty])
           (if (to-boolean (test env))
               (let ([r (body env)])
                 (loop-step r C (lambda (C) (update env) (loop C)) values values))
               C))))]
    [(forin)
     (compile-for-in (compile-ref (cadr node) info)
                     (compile-expr (caddr node) info)
                     (compile-stmt (cadddr node) info))]
    [(forin-var)
     (let* ([name (cadr node)]
            [init (and (caddr node)
                       (compile-assign (list 'assign '= (list 'id name) (caddr node)) info))]
            [loop (compile-for-in (compile-ref (list 'id name) info)
                                  (compile-expr (cadddr node) info)
                                  (compile-stmt (list-ref node 4) info))])
       (if init
           (lambda (env) (init env) (loop env))
           loop))]
    [(continue) (lambda (env) (set! *cval* c-empty) c-cnt)]
    [(break) (lambda (env) (set! *cval* c-empty) c-brk)]
    [(return)
     (if (cadr node)
         (let ([e (compile-expr (cadr node) info)])
           (lambda (env) (set! *cval* (e env)) c-ret))
         (lambda (env) (set! *cval* js-undefined) c-ret))]
    [(with)
     (let ([e (compile-expr (cadr node) info)] [body (compile-stmt (caddr node) info)])
       (lambda (env)
         (let* ([o (to-object (e env))]
                [nenv (make-jsenv (jsenv-act env) (cons o (jsenv-withs env)) (jsenv-this env))]
                [saved *cur-env*])
           (set! *cur-env* nenv)
           (let ([r (body nenv)])
             (set! *cur-env* saved)
             r))))]
    [else (error 'compile "unknown statement" node)]))

;;; ------------------------------------------------------------------
;;; Functions

;; Names declared by var statements, in source order, without duplicates.
(define (collect-vars stmts)
  (let ([acc '()])
    (define (add! n) (unless (memq n acc) (set! acc (cons n acc))))
    (define (walk s)
      (case (car s)
        [(block) (for-each walk (cadr s))]
        [(var) (for-each (lambda (d) (add! (car d))) (cadr s))]
        [(if) (walk (caddr s)) (when (cadddr s) (walk (cadddr s)))]
        [(while) (walk (caddr s))]
        [(for) (when (cadr s) (walk (cadr s))) (walk (list-ref s 4))]
        [(forin) (walk (cadddr s))]
        [(forin-var) (add! (cadr s)) (walk (list-ref s 4))]
        [(with) (walk (caddr s))]
        [else (void)]))
    (for-each walk stmts)
    (reverse acc)))

(define (make-function-info name params body-stmts src)
  (let* ([slots (make-eq-hashtable)]
         [np (length params)])
    (let loop ([ps params] [i 0])
      (unless (null? ps)
        (eq-hashtable-set! slots (car ps) i)
        (loop (cdr ps) (fx+ i 1))))
    (unless (eq-hashtable-contains? slots 'arguments)
      (eq-hashtable-set! slots 'arguments np))
    (let ([n (let loop ([vs (collect-vars body-stmts)] [n (fx+ np 1)])
               (cond
                 [(null? vs) n]
                 [(eq-hashtable-contains? slots (car vs)) (loop (cdr vs) n)]
                 [else (eq-hashtable-set! slots (car vs) n) (loop (cdr vs) (fx+ n 1))]))])
      (let ([info (make-finfo name params np slots np n #f src)])
        (finfo-body-set! info (compile-stmt-list body-stmts info))
        info))))

(define (make-user-function info)
  (let* ([f (make-jsfunc function-prototype 'Function #f js-undefined 2
                         #f #t (symbol->string (finfo-name info)) #f (finfo-src info))]
         [proto (new-object object-prototype 'Object)])
    (jsfunc-call-set! f (make-user-call f info))
    (add-own! f 'length (fixnum->flonum (finfo-nparams info)) attr-fixed)
    (add-own! proto 'constructor f attr-de)
    (add-own! f 'prototype proto attr-de)
    f))

(define (make-user-call f info)
  (let ([np (finfo-nparams info)]
        [nslots (finfo-nslots info)]
        [argslot (finfo-argslot info)]
        [body (finfo-body info)])
    (lambda (this argv)
      (let ([frame (make-vector nslots js-undefined)]
            [n (fxmin (vector-length argv) np)])
        (do ([i 0 (fx+ i 1)]) ((fx= i n)) (vector-set! frame i (vector-ref argv i)))
        (vector-set! frame argslot lazy-args)
        (let* ([act (make-jsact frame f argv info #f #f)]
               [this (cond
                       [(jsobj? this) this]
                       [(or (undefined? this) (null? this)) the-global]
                       [else (to-object this)])]
               [env (make-jsenv act '() this)]
               [saved-env *cur-env*]
               [saved-act (jsfunc-act f)])
          (when (fx>= *depth* max-call-depth)
            (throw-runtime "RangeError: maximum call depth (~a) exceeded" max-call-depth))
          (set! *depth* (fx+ *depth* 1))
          (jsfunc-act-set! f act)
          (set! *cur-env* env)
          (let ([r (body env)])
            (set! *cur-env* saved-env)
            (jsfunc-act-set! f saved-act)
            (set! *depth* (fx- *depth* 1))
            (if (eq? r c-ret) *cval* js-undefined)))))))

;;; ------------------------------------------------------------------
;;; Programs (global and eval code)

;; Compile a Program for execution in an environment whose function info
;; is `info' (#f for global code).  Returns (lambda (env) completion).
(define (compile-js-program ast info)
  (let* ([fdecls (filter (lambda (s) (eq? (car s) 'function)) ast)]
         [stmts (filter (lambda (s) (not (eq? (car s) 'function))) ast)]
         [finfos (map (lambda (d) (make-function-info (cadr d) (caddr d) (cadddr d) (list-ref d 4)))
                      fdecls)]
         [vars (collect-vars stmts)]
         [body (compile-stmt-list stmts info)])
    (lambda (env)
      (let ([act (jsenv-act env)])
        (for-each
         (lambda (fi)
           (let ([f (make-user-function fi)] [name (finfo-name fi)])
             (cond
               [(not act) (define-own! the-global name f 0)]
               [(finfo-slot (jsact-info act) name)
                => (lambda (slot) (vector-set! (jsact-frame act) slot f))]
               [else (eq-hashtable-set! (act-extras! act) name f)])))
         finfos)
        (for-each
         (lambda (name)
           (cond
             [(not act)
              (unless (own-has? the-global name) (add-own! the-global name js-undefined 0))]
             [(finfo-slot (jsact-info act) name) (void)]
             [else
              (let ([t (act-extras! act)])
                (unless (eq-hashtable-contains? t name)
                  (eq-hashtable-set! t name js-undefined)))]))
         vars))
      (body env))))

(define (act-extras! act)
  (or (jsact-extras act)
      (let ([t (make-eq-hashtable)])
        (jsact-extras-set! act t)
        t)))

;; Parse source text as a Program; syntax errors stay syntax errors.
(define (js-compile-source src info)
  (compile-js-program (ps-parse (string->units src) 'program) info))

;; eval code (15.1.2.1): a parse failure is a runtime error.
(define (js-eval-string src)
  (let* ([env (or *cur-env* global-env)]
         [act (jsenv-act env)]
         [code (guard (e [(and (js-error? e) (eq? (js-error-kind e) 'syntax))
                          (raise (make-js-error 'runtime (js-error-message e) #f))])
                 (compile-js-program (ps-parse src 'program) (and act (jsact-info act))))])
    (let ([r (code env)])
      (if (marker? r) js-undefined r))))
