;;; Parser for ECMA-262 1st edition (§11-§14), with automatic semicolon
;;; insertion (§7.8). Produces an AST of tagged lists.
;;;
;;; Expressions:
;;;   (lit v) (this) (id name) (paren e) (member obj prop) (call callee args)
;;;   (new callee args) (postfix op e) (unary op e) (binary op l r)
;;;   (and l r) (or l r) (cond c t f) (assign op target value) (comma l r)
;;;   (funcexpr fninfo-ast)   -- extension: function expressions
;;; Statements:
;;;   (block stmts) (var decls) (empty) (expr e) (if c t f) (while c body)
;;;   (for init test update body) (forin target obj body) (continue) (break)
;;;   (return e) (with e body) (fundecl fn)
;;; fn: (name params body-stmts source-text)

;; kind: program | function-body | params
(define (parse-js src kind)
  (define toks (lex src))
  (define pos 0)
  (define in-func (eq? kind 'function-body))
  (define loop-depth 0)

  (define (tok) (vector-ref toks pos))
  (define (advance!) (let ([t (tok)]) (set! pos (+ pos 1)) t))
  (define (eof?) (eq? (tok-type (tok)) 'eof))
  (define (punct? v) (let ([t (tok)]) (and (eq? (tok-type t) 'punct) (eq? (tok-val t) v))))
  (define (kw? v) (let ([t (tok)]) (and (eq? (tok-type t) 'kw) (eq? (tok-val t) v))))
  (define (fail msg)
    (raise (make-syntax-err
            (if (eof?) (string-append msg " at end of input")
                (format "~a near offset ~a" msg (tok-start (tok))))
            (eof?))))
  (define (expect-punct v)
    (if (punct? v) (advance!) (fail (format "expected ~a" v))))
  (define (consume-semicolon)
    (cond [(punct? '|;|) (advance!)]
          [(or (punct? '|}|) (eof?) (tok-nl (tok))) (void)]
          [else (fail "expected ;")]))
  (define (identifier)
    (let ([t (tok)])
      (if (eq? (tok-type t) 'id) (begin (advance!) (tok-val t)) (fail "expected identifier"))))

  ;; ---- expressions ----
  (define (lhs-form? e)
    (memq (car e) '(id member call new lit this paren funcexpr)))

  (define (parse-expression)
    (let loop ([e (parse-assignment)])
      (if (punct? '|,|)
          (begin (advance!) (loop (list 'comma e (parse-assignment))))
          e)))

  (define assign-ops '(= += -= *= /= %= <<= >>= >>>= &= ^= \x7C;=))

  (define (parse-assignment)
    (let ([lhs (parse-conditional)])
      (let ([t (tok)])
        (if (and (eq? (tok-type t) 'punct) (memq (tok-val t) assign-ops))
            (if (lhs-form? lhs)
                (begin (advance!) (list 'assign (tok-val t) lhs (parse-assignment)))
                (fail "invalid assignment target"))
            lhs))))

  (define (parse-conditional)
    (let ([c (parse-binary 1)])
      (if (punct? '?)
          (begin (advance!)
                 (let ([a (parse-assignment)])
                   (expect-punct ':)
                   (list 'cond c a (parse-assignment))))
          c)))

  (define (binary-prec t)
    (and (eq? (tok-type t) 'punct)
         (case (tok-val t)
           [(\x7C;\x7C;) 1] [(&&) 2] [(\x7C;) 3] [(^) 4] [(&) 5]
           [(== !=) 6] [(< > <= >=) 7] [(<< >> >>>) 8] [(+ -) 9] [(* / %) 10]
           [else #f])))

  (define (parse-binary min-prec)
    (let loop ([left (parse-unary)])
      (let* ([t (tok)] [p (binary-prec t)])
        (if (and p (>= p min-prec))
            (begin
              (advance!)
              (let ([right (parse-binary (+ p 1))]
                    [op (tok-val t)])
                (loop (case op
                        [(\x7C;\x7C;) (list 'or left right)]
                        [(&&) (list 'and left right)]
                        [else (list 'binary op left right)]))))
            left))))

  (define (parse-unary)
    (let ([t (tok)])
      (cond
       [(and (eq? (tok-type t) 'kw) (memq (tok-val t) '(delete void typeof)))
        (advance!) (list 'unary (tok-val t) (parse-unary))]
       [(and (eq? (tok-type t) 'punct) (memq (tok-val t) '(++ --)))
        (advance!) (list 'unary (if (eq? (tok-val t) '++) 'pre++ 'pre--) (parse-unary))]
       [(and (eq? (tok-type t) 'punct) (memq (tok-val t) '(+ - ~ !)))
        (advance!) (list 'unary (tok-val t) (parse-unary))]
       [else (parse-postfix)])))

  (define (parse-postfix)
    (let ([e (parse-lhs)])
      (let ([t (tok)])
        (if (and (eq? (tok-type t) 'punct) (memq (tok-val t) '(++ --)) (not (tok-nl t)))
            (begin (advance!) (list 'postfix (tok-val t) e))
            e))))

  (define (parse-arguments)
    (expect-punct '|(|)
    (if (punct? '|)|)
        (begin (advance!) '())
        (let loop ([args (list (parse-assignment))])
          (cond [(punct? '|,|) (advance!) (loop (cons (parse-assignment) args))]
                [else (expect-punct '|)|) (reverse args)]))))

  (define (parse-member-suffixes e)
    (cond [(punct? '|.|) (advance!) (parse-member-suffixes (list 'member e (list 'lit (identifier))))]
          [(punct? '|[|) (advance!)
           (let ([p (parse-expression)])
             (expect-punct '|]|)
             (parse-member-suffixes (list 'member e p)))]
          [else e]))

  (define (parse-member-expression)
    (if (kw? 'new)
        (begin
          (advance!)
          (let ([callee (parse-member-expression)])
            (parse-member-suffixes
             (if (punct? '|(|)
                 (list 'new callee (parse-arguments))
                 (list 'new callee '())))))
        (parse-member-suffixes (parse-primary))))

  (define (parse-lhs)
    (let loop ([e (parse-member-expression)])
      (cond [(punct? '|(|) (loop (list 'call e (parse-arguments)))]
            [(or (punct? '|.|) (punct? '|[|)) (loop (parse-member-suffixes e))]
            [else e])))

  (define (parse-primary)
    (let ([t (tok)])
      (case (tok-type t)
        [(id) (advance!) (list 'id (tok-val t))]
        [(num str) (advance!) (list 'lit (tok-val t))]
        [(kw)
         (case (tok-val t)
           [(this) (advance!) '(this)]
           [(null) (advance!) (list 'lit js-null)]
           [(true) (advance!) '(lit #t)]
           [(false) (advance!) '(lit #f)]
           [(function) (list 'funcexpr (parse-function #f))]
           [else (fail "unexpected keyword")])]
        [(punct)
         (if (punct? '|(|)
             (begin (advance!)
                    (let ([e (parse-expression)])
                      (expect-punct '|)|)
                      (list 'paren e)))
             (fail "unexpected token"))]
        [else (fail "unexpected end of input")])))

  ;; ---- functions ----
  (define (parse-params)
    (expect-punct '|(|)
    (if (punct? '|)|)
        (begin (advance!) '())
        (let loop ([ps (list (identifier))])
          (cond [(punct? '|,|) (advance!) (loop (cons (identifier) ps))]
                [else (expect-punct '|)|) (reverse ps)]))))

  ;; After the `function` keyword is current. name-required? for declarations.
  (define (parse-function name-required?)
    (let* ([start (tok-start (advance!))]
           [name (cond [(eq? (tok-type (tok)) 'id) (identifier)]
                       [name-required? (fail "function name expected")]
                       [else "anonymous"])]
           [params (parse-params)])
      (unless (punct? '|{|) (fail "expected {"))
      (advance!)
      (let ([saved-in-func in-func] [saved-depth loop-depth])
        (set! in-func #t)
        (set! loop-depth 0)
        (let ([body (parse-statements-until-brace)])
          (let ([end (tok-end (expect-punct '|}|))])
            (set! in-func saved-in-func)
            (set! loop-depth saved-depth)
            (list name params body (substring src start end)))))))

  ;; ---- statements ----
  (define (parse-statements-until-brace)
    (let loop ([acc '()])
      (if (or (punct? '|}|) (eof?))
          (reverse acc)
          (loop (cons (parse-statement) acc)))))

  (define (parse-var-decls)
    (let loop ([acc '()])
      (let* ([name (identifier)]
             [init (if (punct? '=) (begin (advance!) (parse-assignment)) #f)]
             [acc (cons (cons name init) acc)])
        (if (punct? '|,|)
            (begin (advance!) (loop acc))
            (reverse acc)))))

  (define (parse-loop-body)
    (set! loop-depth (+ loop-depth 1))
    (let ([s (parse-statement)])
      (set! loop-depth (- loop-depth 1))
      s))

  (define (parse-statement)
    (let ([t (tok)])
      (case (tok-type t)
        [(punct)
         (case (tok-val t)
           [(|{|) (advance!)
            (let ([stmts (parse-statements-until-brace)])
              (expect-punct '|}|)
              (list 'block stmts))]
           [(|;|) (advance!) '(empty)]
           [else (parse-expression-statement)])]
        [(kw)
         (case (tok-val t)
           [(var) (advance!)
            (let ([decls (parse-var-decls)])
              (consume-semicolon)
              (list 'var decls))]
           [(if) (advance!)
            (expect-punct '|(|)
            (let ([c (parse-expression)])
              (expect-punct '|)|)
              (let ([then (parse-statement)])
                (if (kw? 'else)
                    (begin (advance!) (list 'if c then (parse-statement)))
                    (list 'if c then #f))))]
           [(while) (advance!)
            (expect-punct '|(|)
            (let ([c (parse-expression)])
              (expect-punct '|)|)
              (list 'while c (parse-loop-body)))]
           [(for) (advance!) (parse-for)]
           [(continue) (advance!)
            (when (= loop-depth 0) (fail "continue outside a loop"))
            (consume-semicolon) '(continue)]
           [(break) (advance!)
            (when (= loop-depth 0) (fail "break outside a loop"))
            (consume-semicolon) '(break)]
           [(return) (advance!)
            (unless in-func (fail "return outside a function"))
            (if (or (punct? '|;|) (punct? '|}|) (eof?) (tok-nl (tok)))
                (begin (consume-semicolon) '(return #f))
                (let ([e (parse-expression)])
                  (consume-semicolon)
                  (list 'return e)))]
           [(with) (advance!)
            (expect-punct '|(|)
            (let ([e (parse-expression)])
              (expect-punct '|)|)
              (list 'with e (parse-statement)))]
           [(function) (fail "function declaration not allowed here")]
           [else (parse-expression-statement)])]
        [(eof) (fail "statement expected")]
        [else (parse-expression-statement)])))

  (define (parse-expression-statement)
    (let ([e (parse-expression)])
      (consume-semicolon)
      (list 'expr e)))

  (define (parse-for)
    (expect-punct '|(|)
    (let ([init
           (cond
            [(kw? 'var) (advance!) (list 'var (parse-var-decls))]
            [(punct? '|;|) #f]
            [else (list 'expr (parse-expression))])])
      (if (kw? 'in)
          (begin
            (advance!)
            (let ([target
                   (cond
                    [(and (eq? (car init) 'var) (= (length (cadr init)) 1)) init]
                    [(and (eq? (car init) 'expr) (lhs-form? (cadr init))) (cadr init)]
                    [else (fail "invalid for-in target")])])
              (let ([obj (parse-expression)])
                (expect-punct '|)|)
                (list 'forin target obj (parse-loop-body)))))
          (begin
            (expect-punct '|;|)
            (let ([test (if (punct? '|;|) #f (parse-expression))])
              (expect-punct '|;|)
              (let ([update (if (punct? '|)|) #f (parse-expression))])
                (expect-punct '|)|)
                (list 'for init test update (parse-loop-body))))))))

  (define (parse-source-elements)
    (let loop ([acc '()])
      (cond [(eof?) (reverse acc)]
            [(kw? 'function) (loop (cons (list 'fundecl (parse-function #t)) acc))]
            [else (loop (cons (parse-statement) acc))])))

  (case kind
    [(program) (parse-source-elements)]
    [(function-body)
     (let ([stmts (let loop ([acc '()])
                    (if (eof?) (reverse acc) (loop (cons (parse-statement) acc))))])
       stmts)]
    [(params)
     (if (eof?)
         '()
         (let loop ([ps (list (identifier))])
           (cond [(punct? '|,|) (advance!) (loop (cons (identifier) ps))]
                 [(eof?) (reverse ps)]
                 [else (fail "bad parameter list")])))]))
