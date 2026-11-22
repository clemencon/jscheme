;;; Date objects (§15.9). Local time is UTC: LocalTZA and DaylightSavingTA
;;; are 0 (the test harness runs with TZ=UTC).

(define ms-per-day 86400000)

;; Exact-integer helpers on time values
(define (day-of t) (floor (/ t ms-per-day)))
(define (time-within-day t) (modulo t ms-per-day))
(define (day-from-year y)
  (+ (* 365 (- y 1970)) (floor (/ (- y 1969) 4)) (- (floor (/ (- y 1901) 100))) (floor (/ (- y 1601) 400))))
(define (time-from-year y) (* ms-per-day (day-from-year y)))
(define (leap-year? y)
  (and (= 0 (modulo y 4)) (or (not (= 0 (modulo y 100))) (= 0 (modulo y 400)))))
(define (year-from-time t)
  (let loop ([y (+ 1970 (floor (/ (day-of t) 365.2425)))])
    (let ([y (exact (floor y))])
      (cond [(> (time-from-year y) t) (loop (- y 1))]
            [(<= (time-from-year (+ y 1)) t) (loop (+ y 1))]
            [else y]))))
(define (day-within-year t) (- (day-of t) (day-from-year (year-from-time t))))
(define month-starts '#(0 31 59 90 120 151 181 212 243 273 304 334 365))
(define (month-start m leap) (+ (vector-ref month-starts m) (if (and leap (>= m 2)) 1 0)))
(define (month-from-time t)
  (let ([d (day-within-year t)] [leap (leap-year? (year-from-time t))])
    (let loop ([m 0])
      (if (< d (month-start (+ m 1) leap)) m (loop (+ m 1))))))
(define (date-from-time t)
  (let ([leap (leap-year? (year-from-time t))])
    (+ 1 (- (day-within-year t) (month-start (month-from-time t) leap)))))
(define (week-day t) (modulo (+ (day-of t) 4) 7))
(define (hour-from-time t) (modulo (floor (/ t 3600000)) 24))
(define (min-from-time t) (modulo (floor (/ t 60000)) 60))
(define (sec-from-time t) (modulo (floor (/ t 1000)) 60))
(define (ms-from-time t) (modulo t 1000))

;; Operators on flonums (§15.9.1.11-14)
(define (js-make-time h m s ms)
  (if (not (and (finite? h) (finite? m) (finite? s) (finite? ms)))
      +nan.0
      (fl+ (fl+ (fl+ (fl* (fltruncate h) 3600000.0) (fl* (fltruncate m) 60000.0))
                (fl* (fltruncate s) 1000.0))
           (fltruncate ms))))

(define (make-day year month date)
  (if (not (and (finite? year) (finite? month) (finite? date)))
      +nan.0
      (let* ([y (exact (fltruncate year))]
             [m (exact (fltruncate month))]
             [dt (fltruncate date)]
             [ym (+ y (floor (/ m 12)))]
             [mn (modulo m 12)])
        (if (> (abs ym) 400000)
            +nan.0
            (fl+ (inexact (+ (day-from-year ym) (month-start mn (leap-year? ym))))
                 (fl- dt 1.0))))))

(define (js-make-date day time)
  (if (not (and (finite? day) (finite? time)))
      +nan.0
      (fl+ (fl* day 86400000.0) time)))

(define (time-clip t)
  (cond [(not (finite? t)) +nan.0]
        [(fl> (flabs t) 8.64e15) +nan.0]
        [else (fl+ (fltruncate t) 0.0)]))

(define (current-time-ms)
  (let ([t (current-time)])
    (inexact (+ (* (time-second t) 1000) (quotient (time-nanosecond t) 1000000)))))

;; Date string format: "Fri, 15 Jun 2001 10:20:30 GMT"
(define day-names '#("Sun" "Mon" "Tue" "Wed" "Thu" "Fri" "Sat"))
(define month-names '#("Jan" "Feb" "Mar" "Apr" "May" "Jun" "Jul" "Aug" "Sep" "Oct" "Nov" "Dec"))

(define (pad2 n) (if (< n 10) (string-append "0" (number->string n)) (number->string n)))

(define (time->string tv)
  (if (nan? tv)
      "Invalid Date"
      (let ([t (exact tv)])
        (string-append
         (vector-ref day-names (week-day t)) ", "
         (pad2 (date-from-time t)) " "
         (vector-ref month-names (month-from-time t)) " "
         (number->string (year-from-time t)) " "
         (pad2 (hour-from-time t)) ":" (pad2 (min-from-time t)) ":" (pad2 (sec-from-time t))
         " GMT"))))

;; Date.parse: accepts the format above and common variants.
(define (parse-date str)
  (let ([n (string-length str)])
    (define (ch i) (if (< i n) (string-ref str i) #\nul))
    (define (lower s) (string-downcase s))
    (let loop ([i 0] [year #f] [month #f] [day #f] [h 0] [mi 0] [s 0] [ms 0] [offset 0] [nums '()] [pm #f] [am #f])
      (cond
       [(>= i n)
        ;; assign plain numbers
        (let assign ([nums (reverse nums)] [year year] [day day] [month month])
          (if (null? nums)
              (if (and year month day)
                  (let* ([yr (if (and (>= year 0) (<= year 99)) (+ year 1900) year)]
                         [h (cond [(and pm (< h 12)) (+ h 12)] [(and am (= h 12)) 0] [else h])]
                         [t (js-make-date (make-day (inexact yr) (inexact month) (inexact day))
                                       (js-make-time (inexact h) (inexact mi) (inexact s) (inexact ms)))])
                    (time-clip (fl- t (inexact (* offset 60000)))))
                  +nan.0)
              (let ([v (car nums)])
                (cond
                 [(and (not day) month (<= 1 v 31) (not (and year (not month)))) (assign (cdr nums) year v month)]
                 [(and (not day) (not month) (<= 1 v 31)) (assign (cdr nums) year v month)]
                 [(not year) (assign (cdr nums) v day month)]
                 [(and (not month) (<= 1 v 12)) (assign (cdr nums) year day (- v 1))]
                 [else +nan.0]))))]
       [(memv (ch i) '(#\space #\, #\tab #\newline)) (loop (+ i 1) year month day h mi s ms offset nums pm am)]
       [(char-alphabetic? (ch i))
        (let* ([j (let scan ([j i]) (if (char-alphabetic? (ch j)) (scan (+ j 1)) j))]
               [w (lower (substring str i j))])
          (cond
           [(member w '("gmt" "utc" "ut" "z"))
            (loop j year month day h mi s ms offset nums pm am)]
           [(string=? w "am") (loop j year month day h mi s ms offset nums pm #t)]
           [(string=? w "pm") (loop j year month day h mi s ms offset nums #t am)]
           [(>= (string-length w) 3)
            (let ([mi-idx (let find ([k 0])
                            (cond [(= k 12) #f]
                                  [(string=? (lower (vector-ref month-names k)) (substring w 0 3)) k]
                                  [else (find (+ k 1))]))]
                  [is-day (let find ([k 0])
                            (cond [(= k 7) #f]
                                  [(string=? (lower (vector-ref day-names k)) (substring w 0 3)) #t]
                                  [else (find (+ k 1))]))])
              (cond [mi-idx (loop j year mi-idx day h mi s ms offset nums pm am)]
                    [is-day (loop j year month day h mi s ms offset nums pm am)]
                    [else +nan.0]))]
           [else +nan.0]))]
       [(or (digit-char? (ch i))
            (and (memv (ch i) '(#\+ #\-)) (digit-char? (ch (+ i 1)))))
        (let* ([sign (case (ch i) [(#\-) -1] [else 1])]
               [signed (memv (ch i) '(#\+ #\-))]
               [ds (if signed (+ i 1) i)]
               [j (let scan ([j ds]) (if (digit-char? (ch j)) (scan (+ j 1)) j))]
               [v (string->number (substring str ds j))])
          (cond
           [(char=? (ch j) #\:)
            ;; time hh:mm[:ss[.mmm]]
            (let* ([k (+ j 1)]
                   [k2 (let scan ([k k]) (if (digit-char? (ch k)) (scan (+ k 1)) k))]
                   [mm (string->number (substring str k k2))])
              (if (not mm)
                  +nan.0
                  (if (char=? (ch k2) #\:)
                      (let* ([k3 (+ k2 1)]
                             [k4 (let scan ([k k3]) (if (digit-char? (ch k)) (scan (+ k 1)) k))]
                             [ss (string->number (substring str k3 k4))])
                        (if (not ss)
                            +nan.0
                            (if (char=? (ch k4) #\.)
                                (let* ([k5 (+ k4 1)]
                                       [k6 (let scan ([k k5]) (if (digit-char? (ch k)) (scan (+ k 1)) k))]
                                       [frac (substring str k5 k6)]
                                       [msv (if (= (string-length frac) 0) 0
                                                (exact (floor (* 1000 (/ (string->number frac) (expt 10 (string-length frac)))))))])
                                  (loop k6 year month day v mm ss msv offset nums pm am))
                                (loop k4 year month day v mm ss 0 offset nums pm am))))
                      (loop k2 year month day v mm 0 0 offset nums pm am))))]
           [(and signed (or (null? nums) #t) (>= (- j ds) 4) (pair? nums) day month year)
            ;; time zone offset like +0100
            (loop j year month day h mi s ms (* sign (+ (* 60 (quotient v 100)) (remainder v 100))) nums pm am)]
           [(and signed (= sign -1))
            (loop j (- v) month day h mi s ms offset nums pm am)]
           [else (loop j year month day h mi s ms offset (cons v nums) pm am)]))]
       [else +nan.0]))))

;; ---------------------------------------------------------------------

(define (this-time this what)
  (check-class this 'Date what)
  (jsobj-value this))

(define (make-date-object tv)
  (let ([o (make-obj date-prototype 'Date)])
    (jsobj-value-set! o tv)
    o))

(define (year-arg y)
  (if (and (not (nan? y)) (let ([i (fltruncate y)]) (and (fl>= i 0.0) (fl<= i 99.0))))
      (fl+ 1900.0 (fltruncate y))
      y))

;; Date from components (local time = UTC), for new Date and Date.UTC
(define (components->time nums)
  (let* ([n (length nums)]
         [get (lambda (i default) (if (< i n) (list-ref nums i) default))]
         [y (year-arg (get 0 +nan.0))]
         [day (make-day y (get 1 0.0) (get 2 1.0))]
         [time (js-make-time (get 3 0.0) (get 4 0.0) (get 5 0.0) (get 6 0.0))])
    (time-clip (js-make-date day time))))

(define (numbers-of args)
  (let loop ([as args])
    (if (null? as) '()
        (let ([v (to-number (car as))])
          (cons v (loop (cdr as)))))))

(define (date-construct args)
  (cond
   [(null? args) (make-date-object (current-time-ms))]
   [(null? (cdr args))
    (let ([v (to-primitive (car args) #f)])
      (make-date-object
       (time-clip (if (string? v) (parse-date v) (to-number v)))))]
   [else (make-date-object (components->time (numbers-of args)))]))

(define date-constructor
  (make-native "Date" 7 (lambda (this args) (time->string (current-time-ms)))))
(jsobj-construct-set! date-constructor date-construct)
(add-global! "Date" date-constructor)
(add-prop! date-constructor "prototype" date-prototype (fxior RO DD DE))
(jsobj-value-set! date-prototype +nan.0)
(add-prop! date-prototype "constructor" date-constructor DE)

(defmethod! date-constructor "parse" 1
  (lambda (this args) (parse-date (to-string (arg args 0)))))
(defmethod! date-constructor "UTC" 7
  (lambda (this args) (components->time (numbers-of args))))

(define (date-string-method name)
  (defmethod! date-prototype name 0
    (lambda (this args) (time->string (this-time this (string-append "Date.prototype." name))))))
(date-string-method "toString")
(date-string-method "toLocaleString")
(let ([f (date-string-method "toUTCString")])
  (add-prop! date-prototype "toGMTString" f DE))

(defmethod! date-prototype "valueOf" 0
  (lambda (this args) (this-time this "Date.prototype.valueOf")))
(defmethod! date-prototype "getTime" 0
  (lambda (this args) (this-time this "Date.prototype.getTime")))

(define-syntax defgetter
  (syntax-rules ()
    [(_ name (t) expr)
     (defmethod! date-prototype name 0
       (lambda (this args)
         (let ([tv (this-time this name)])
           (if (nan? tv) +nan.0
               (let ([t (exact tv)]) (inexact expr))))))]))

(defgetter "getYear" (t) (- (year-from-time t) 1900))
(defgetter "getFullYear" (t) (year-from-time t))
(defgetter "getUTCFullYear" (t) (year-from-time t))
(defgetter "getMonth" (t) (month-from-time t))
(defgetter "getUTCMonth" (t) (month-from-time t))
(defgetter "getDate" (t) (date-from-time t))
(defgetter "getUTCDate" (t) (date-from-time t))
(defgetter "getDay" (t) (week-day t))
(defgetter "getUTCDay" (t) (week-day t))
(defgetter "getHours" (t) (hour-from-time t))
(defgetter "getUTCHours" (t) (hour-from-time t))
(defgetter "getMinutes" (t) (min-from-time t))
(defgetter "getUTCMinutes" (t) (min-from-time t))
(defgetter "getSeconds" (t) (sec-from-time t))
(defgetter "getUTCSeconds" (t) (sec-from-time t))
(defgetter "getMilliseconds" (t) (ms-from-time t))
(defgetter "getUTCMilliseconds" (t) (ms-from-time t))
(defgetter "getTimezoneOffset" (t) 0)

(defmethod! date-prototype "setTime" 1
  (lambda (this args)
    (this-time this "Date.prototype.setTime")
    (let ([v (time-clip (to-number (arg args 0)))])
      (jsobj-value-set! this v)
      v)))

;; Setters. (setter name nargs proc): proc receives the time value t (an
;; exact integer, or #f when NaN) and the list of converted arguments, and
;; returns the new unclipped time value.
(define (field t f) (inexact (f t)))

(define-syntax defsetter
  (syntax-rules ()
    [(_ name len nan-zero (t a) body)
     (defmethod! date-prototype name len
       (lambda (this args)
         (let* ([tv (this-time this name)]
                [tv (if (and nan-zero (nan? tv)) 0.0 tv)]
                [a (numbers-of args)])
           (let ([v (if (nan? tv) +nan.0
                        (let ([t (exact tv)]) (time-clip body)))])
             (jsobj-value-set! this v)
             v))))]))

(define (opt a i default) (if (< i (length a)) (list-ref a i) default))

(define (set-time-fields t h m s ms)
  (js-make-date (inexact (day-of t)) (js-make-time h m s ms)))

(defsetter "setMilliseconds" 1 #f (t a)
  (set-time-fields t (field t hour-from-time) (field t min-from-time) (field t sec-from-time) (opt a 0 +nan.0)))
(defsetter "setUTCMilliseconds" 1 #f (t a)
  (set-time-fields t (field t hour-from-time) (field t min-from-time) (field t sec-from-time) (opt a 0 +nan.0)))
(defsetter "setSeconds" 2 #f (t a)
  (set-time-fields t (field t hour-from-time) (field t min-from-time) (opt a 0 +nan.0) (opt a 1 (field t ms-from-time))))
(defsetter "setUTCSeconds" 2 #f (t a)
  (set-time-fields t (field t hour-from-time) (field t min-from-time) (opt a 0 +nan.0) (opt a 1 (field t ms-from-time))))
(defsetter "setMinutes" 3 #f (t a)
  (set-time-fields t (field t hour-from-time) (opt a 0 +nan.0) (opt a 1 (field t sec-from-time)) (opt a 2 (field t ms-from-time))))
(defsetter "setUTCMinutes" 3 #f (t a)
  (set-time-fields t (field t hour-from-time) (opt a 0 +nan.0) (opt a 1 (field t sec-from-time)) (opt a 2 (field t ms-from-time))))
(defsetter "setHours" 4 #f (t a)
  (set-time-fields t (opt a 0 +nan.0) (opt a 1 (field t min-from-time)) (opt a 2 (field t sec-from-time)) (opt a 3 (field t ms-from-time))))
(defsetter "setUTCHours" 4 #f (t a)
  (set-time-fields t (opt a 0 +nan.0) (opt a 1 (field t min-from-time)) (opt a 2 (field t sec-from-time)) (opt a 3 (field t ms-from-time))))

(define (set-date-fields t y m d)
  (js-make-date (make-day y m d) (inexact (time-within-day t))))

(defsetter "setDate" 1 #f (t a)
  (set-date-fields t (field t year-from-time) (field t month-from-time) (opt a 0 +nan.0)))
(defsetter "setUTCDate" 1 #f (t a)
  (set-date-fields t (field t year-from-time) (field t month-from-time) (opt a 0 +nan.0)))
(defsetter "setMonth" 2 #f (t a)
  (set-date-fields t (field t year-from-time) (opt a 0 +nan.0) (opt a 1 (field t date-from-time))))
(defsetter "setUTCMonth" 2 #f (t a)
  (set-date-fields t (field t year-from-time) (opt a 0 +nan.0) (opt a 1 (field t date-from-time))))
(defsetter "setFullYear" 3 #t (t a)
  (set-date-fields t (opt a 0 +nan.0) (opt a 1 (field t month-from-time)) (opt a 2 (field t date-from-time))))
(defsetter "setUTCFullYear" 3 #t (t a)
  (set-date-fields t (opt a 0 +nan.0) (opt a 1 (field t month-from-time)) (opt a 2 (field t date-from-time))))

(defmethod! date-prototype "setYear" 1
  (lambda (this args)
    (let* ([tv (this-time this "setYear")]
           [t (exact (if (nan? tv) 0.0 tv))]
           [y (to-number (arg args 0))])
      (if (nan? y)
          (begin (jsobj-value-set! this +nan.0) +nan.0)
          (let ([v (time-clip (set-date-fields t (year-arg y) (field t month-from-time) (field t date-from-time)))])
            (jsobj-value-set! this v)
            v)))))
