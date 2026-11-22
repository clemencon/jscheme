;;; Compiler from AST to Scheme closures, and execution contexts (§10-§14).

(define-record-type ctx (fields scope varobj this var-attrs))

;; "normal completion" without a value
(define empty-c (string->uninterned-symbol "empty"))
;; abrupt completions: break, continue, return
(define-record-type abrupt (fields type (mutable value)))

(define *caller-ctx* #f)

(define-record-type fninfo (fields name params body vars source))

;; ---------------------------------------------------------------------
;; Identifier resolution (§10.1.4)

(define (scope-lookup scope name)
  (let loop ([s scope])
    (cond [(null? s) #f]
          [(js-has-property? (car s) name) (car s)]
          [else (loop (cdr s))])))

(define (lookup-value scope name)
  (let loop ([s scope])
    (if (null? s)
        (throw-error "~a is not defined" name)
        (let find ([o (car s)])
          (if o
              (let ([p (hashtable-ref (jsobj-props o) name #f)])
                (if p (prop-get p) (find (jsobj-proto o))))
              (loop (cdr s)))))))

;; References are returned as two values: base and name. The base is an
;; object, #f for an unresolvable identifier, or 'value when the
;; expression is not a reference (name is then the value).
(define (get-ref base name)
  (cond [(eq? base 'value) name]
        [(not base) (throw-error "~a is not defined" name)]
        [else (js-get base name)]))

(define (put-ref! base name v)
  (cond [(eq? base 'value) (throw-error "invalid assignment target")]
        [(not base) (js-put! global-object name v)]
        [else (js-put! base name v)]))

(define (to-property-name k) (if (string? k) k (to-string k)))

;; Property access on any value (§11.2.1 steps 5-7)
(define (get-member b k)
  (cond
   [(jsobj? b) (js-get b (to-property-name k))]
   [(string? b)
    (let ([name (to-property-name k)])
      (if (string=? name "length")
          (fixnum->flonum (string-length b))
          (js-get string-prototype name)))]
   [(flonum? b) (js-get number-prototype (to-property-name k))]
   [(boolean? b) (js-get boolean-prototype (to-property-name k))]
   [else (to-object b)]))

(define (strip-paren e)
  (if (eq? (car e) 'paren) (strip-paren (cadr e)) e))

(define (eval-args argc c)
  (let loop ([as argc])
    (if (null? as)
        '()
        (let ([v ((car as) c)])
          (cons v (loop (cdr as)))))))

;; ---------------------------------------------------------------------
;; Operators

(define (js-add a b)
  (if (and (flonum? a) (flonum? b))
      (fl+ a b)
      (let* ([pa (to-primitive a #f)] [pb (to-primitive b #f)])
        (if (or (string? pa) (string? pb))
            (let* ([sa (to-string pa)] [sb (to-string pb)]) (string-append sa sb))
            (let* ([x (to-number pa)] [y (to-number pb)]) (fl+ x y))))))

(define (num2 op)
  (lambda (a b)
    (let* ([x (to-number a)] [y (to-number b)]) (op x y))))

(define (js-mod x y)
  (cond [(or (nan? x) (nan? y) (infinite? x) (fl= y 0.0)) +nan.0]
        [(infinite? y) x]
        [(fl= x 0.0) x]
        [(and (flinteger? x) (flinteger? y) (fl< (flabs x) 1e15) (fl< (flabs y) 1e15))
         (let ([r (remainder (flonum->fixnum x) (flonum->fixnum y))])
           (if (fx= r 0) (if (fl< x 0.0) -0.0 0.0) (fixnum->flonum r)))]
        [else
         (let* ([ex (exact x)] [ey (exact y)]
                [r (- ex (* ey (truncate (/ ex ey))))])
           (if (= r 0) (if (fl< x 0.0) -0.0 0.0) (inexact r)))]))

(define (int32 n)
  (let ([u (logand n #xFFFFFFFF)])
    (if (>= u #x80000000) (- u #x100000000) u)))

(define (js-shl a b)
  (let* ([x (to-int32 a)] [y (to-uint32 b)])
    (fixnum->flonum (int32 (ash x (logand y 31))))))
(define (js-sar a b)
  (let* ([x (to-int32 a)] [y (to-uint32 b)])
    (fixnum->flonum (ash x (- (logand y 31))))))
(define (js-shr a b)
  (let* ([x (to-uint32 a)] [y (to-uint32 b)])
    (fixnum->flonum (ash x (- (logand y 31))))))
(define (bitop op)
  (lambda (a b)
    (let* ([x (to-int32 a)] [y (to-int32 b)])
      (fixnum->flonum (int32 (op x y))))))

;; The comparison x < y of §11.8.5; returns #t, #f or 'undefined.
(define (compare-less px py)
  (if (and (string? px) (string? py))
      (string<? px py)
      (let* ([x (to-number px)] [y (to-number py)])
        (if (or (nan? x) (nan? y)) 'undefined (fl< x y)))))

(define (js-less a b)
  (let* ([px (to-primitive a 'number)] [py (to-primitive b 'number)])
    (eq? (compare-less px py) #t)))
(define (js-greater a b)
  (let* ([py (to-primitive b 'number)] [px (to-primitive a 'number)])
    (eq? (compare-less py px) #t)))
(define (js-less-eq a b)
  (let* ([py (to-primitive b 'number)] [px (to-primitive a 'number)])
    (eq? (compare-less py px) #f)))
(define (js-greater-eq a b)
  (let* ([px (to-primitive a 'number)] [py (to-primitive b 'number)])
    (eq? (compare-less px py) #f)))

;; == (§11.9.3)
(define (js-equal? x y)
  (cond
   [(and (flonum? x) (flonum? y)) (fl= x y)]
   [(and (string? x) (string? y)) (string=? x y)]
   [(and (boolean? x) (boolean? y)) (eq? x y)]
   [(and (jsobj? x) (jsobj? y)) (eq? x y)]
   [(or (eq? x js-undefined) (eq? x js-null)) (or (eq? y js-undefined) (eq? y js-null))]
   [(or (eq? y js-undefined) (eq? y js-null)) #f]
   [(and (flonum? x) (string? y)) (fl= x (to-number y))]
   [(and (string? x) (flonum? y)) (fl= (to-number x) y)]
   [(boolean? x) (js-equal? (to-number x) y)]
   [(boolean? y) (js-equal? x (to-number y))]
   [(and (or (string? x) (flonum? x)) (jsobj? y)) (js-equal? x (to-primitive y #f))]
   [(and (jsobj? x) (or (string? y) (flonum? y))) (js-equal? (to-primitive x #f) y)]
   [else #f]))

(define (binary-proc op)
  (case op
    [(+) js-add]
    [(-) (num2 fl-)]
    [(*) (num2 fl*)]
    [(/) (num2 fl/)]
    [(%) (num2 js-mod)]
    [(<<) js-shl]
    [(>>) js-sar]
    [(>>>) js-shr]
    [(&) (bitop logand)]
    [(^) (bitop logxor)]
    [(\x7C;) (bitop logior)]
    [(<) js-less]
    [(>) js-greater]
    [(<=) js-less-eq]
    [(>=) js-greater-eq]
    [(==) js-equal?]
    [(!=) (lambda (a b) (not (js-equal? a b)))]
    [else (error 'binary-proc "unknown operator" op)]))

(define (compound->binary op)
  (case op
    [(+=) '+] [(-=) '-] [(*=) '*] [(/=) '/] [(%=) '%]
    [(<<=) '<<] [(>>=) '>>] [(>>>=) '>>>] [(&=) '&] [(^=) '^] [(\x7C;=) '\x7C;]))

;; ---------------------------------------------------------------------
;; Expressions

(define (cref e)
  (let ([e (strip-paren e)])
    (case (car e)
      [(id)
       (let ([name (cadr e)])
         (lambda (c) (values (scope-lookup (ctx-scope c) name) name)))]
      [(member)
       (let ([o (cx (cadr e))] [p (cx (caddr e))])
         (lambda (c)
           (let* ([bv (o c)] [k (p c)] [base (to-object bv)])
             (values base (to-property-name k)))))]
      [else
       (let ([v (cx e)])
         (lambda (c) (values 'value (v c))))])))

(define (call-this base)
  (if (or (not (jsobj? base)) (eq? (jsobj-class base) 'activation))
      global-object
      base))

(define (cx e)
  (case (car e)
    [(lit) (let ([v (cadr e)]) (lambda (c) v))]
    [(this) (lambda (c) (ctx-this c))]
    [(id) (let ([name (cadr e)]) (lambda (c) (lookup-value (ctx-scope c) name)))]
    [(paren) (cx (cadr e))]
    [(member)
     (let ([o (cx (cadr e))] [p (cx (caddr e))])
       (lambda (c)
         (let* ([b (o c)] [k (p c)])
           (get-member b k))))]
    [(call) (compile-call (cadr e) (caddr e))]
    [(new)
     (let ([fc (cx (cadr e))] [argc (map cx (caddr e))])
       (lambda (c)
         (let* ([f (fc c)] [args (eval-args argc c)])
           (js-construct f args))))]
    [(postfix)
     (let ([r (cref (caddr e))] [delta (if (eq? (cadr e) '++) 1.0 -1.0)])
       (lambda (c)
         (let-values ([(b n) (r c)])
           (let ([old (to-number (get-ref b n))])
             (put-ref! b n (fl+ old delta))
             old))))]
    [(unary) (compile-unary (cadr e) (caddr e))]
    [(binary)
     (let ([l (cx (caddr e))] [r (cx (cadddr e))] [f (binary-proc (cadr e))])
       (lambda (c)
         (let* ([a (l c)] [b (r c)])
           (f a b))))]
    [(and)
     (let ([l (cx (cadr e))] [r (cx (caddr e))])
       (lambda (c) (let ([v (l c)]) (if (to-boolean v) (r c) v))))]
    [(or)
     (let ([l (cx (cadr e))] [r (cx (caddr e))])
       (lambda (c) (let ([v (l c)]) (if (to-boolean v) v (r c)))))]
    [(cond)
     (let ([t (cx (cadr e))] [a (cx (caddr e))] [b (cx (cadddr e))])
       (lambda (c) (if (to-boolean (t c)) (a c) (b c))))]
    [(assign)
     (let ([op (cadr e)] [r (cref (caddr e))] [v (cx (cadddr e))])
       (if (eq? op '=)
           (lambda (c)
             (let-values ([(b n) (r c)])
               (let ([val (v c)])
                 (put-ref! b n val)
                 val)))
           (let ([f (binary-proc (compound->binary op))])
             (lambda (c)
               (let-values ([(b n) (r c)])
                 (let* ([old (get-ref b n)] [rhs (v c)] [val (f old rhs)])
                   (put-ref! b n val)
                   val))))))]
    [(comma)
     (let ([l (cx (cadr e))] [r (cx (caddr e))])
       (lambda (c) (l c) (r c)))]
    [(funcexpr)
     (let ([fi (compile-function (cadr e))])
       (lambda (c) (make-script-function fi)))]
    [else (error 'cx "unknown expression" e)]))

(define (compile-call callee args)
  (let ([argc (map cx args)] [callee (strip-paren callee)])
    (case (car callee)
      [(id)
       (let ([name (cadr callee)])
         (lambda (c)
           (let* ([base (scope-lookup (ctx-scope c) name)]
                  [args (eval-args argc c)])
             (unless base (throw-error "~a is not defined" name))
             (let ([f (js-get base name)])
               (unless (and (jsobj? f) (jsobj-call f))
                 (throw-error "~a is not a function" name))
               (set! *caller-ctx* c)
               ((jsobj-call f) (call-this base) args)))))]
      [(member)
       (let ([o (cx (cadr callee))] [p (cx (caddr callee))]
             [label (let ([pe (caddr callee)]) (if (and (eq? (car pe) 'lit) (string? (cadr pe))) (cadr pe) "expression"))])
         (lambda (c)
           (let* ([bv (o c)] [k (p c)] [base (to-object bv)] [name (to-property-name k)]
                  [args (eval-args argc c)])
             (let ([f (js-get base name)])
               (unless (and (jsobj? f) (jsobj-call f))
                 (throw-error "~a is not a function" label))
               (set! *caller-ctx* c)
               ((jsobj-call f) base args)))))]
      [else
       (let ([fc (cx callee)])
         (lambda (c)
           (let* ([f (fc c)] [args (eval-args argc c)])
             (unless (and (jsobj? f) (jsobj-call f))
               (throw-error "~a is not a function" (js-typeof f)))
             (set! *caller-ctx* c)
             ((jsobj-call f) global-object args))))])))

(define (compile-unary op operand)
  (case op
    [(delete)
     (let ([r (cref operand)])
       (lambda (c)
         (let-values ([(b n) (r c)])
           (cond [(eq? b 'value) (throw-error "delete of a value that is not a reference")]
                 [(not b) #t]
                 [else (js-delete! b n)]))))]
    [(void)
     (let ([v (cx operand)]) (lambda (c) (v c) js-undefined))]
    [(typeof)
     (let ([o (strip-paren operand)])
       (if (eq? (car o) 'id)
           (let ([name (cadr o)])
             (lambda (c)
               (let ([b (scope-lookup (ctx-scope c) name)])
                 (if b (js-typeof (js-get b name)) "undefined"))))
           (let ([v (cx o)]) (lambda (c) (js-typeof (v c))))))]
    [(pre++ pre--)
     (let ([r (cref operand)] [delta (if (eq? op 'pre++) 1.0 -1.0)])
       (lambda (c)
         (let-values ([(b n) (r c)])
           (let ([new (fl+ (to-number (get-ref b n)) delta)])
             (put-ref! b n new)
             new))))]
    [(+) (let ([v (cx operand)]) (lambda (c) (to-number (v c))))]
    [(-) (let ([v (cx operand)]) (lambda (c) (fl- (to-number (v c)))))]
    [(~) (let ([v (cx operand)]) (lambda (c) (fixnum->flonum (lognot (to-int32 (v c))))))]
    [(!) (let ([v (cx operand)]) (lambda (c) (not (to-boolean (v c)))))]
    [else (error 'compile-unary "unknown operator" op)]))

;; ---------------------------------------------------------------------
;; Statements. A compiled statement returns a value, empty-c, or an
;; abrupt record.

(define (compile-stmt-list stmts)
  (let* ([procs (list->vector (map cs stmts))] [n (vector-length procs)])
    (lambda (c)
      (let loop ([i 0] [v empty-c])
        (if (fx= i n)
            v
            (let ([r ((vector-ref procs i) c)])
              (if (abrupt? r)
                  (begin
                    (when (eq? (abrupt-value r) empty-c) (abrupt-value-set! r v))
                    r)
                  (loop (fx+ i 1) (if (eq? r empty-c) v r)))))))))

(define (compile-var-decl d)
  (let ([name (car d)])
    (if (cdr d)
        (let ([init (cx (cdr d))])
          (lambda (c)
            (let ([b (scope-lookup (ctx-scope c) name)])
              (put-ref! b name (init c)))))
        (lambda (c) (void)))))

(define (cs s)
  (case (car s)
    [(block) (compile-stmt-list (cadr s))]
    [(var)
     (let ([ds (map compile-var-decl (cadr s))])
       (lambda (c) (for-each (lambda (d) (d c)) ds) empty-c))]
    [(empty) (lambda (c) empty-c)]
    [(fundecl) (lambda (c) empty-c)]
    [(expr) (cx (cadr s))]
    [(if)
     (let ([t (cx (cadr s))] [a (cs (caddr s))] [b (if (cadddr s) (cs (cadddr s)) #f)])
       (if b
           (lambda (c) (if (to-boolean (t c)) (a c) (b c)))
           (lambda (c) (if (to-boolean (t c)) (a c) empty-c))))]
    [(while)
     (let ([t (cx (cadr s))] [body (cs (caddr s))])
       (lambda (c)
         (let loop ([C empty-c])
           (if (to-boolean (t c))
               (let ([res (body c)])
                 (if (abrupt? res)
                     (let ([C (if (eq? (abrupt-value res) empty-c) C (abrupt-value res))])
                       (case (abrupt-type res)
                         [(break) C]
                         [(continue) (loop C)]
                         [else res]))
                     (loop (if (eq? res empty-c) C res))))
               C))))]
    [(for)
     (let* ([init (cadr s)]
            [initc (cond [(not init) (lambda (c) (void))]
                         [(eq? (car init) 'var) (cs init)]
                         [else (cx (cadr init))])]
            [t (if (caddr s) (cx (caddr s)) (lambda (c) #t))]
            [u (if (cadddr s) (cx (cadddr s)) (lambda (c) (void)))]
            [body (cs (list-ref s 4))])
       (lambda (c)
         (initc c)
         (let loop ([C empty-c])
           (if (to-boolean (t c))
               (let ([res (body c)])
                 (if (abrupt? res)
                     (let ([C (if (eq? (abrupt-value res) empty-c) C (abrupt-value res))])
                       (case (abrupt-type res)
                         [(break) C]
                         [(continue) (u c) (loop C)]
                         [else res]))
                     (begin (u c) (loop (if (eq? res empty-c) C res)))))
               C))))]
    [(forin)
     (let* ([target (cadr s)]
            [is-var (and (pair? target) (eq? (car target) 'var))]
            [initc (if is-var (cs target) (lambda (c) (void)))]
            [refc (if is-var (cref (list 'id (car (car (cadr target))))) (cref target))]
            [objc (cx (caddr s))]
            [body (cs (cadddr s))])
       (lambda (c)
         (initc c)
         (let* ([o (to-object (objc c))] [names (enumerable-names o)])
           (let loop ([names names] [C empty-c])
             (cond
              [(null? names) C]
              [(not (js-has-property? o (car names))) (loop (cdr names) C)]
              [else
               (let-values ([(b n) (refc c)])
                 (put-ref! b n (car names)))
               (let ([res (body c)])
                 (if (abrupt? res)
                     (let ([C (if (eq? (abrupt-value res) empty-c) C (abrupt-value res))])
                       (case (abrupt-type res)
                         [(break) C]
                         [(continue) (loop (cdr names) C)]
                         [else res]))
                     (loop (cdr names) (if (eq? res empty-c) C res))))])))))]
    [(continue) (lambda (c) (make-abrupt 'continue empty-c))]
    [(break) (lambda (c) (make-abrupt 'break empty-c))]
    [(return)
     (if (cadr s)
         (let ([v (cx (cadr s))]) (lambda (c) (make-abrupt 'return (v c))))
         (lambda (c) (make-abrupt 'return js-undefined)))]
    [(with)
     (let ([e (cx (cadr s))] [body (cs (caddr s))])
       (lambda (c)
         (let ([o (to-object (e c))])
           (body (make-ctx (cons o (ctx-scope c)) (ctx-varobj c) (ctx-this c) (ctx-var-attrs c))))))]
    [else (error 'cs "unknown statement" s)]))

;; Names for for-in (§12.6.3): own properties first, then the prototype's,
;; skipping DontEnum and shadowed ones.
(define (enumerable-names o)
  (let ([seen (make-hashtable string-hash string=?)])
    (let loop ([o o] [acc '()])
      (if (not o)
          (reverse acc)
          (let-values ([(keys vals) (hashtable-entries (jsobj-props o))])
            (let* ([pairs (map cons (vector->list keys) (vector->list vals))]
                   [pairs (list-sort (lambda (a b) (fx< (prop-seq (cdr a)) (prop-seq (cdr b)))) pairs)]
                   [acc (fold-left
                         (lambda (acc pr)
                           (if (hashtable-ref seen (car pr) #f)
                               acc
                               (begin
                                 (hashtable-set! seen (car pr) #t)
                                 (if (attr? (cdr pr) DE) acc (cons (car pr) acc)))))
                         acc pairs)])
              (loop (jsobj-proto o) acc)))))))

;; ---------------------------------------------------------------------
;; Functions and programs

(define (collect-vars stmts)
  (let ([acc '()])
    (define (walk s)
      (case (car s)
        [(var) (for-each (lambda (d) (set! acc (cons (car d) acc))) (cadr s))]
        [(block) (for-each walk (cadr s))]
        [(if) (walk (caddr s)) (when (cadddr s) (walk (cadddr s)))]
        [(while) (walk (caddr s))]
        [(for) (when (and (cadr s) (eq? (car (cadr s)) 'var)) (walk (cadr s)))
               (walk (list-ref s 4))]
        [(forin) (when (eq? (car (cadr s)) 'var) (walk (cadr s)))
                 (walk (cadddr s))]
        [(with) (walk (caddr s))]
        [else (void)]))
    (for-each walk stmts)
    (reverse acc)))

;; fn: (name params body-stmts source)
(define (compile-function fn)
  (let ([body (caddr fn)])
    (make-fninfo (car fn) (cadr fn) (compile-stmt-list body) (collect-vars body) (cadddr fn))))

(define (compile-js-program ast)
  (let ([fundecls (map (lambda (s) (compile-function (cadr s)))
                       (filter (lambda (s) (eq? (car s) 'fundecl)) ast))])
    (values fundecls (collect-vars ast) (compile-stmt-list ast))))

;; Variable instantiation (§10.1.3)
(define (instantiate! varobj fundecls vars attrs)
  (for-each (lambda (fi) (define-prop! varobj (fninfo-name fi) (make-script-function fi) attrs))
            fundecls)
  (for-each (lambda (v) (unless (own-prop varobj v) (add-prop! varobj v js-undefined attrs)))
            vars))

(define (call-script f fi this args)
  (let* ([act (make-obj #f 'activation)]
         [argsobj (make-obj object-prototype 'Object)]
         [c (make-ctx (list act global-object) act (if (jsobj? this) this global-object) DD)]
         [params (fninfo-params fi)])
    (add-prop! argsobj "callee" f DE)
    (add-prop! argsobj "length" (fixnum->flonum (length args)) DE)
    (add-prop! act "arguments" argsobj DD)
    (let loop ([ps params] [as args])
      (unless (null? ps)
        (let ([v (if (null? as) js-undefined (car as))] [p (own-prop act (car ps))])
          (if p (prop-value-set! p v) (add-prop! act (car ps) v DD)))
        (loop (cdr ps) (if (null? as) '() (cdr as)))))
    (let loop ([i 0] [as args] [ps params])
      (unless (null? as)
        (add-prop! argsobj (number->string i)
                   (if (null? ps) (car as) (make-prop-alias (own-prop act (car ps))))
                   DE)
        (loop (fx+ i 1) (cdr as) (if (null? ps) '() (cdr ps)))))
    (for-each (lambda (v) (unless (own-prop act v) (add-prop! act v js-undefined DD)))
              (fninfo-vars fi))
    (let* ([ap (own-prop f "arguments")]
           [old (and ap (prop-value ap))])
      (when ap (prop-value-set! ap argsobj))
      (let ([r ((fninfo-body fi) c)])
        (when ap (prop-value-set! ap old))
        (if (and (abrupt? r) (eq? (abrupt-type r) 'return))
            (abrupt-value r)
            js-undefined)))))

(define (construct-script f args)
  (let* ([proto (js-get f "prototype")]
         [o (make-obj (if (jsobj? proto) proto object-prototype) 'Object)]
         [r ((jsobj-call f) o args)])
    (if (jsobj? r) r o)))

(define (make-script-function fi)
  (let ([f (make-obj function-prototype 'Function)])
    (jsobj-info-set! f fi)
    (jsobj-call-set! f (lambda (this args) (call-script f fi this args)))
    (jsobj-construct-set! f (lambda (args) (construct-script f args)))
    (add-prop! f "length" (fixnum->flonum (length (fninfo-params fi))) (fxior RO DD DE))
    (let ([p (make-obj object-prototype 'Object)])
      (add-prop! p "constructor" f DE)
      (add-prop! f "prototype" p DE))
    (add-prop! f "arguments" js-null (fxior DE DD))
    f))

;; Parses and runs src as a Program in context c (eval code, §10.2.2).
(define (eval-in-context src c attrs)
  (let ([ast (guard (e [(syntax-err? e) (throw-error "SyntaxError: ~a" (syntax-err-msg e))])
               (parse-js src 'program))])
    (run-program ast c attrs)))

(define (run-program ast c attrs)
  (let-values ([(fundecls vars body) (compile-js-program ast)])
    (instantiate! (ctx-varobj c) fundecls vars attrs)
    (let ([r (body c)])
      (cond [(abrupt? r) (if (eq? (abrupt-value r) empty-c) js-undefined (abrupt-value r))]
            [(eq? r empty-c) js-undefined]
            [else r]))))
