;;; Lexer for ECMA-262 1st edition (§7).
;;; Source text is a Scheme string whose characters are UTF-16 code units
;;; (see unit->char in strings.ss).

(define-record-type syntax-err (fields msg at-eof?))

(define-record-type tok (fields type val nl start end))
;; type: id kw punct num str eof
;; nl: #t when a line terminator precedes the token

(define keywords
  '("break" "continue" "delete" "else" "for" "function" "if" "in" "new"
    "return" "this" "typeof" "var" "void" "while" "with"
    "null" "true" "false"
    ;; future reserved words (§7.4.3)
    "case" "catch" "class" "const" "debugger" "default" "do" "enum"
    "export" "extends" "finally" "import" "super" "switch" "throw" "try"))

(define keyword-table
  (let ([h (make-hashtable string-hash string=?)])
    (for-each (lambda (k) (hashtable-set! h k (string->symbol k))) keywords)
    h))

(define punctuators
  ;; longest first
  '(">>>=" ">>>" "<<=" ">>=" "==" "!=" "<=" ">=" "&&" "||" "++" "--"
    "+=" "-=" "*=" "/=" "&=" "|=" "^=" "%=" "<<" ">>"
    "=" ">" "<" "," "!" "~" "?" ":" "." "+" "-" "*" "/" "&" "|" "^" "%"
    "(" ")" "{" "}" "[" "]" ";"))

(define punct-symbols
  (let ([h (make-hashtable string-hash string=?)])
    (for-each (lambda (p) (hashtable-set! h p (string->symbol p))) punctuators)
    h))

(define (ws-char? c)
  (memv c '(#\tab #\xB #\xC #\space)))
(define (lt-char? c)
  (or (char=? c #\newline) (char=? c #\return)))
(define (ascii-letter? c)
  (or (char<=? #\a c #\z) (char<=? #\A c #\Z)))
(define (digit-char? c) (char<=? #\0 c #\9))
(define (id-start? c) (or (ascii-letter? c) (char=? c #\_) (char=? c #\$)))
(define (id-part? c) (or (id-start? c) (digit-char? c)))
(define (hex-digit-value c)
  (cond [(char<=? #\0 c #\9) (- (char->integer c) 48)]
        [(char<=? #\a c #\f) (- (char->integer c) 87)]
        [(char<=? #\A c #\F) (- (char->integer c) 55)]
        [else #f]))

;; The number value for the mathematical value m * 10^e (m an exact
;; nonnegative integer), rounded as in §8.5.
(define (decimal->number m e)
  (cond [(= m 0) 0.0]
        [else
         (let ([digits (string-length (number->string m))])
           (cond [(> (+ digits e) 330) +inf.0]
                 [(< (+ digits e) -350) 0.0]
                 [else (inexact (* m (expt 10 e)))]))]))

(define (lex src)
  (let ([n (string-length src)])
    (define (ch i) (if (< i n) (string-ref src i) #\nul))
    (define (err msg at-eof) (raise (make-syntax-err msg at-eof)))
    (let loop ([i 0] [nl #f] [acc '()])
      (if (>= i n)
          (list->vector (reverse (cons (make-tok 'eof #f nl n n) acc)))
          (let ([c (string-ref src i)])
            (cond
             [(ws-char? c) (loop (+ i 1) nl acc)]
             [(lt-char? c) (loop (+ i 1) #t acc)]
             ;; comments
             [(and (char=? c #\/) (char=? (ch (+ i 1)) #\/))
              (let skip ([j (+ i 2)])
                (if (or (>= j n) (lt-char? (string-ref src j)))
                    (loop j nl acc)
                    (skip (+ j 1))))]
             [(and (char=? c #\/) (char=? (ch (+ i 1)) #\*))
              (let skip ([j (+ i 2)] [has-nl #f])
                (cond [(>= (+ j 1) n) (err "unterminated comment" #t)]
                      [(and (char=? (string-ref src j) #\*) (char=? (string-ref src (+ j 1)) #\/))
                       (loop (+ j 2) (or nl has-nl) acc)]
                      [else (skip (+ j 1) (or has-nl (lt-char? (string-ref src j))))]))]
             ;; identifiers and reserved words
             [(id-start? c)
              (let scan ([j (+ i 1)])
                (if (and (< j n) (id-part? (string-ref src j)))
                    (scan (+ j 1))
                    (let* ([s (substring src i j)]
                           [k (hashtable-ref keyword-table s #f)])
                      (loop j #f (cons (if k (make-tok 'kw k nl i j) (make-tok 'id s nl i j)) acc)))))]
             ;; numbers
             [(or (digit-char? c) (and (char=? c #\.) (digit-char? (ch (+ i 1)))))
              (let-values ([(v j) (lex-number src i n err)])
                (when (and (< j n) (id-part? (string-ref src j)))
                  (err "identifier directly after number" #f))
                (loop j #f (cons (make-tok 'num v nl i j) acc)))]
             ;; strings
             [(or (char=? c #\") (char=? c #\'))
              (let-values ([(v j) (lex-string src i n err)])
                (loop j #f (cons (make-tok 'str v nl i j) acc)))]
             [else
              (let try ([ps punctuators])
                (cond [(null? ps) (err (format "unexpected character U+~4,'0x" (char->unit c)) #f)]
                      [(let ([p (car ps)])
                         (and (<= (+ i (string-length p)) n)
                              (string=? p (substring src i (+ i (string-length p))))))
                       (let ([p (car ps)])
                         (loop (+ i (string-length p)) #f
                               (cons (make-tok 'punct (hashtable-ref punct-symbols p #f) nl i (+ i (string-length p))) acc)))]
                      [else (try (cdr ps))]))]))))))

(define (lex-number src i n err)
  (define (ch k) (if (< k n) (string-ref src k) #\nul))
  (cond
   [(and (char=? (ch i) #\0) (memv (ch (+ i 1)) '(#\x #\X)))
    (let scan ([j (+ i 2)] [v 0])
      (let ([d (and (< j n) (hex-digit-value (string-ref src j)))])
        (if d
            (scan (+ j 1) (+ (* v 16) d))
            (if (= j (+ i 2))
                (err "hexadecimal literal without digits" (>= j n))
                (values (inexact v) j)))))]
   [(and (char=? (ch i) #\0) (digit-char? (ch (+ i 1))))
    (let scan ([j (+ i 1)] [v 0])
      (let ([c (ch j)])
        (if (char<=? #\0 c #\7)
            (scan (+ j 1) (+ (* v 8) (- (char->integer c) 48)))
            (values (inexact v) j))))]
   [else
    (let* ([int-end (let scan ([j i]) (if (digit-char? (ch j)) (scan (+ j 1)) j))]
           [int-digits (substring src i int-end)])
      (let-values ([(frac-digits after-frac)
                    (if (char=? (ch int-end) #\.)
                        (let scan ([j (+ int-end 1)])
                          (if (digit-char? (ch j)) (scan (+ j 1))
                              (values (substring src (+ int-end 1) j) j)))
                        (values "" int-end))])
        (let-values ([(exp after-exp)
                      (if (memv (ch after-frac) '(#\e #\E))
                          (let* ([k (+ after-frac 1)]
                                 [sign (case (ch k) [(#\+) 1] [(#\-) -1] [else 0])]
                                 [ds (if (= sign 0) k (+ k 1))])
                            (let scan ([j ds])
                              (if (digit-char? (ch j)) (scan (+ j 1))
                                  (if (= j ds)
                                      (err "exponent without digits" (>= j n))
                                      (values (* (if (= sign -1) -1 1) (string->number (substring src ds j))) j)))))
                          (values 0 after-frac))])
          (let ([m (string->number (string-append "0" int-digits frac-digits))])
            (values (decimal->number m (- exp (string-length frac-digits))) after-exp)))))]))

(define (lex-string src i n err)
  (let ([q (string-ref src i)] [out (open-output-string)])
    (define (ch k) (if (< k n) (string-ref src k) #\nul))
    (define (octal? c) (char<=? #\0 c #\7))
    (let loop ([j (+ i 1)])
      (when (>= j n) (err "unterminated string literal" #t))
      (let ([c (string-ref src j)])
        (cond
         [(char=? c q) (values (get-output-string out) (+ j 1))]
         [(lt-char? c) (err "line terminator in string literal" #f)]
         [(char=? c #\\)
          (when (>= (+ j 1) n) (err "unterminated string literal" #t))
          (let ([e (string-ref src (+ j 1))])
            (case e
              [(#\b) (write-char #\backspace out) (loop (+ j 2))]
              [(#\t) (write-char #\tab out) (loop (+ j 2))]
              [(#\n) (write-char #\newline out) (loop (+ j 2))]
              [(#\f) (write-char #\page out) (loop (+ j 2))]
              [(#\r) (write-char #\return out) (loop (+ j 2))]
              [(#\x)
               (let ([a (hex-digit-value (ch (+ j 2)))] [b (hex-digit-value (ch (+ j 3)))])
                 (unless (and a b) (err "bad hexadecimal escape" (>= (+ j 3) n)))
                 (write-char (unit->char (+ (* a 16) b)) out)
                 (loop (+ j 4)))]
              [(#\u)
               (let ([ds (map (lambda (k) (hex-digit-value (ch (+ j k)))) '(2 3 4 5))])
                 (unless (andmap values ds) (err "bad unicode escape" (>= (+ j 5) n)))
                 (write-char (unit->char (fold-left (lambda (a d) (+ (* a 16) d)) 0 ds)) out)
                 (loop (+ j 6)))]
              [else
               (cond
                [(lt-char? e) (err "line terminator in string literal" #f)]
                [(octal? e)
                 (let* ([d0 (- (char->integer e) 48)]
                        [c1 (ch (+ j 2))])
                   (if (octal? c1)
                       (let ([d1 (- (char->integer c1) 48)] [c2 (ch (+ j 3))])
                         (if (and (<= d0 3) (octal? c2))
                             (begin (write-char (unit->char (+ (* d0 64) (* d1 8) (- (char->integer c2) 48))) out)
                                    (loop (+ j 4)))
                             (begin (write-char (unit->char (+ (* d0 8) d1)) out)
                                    (loop (+ j 3)))))
                       (begin (write-char (unit->char d0) out) (loop (+ j 2)))))]
                [else (write-char e out) (loop (+ j 2))])]))]
         [else (write-char c out) (loop (+ j 1))])))))
