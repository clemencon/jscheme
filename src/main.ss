;;; jscheme: an interpreter for ECMA-262 1st edition.
;;; Entry point: command line handling, file mode and REPL (README "Host contract").

(import (chezscheme))

(include "strings.ss")
(include "lexer.ss")
(include "runtime.ss")
(include "parser.ss")
(include "interp.ss")
(include "builtins.ss")
(include "date.ss")

(define global-ctx (make-ctx (list global-object) global-object global-object 0))
(set! *caller-ctx* global-ctx)

(define (report kind msg)
  (flush-stdout)
  (let ([e (current-error-port)])
    (put-string e (string-append kind ": " (js->codepoints msg) "\n"))
    (flush-output-port e)))

(define (condition->string c)
  (if (condition? c)
      (call-with-string-output-port
       (lambda (p) (display-condition c p)))
      (format "~s" c)))

;; Runs a whole program text. Returns the exit code.
(define (run-source src)
  (let ([ast (guard (e [(syntax-err? e) (report "SyntaxError" (syntax-err-msg e)) #f])
               (parse-js src 'program))])
    (if (not ast)
        2
        (guard (e [(js-error? e) (report "Error" (js-error-msg e)) 1]
                  [else (report "Error" (condition->string e)) 1])
          (run-program ast global-ctx 0)
          (flush-stdout)
          0))))

(define (usage-error msg)
  (let ([e (current-error-port)])
    (put-string e (string-append "jscheme: " msg "\n"
                                 "usage: jscheme [-e source | file.js]\n"))
    (flush-output-port e))
  (exit 3))

(define (read-file path)
  (guard (e [else #f])
    (and (file-regular? path)
         (let ([p (open-file-input-port path)])
           (let ([bv (get-bytevector-all p)])
             (close-port p)
             (if (eof-object? bv) (make-bytevector 0) bv))))))

;; ---------------------------------------------------------------------
;; REPL

(define isatty
  (guard (e [else (lambda (fd) 0)])
    (load-shared-object "libc.so.6")
    (foreign-procedure "isatty" (int) int)))

(define (repl)
  (let ([tty (guard (e [else #f]) (= 1 (isatty 0)))]
        [in (current-input-port)])
    (define (prompt)
      (when tty
        (put-string (current-output-port) "> ")
        (flush-output-port (current-output-port))))
    ;; evaluate buffer text; returns #t when consumed, #f when incomplete
    (define (try-eval text final?)
      (let ([ast (guard (e [(syntax-err? e)
                            (if (and (syntax-err-at-eof? e) (not final?))
                                'incomplete
                                (begin (report "SyntaxError" (syntax-err-msg e)) 'error))])
                   (parse-js text 'program))])
        (cond
         [(eq? ast 'incomplete) #f]
         [(eq? ast 'error) #t]
         [else
          (guard (e [(js-error? e) (report "Error" (js-error-msg e))]
                    [else (report "Error" (condition->string e))])
            (set! *caller-ctx* global-ctx)
            (let ([v (run-program ast global-ctx 0)])
              (unless (undefined? v)
                (write-js-string (to-string v))
                (put-u8 stdout 10))
              (flush-stdout)))
          #t])))
    (prompt)
    (let loop ([buf ""])
      (let ([line (get-line in)])
        (if (eof-object? line)
            (begin
              (unless (string=? (trim-str-ws buf) "")
                (try-eval buf #t))
              (flush-stdout)
              (exit 0))
            (let ([text (string-append buf (codepoints->js line) "\n")])
              (if (string=? (trim-str-ws text) "")
                  (begin (prompt) (loop ""))
                  (if (try-eval text #f)
                      (begin (prompt) (loop ""))
                      (loop text)))))))))

;; ---------------------------------------------------------------------

(define (main args)
  (cond
   [(null? args) (repl)]
   [(string=? (car args) "-e")
    (if (and (pair? (cdr args)) (null? (cddr args)))
        (exit (run-source (codepoints->js (cadr args))))
        (usage-error "-e needs exactly one source argument"))]
   [(and (> (string-length (car args)) 0) (char=? (string-ref (car args) 0) #\-))
    (usage-error (string-append "unknown option " (car args)))]
   [(null? (cdr args))
    (let ([bv (read-file (car args))])
      (if bv
          (exit (run-source (utf8->js bv)))
          (usage-error (string-append "cannot read file " (car args)))))]
   [else (usage-error "too many arguments")]))

(main (cdr (command-line)))
