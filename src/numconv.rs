// Number <-> String conversions (§9.3.1, §9.8.1, §15.1.2.2, §15.1.2.3).

/// ToString applied to the Number type (§9.8.1), with the shortest round-tripping digits.
pub fn num_to_string(m: f64) -> String {
    if m.is_nan() {
        return "NaN".into();
    }
    if m == 0.0 {
        return "0".into();
    }
    if m < 0.0 {
        return format!("-{}", num_to_string(-m));
    }
    if m.is_infinite() {
        return "Infinity".into();
    }
    if m < 1e15 && m.fract() == 0.0 {
        return format!("{}", m as u64);
    }
    let e = format!("{:e}", m);
    let (mant, exp) = e.split_once('e').unwrap();
    let digits: String = mant.chars().filter(|c| *c != '.').collect();
    let k = digits.len() as i32;
    let n = exp.parse::<i32>().unwrap() + 1;
    if k <= n && n <= 21 {
        let mut s = digits;
        for _ in 0..(n - k) {
            s.push('0');
        }
        s
    } else if 0 < n && n <= 21 {
        format!("{}.{}", &digits[..n as usize], &digits[n as usize..])
    } else if -6 < n && n <= 0 {
        let mut s = String::from("0.");
        for _ in 0..(-n) {
            s.push('0');
        }
        s + &digits
    } else {
        let sign = if n - 1 < 0 { '-' } else { '+' };
        if k == 1 {
            format!("{}e{}{}", digits, sign, (n - 1).abs())
        } else {
            format!("{}.{}e{}{}", &digits[..1], &digits[1..], sign, (n - 1).abs())
        }
    }
}

/// Number.prototype.toString with a radix other than 10.
pub fn num_to_radix(m: f64, radix: u32) -> String {
    if !m.is_finite() {
        return num_to_string(m);
    }
    let neg = m < 0.0;
    let mut x = m.abs();
    let mut int = x.trunc();
    x -= int;
    let digits = b"0123456789abcdefghijklmnopqrstuvwxyz";
    let mut ip = Vec::new();
    if int == 0.0 {
        ip.push(b'0');
    }
    while int > 0.0 {
        let d = (int % radix as f64) as usize;
        ip.push(digits[d]);
        int = ((int - d as f64) / radix as f64).trunc();
    }
    ip.reverse();
    let mut s = String::from_utf8(ip).unwrap();
    if x > 0.0 {
        s.push('.');
        let mut n = 0;
        while x > 0.0 && n < 52 {
            x *= radix as f64;
            let d = x.trunc();
            s.push(digits[d as usize] as char);
            x -= d;
            n += 1;
        }
    }
    if neg {
        s.insert(0, '-');
    }
    s
}

pub fn is_str_ws(c: u16) -> bool {
    matches!(c, 9 | 10 | 11 | 12 | 13 | 32)
}

fn is_digit(c: u16) -> bool {
    (0x30..=0x39).contains(&c)
}

/// Length of the longest prefix of s that is an unsigned StrDecimalLiteral (§9.3.1), or 0.
pub fn decimal_prefix(s: &[u16]) -> usize {
    let inf: Vec<u16> = "Infinity".encode_utf16().collect();
    if s.starts_with(&inf) {
        return inf.len();
    }
    let mut i = 0;
    let mut nd = 0;
    while i < s.len() && is_digit(s[i]) {
        i += 1;
        nd += 1;
    }
    if i < s.len() && s[i] == b'.' as u16 {
        let mut j = i + 1;
        let mut fd = 0;
        while j < s.len() && is_digit(s[j]) {
            j += 1;
            fd += 1;
        }
        if nd + fd > 0 {
            i = j;
            nd += fd;
        }
    }
    if nd == 0 {
        return 0;
    }
    if i < s.len() && (s[i] == b'e' as u16 || s[i] == b'E' as u16) {
        let mut j = i + 1;
        if j < s.len() && (s[j] == b'+' as u16 || s[j] == b'-' as u16) {
            j += 1;
        }
        let ds = j;
        while j < s.len() && is_digit(s[j]) {
            j += 1;
        }
        if j > ds {
            i = j;
        }
    }
    i
}

/// The number value of an unsigned StrDecimalLiteral.
pub fn parse_decimal(s: &[u16]) -> f64 {
    let t = String::from_utf16_lossy(s);
    if t == "Infinity" {
        return f64::INFINITY;
    }
    // Normalise "5." and ".5" forms so that every digit sequence is non-empty.
    let mut n = String::with_capacity(t.len() + 2);
    let b = t.as_bytes();
    for (i, &c) in b.iter().enumerate() {
        if c == b'.' {
            if i == 0 {
                n.push('0');
            }
            n.push('.');
            if i + 1 >= b.len() || !b[i + 1].is_ascii_digit() {
                n.push('0');
            }
        } else {
            n.push(c as char);
        }
    }
    n.parse::<f64>().unwrap_or(f64::NAN)
}

/// ToNumber applied to the String type (§9.3.1).
pub fn str_to_number(s: &[u16]) -> f64 {
    let mut a = 0;
    let mut b = s.len();
    while a < b && is_str_ws(s[a]) {
        a += 1;
    }
    while b > a && is_str_ws(s[b - 1]) {
        b -= 1;
    }
    let s = &s[a..b];
    if s.is_empty() {
        return 0.0;
    }
    if s.len() > 2 && s[0] == b'0' as u16 && (s[1] == b'x' as u16 || s[1] == b'X' as u16) {
        let mut v = 0f64;
        for &c in &s[2..] {
            match crate::lexer::hex_val(c as u32) {
                Some(d) => v = v * 16.0 + d as f64,
                None => return f64::NAN,
            }
        }
        return v;
    }
    let (neg, body) = match s[0] {
        0x2B => (false, &s[1..]),
        0x2D => (true, &s[1..]),
        _ => (false, s),
    };
    if body.is_empty() || decimal_prefix(body) != body.len() {
        return f64::NAN;
    }
    let v = parse_decimal(body);
    if neg { -v } else { v }
}
