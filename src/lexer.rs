// Lexical analysis (ECMA-262 1st edition, chapter 7).

#[derive(Clone, Debug, PartialEq)]
pub enum Tok {
    Ident(String),
    Kw(&'static str),
    Num(f64),
    Str(Vec<u16>),
    P(&'static str),
    Eof,
}

#[derive(Clone, Debug)]
pub struct Token {
    pub t: Tok,
    /// A LineTerminator occurs between the previous token and this one.
    pub nl: bool,
    pub start: usize,
    pub end: usize,
    pub line: u32,
}

#[derive(Debug, Clone)]
pub struct SyntaxError {
    pub msg: String,
    pub line: u32,
    /// The error was caused by reaching the end of the input (the REPL may read more).
    pub eof: bool,
}

const KEYWORDS: &[&str] = &[
    "break", "continue", "delete", "else", "for", "function", "if", "in", "new", "return", "this",
    "typeof", "var", "void", "while", "with", "null", "true", "false",
];

const FUTURE_RESERVED: &[&str] = &[
    "case", "catch", "class", "const", "debugger", "default", "do", "enum", "export", "extends",
    "finally", "import", "super", "switch", "throw", "try",
];

const PUNCTS: &[&str] = &[
    ">>>=", ">>>", "<<=", ">>=", "==", "!=", "<=", ">=", "&&", "||", "++", "--", "+=", "-=", "*=",
    "/=", "&=", "|=", "^=", "%=", "<<", ">>", "=", "<", ">", "!", "~", "?", ":", ",", ".", "+", "-",
    "*", "/", "&", "|", "^", "%", "(", ")", "{", "}", "[", "]", ";",
];

fn is_id_start(c: char) -> bool {
    c.is_ascii_alphabetic() || c == '$' || c == '_'
}

fn is_id_part(c: char) -> bool {
    c.is_ascii_alphanumeric() || c == '$' || c == '_'
}

fn is_lt(c: char) -> bool {
    c == '\n' || c == '\r'
}

fn is_ws(c: char) -> bool {
    c == '\t' || c == '\u{0B}' || c == '\u{0C}' || c == ' '
}

pub fn hex_val(c: u32) -> Option<u32> {
    match c {
        0x30..=0x39 => Some(c - 0x30),
        0x41..=0x46 => Some(c - 0x41 + 10),
        0x61..=0x66 => Some(c - 0x61 + 10),
        _ => None,
    }
}

struct Lexer<'a> {
    s: &'a [char],
    i: usize,
    line: u32,
}

impl<'a> Lexer<'a> {
    fn peek(&self, k: usize) -> Option<char> {
        self.s.get(self.i + k).copied()
    }

    fn err(&self, msg: &str, eof: bool) -> SyntaxError {
        SyntaxError { msg: msg.to_string(), line: self.line, eof }
    }

    /// Skips white space and comments; returns whether a line terminator was seen.
    fn skip(&mut self) -> Result<bool, SyntaxError> {
        let mut nl = false;
        loop {
            match self.peek(0) {
                Some(c) if is_ws(c) => self.i += 1,
                Some(c) if is_lt(c) => {
                    nl = true;
                    self.line += 1;
                    self.i += 1;
                }
                Some('/') if self.peek(1) == Some('/') => {
                    self.i += 2;
                    while let Some(c) = self.peek(0) {
                        if is_lt(c) {
                            break;
                        }
                        self.i += 1;
                    }
                }
                Some('/') if self.peek(1) == Some('*') => {
                    self.i += 2;
                    loop {
                        match self.peek(0) {
                            None => return Err(self.err("unterminated comment", true)),
                            Some('*') if self.peek(1) == Some('/') => {
                                self.i += 2;
                                break;
                            }
                            Some(c) => {
                                if is_lt(c) {
                                    nl = true;
                                    self.line += 1;
                                }
                                self.i += 1;
                            }
                        }
                    }
                }
                _ => return Ok(nl),
            }
        }
    }

    fn number(&mut self) -> Result<f64, SyntaxError> {
        let start = self.i;
        let c0 = self.peek(0).unwrap();
        if c0 == '0' && matches!(self.peek(1), Some('x') | Some('X')) {
            self.i += 2;
            let mut v = 0f64;
            let mut n = 0;
            while let Some(d) = self.peek(0).and_then(|c| hex_val(c as u32)) {
                v = v * 16.0 + d as f64;
                self.i += 1;
                n += 1;
            }
            if n == 0 {
                return Err(self.err("missing hexadecimal digits", false));
            }
            return self.after_number(v);
        }
        if c0 == '0' && matches!(self.peek(1), Some('0'..='7')) {
            self.i += 1;
            let mut v = 0f64;
            while let Some(c @ '0'..='7') = self.peek(0) {
                v = v * 8.0 + (c as u32 - 0x30) as f64;
                self.i += 1;
            }
            return self.after_number(v);
        }
        while let Some('0'..='9') = self.peek(0) {
            self.i += 1;
        }
        if self.peek(0) == Some('.') {
            self.i += 1;
            while let Some('0'..='9') = self.peek(0) {
                self.i += 1;
            }
        }
        if matches!(self.peek(0), Some('e') | Some('E')) {
            self.i += 1;
            if matches!(self.peek(0), Some('+') | Some('-')) {
                self.i += 1;
            }
            let ds = self.i;
            while let Some('0'..='9') = self.peek(0) {
                self.i += 1;
            }
            if ds == self.i {
                return Err(self.err("missing exponent digits", false));
            }
        }
        let text: Vec<u16> = self.s[start..self.i].iter().map(|c| *c as u16).collect();
        let v = crate::numconv::parse_decimal(&text);
        self.after_number(v)
    }

    fn after_number(&self, v: f64) -> Result<f64, SyntaxError> {
        if let Some(c) = self.peek(0) {
            if is_id_part(c) {
                return Err(self.err("identifier starts immediately after a number", false));
            }
        }
        Ok(v)
    }

    fn string(&mut self, q: char) -> Result<Vec<u16>, SyntaxError> {
        self.i += 1;
        let mut out: Vec<u16> = Vec::new();
        loop {
            let c = match self.peek(0) {
                None => return Err(self.err("unterminated string literal", false)),
                Some(c) => c,
            };
            if c == q {
                self.i += 1;
                return Ok(out);
            }
            if is_lt(c) {
                return Err(self.err("line terminator in string literal", false));
            }
            if c != '\\' {
                let mut buf = [0u16; 2];
                out.extend_from_slice(c.encode_utf16(&mut buf));
                self.i += 1;
                continue;
            }
            self.i += 1;
            let e = match self.peek(0) {
                None => return Err(self.err("unterminated string literal", false)),
                Some(e) => e,
            };
            if is_lt(e) {
                return Err(self.err("line terminator in string literal", false));
            }
            self.i += 1;
            match e {
                'b' => out.push(8),
                't' => out.push(9),
                'n' => out.push(10),
                'f' => out.push(12),
                'r' => out.push(13),
                'x' => {
                    let a = self.peek(0).and_then(|c| hex_val(c as u32));
                    let b = self.peek(1).and_then(|c| hex_val(c as u32));
                    match (a, b) {
                        (Some(a), Some(b)) => {
                            out.push((a * 16 + b) as u16);
                            self.i += 2;
                        }
                        _ => return Err(self.err("bad hexadecimal escape", false)),
                    }
                }
                'u' => {
                    let mut v = 0u32;
                    for k in 0..4 {
                        match self.peek(k).and_then(|c| hex_val(c as u32)) {
                            Some(d) => v = v * 16 + d,
                            None => return Err(self.err("bad Unicode escape", false)),
                        }
                    }
                    self.i += 4;
                    out.push(v as u16);
                }
                '0'..='7' => {
                    let mut v = e as u32 - 0x30;
                    let max = if e <= '3' { 2 } else { 1 };
                    for _ in 0..max {
                        match self.peek(0) {
                            Some(d @ '0'..='7') => {
                                v = v * 8 + (d as u32 - 0x30);
                                self.i += 1;
                            }
                            _ => break,
                        }
                    }
                    out.push(v as u16);
                }
                other => {
                    let mut buf = [0u16; 2];
                    out.extend_from_slice(other.encode_utf16(&mut buf));
                }
            }
        }
    }
}

pub fn tokenize(s: &[char]) -> Result<Vec<Token>, SyntaxError> {
    let mut lx = Lexer { s, i: 0, line: 1 };
    let mut toks = Vec::new();
    loop {
        let nl = lx.skip()?;
        let start = lx.i;
        let line = lx.line;
        let c = match lx.peek(0) {
            None => {
                toks.push(Token { t: Tok::Eof, nl, start, end: start, line });
                return Ok(toks);
            }
            Some(c) => c,
        };
        let t = if is_id_start(c) {
            while let Some(c) = lx.peek(0) {
                if !is_id_part(c) {
                    break;
                }
                lx.i += 1;
            }
            let w: String = s[start..lx.i].iter().collect();
            if let Some(k) = KEYWORDS.iter().find(|k| **k == w) {
                Tok::Kw(k)
            } else if FUTURE_RESERVED.contains(&w.as_str()) {
                return Err(SyntaxError { msg: format!("reserved word '{}'", w), line, eof: false });
            } else {
                Tok::Ident(w)
            }
        } else if c.is_ascii_digit() || (c == '.' && matches!(lx.peek(1), Some('0'..='9'))) {
            Tok::Num(lx.number()?)
        } else if c == '"' || c == '\'' {
            Tok::Str(lx.string(c)?)
        } else {
            let mut found = None;
            for p in PUNCTS {
                let n = p.len();
                if lx.i + n <= s.len() && p.chars().zip(&s[lx.i..lx.i + n]).all(|(a, b)| a == *b) {
                    found = Some(*p);
                    break;
                }
            }
            match found {
                Some(p) => {
                    lx.i += p.len();
                    Tok::P(p)
                }
                None => {
                    return Err(SyntaxError { msg: format!("unexpected character '{}'", c), line, eof: false });
                }
            }
        };
        toks.push(Token { t, nl, start, end: lx.i, line });
    }
}
