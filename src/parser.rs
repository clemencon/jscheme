// Syntactic analysis with automatic semicolon insertion (§7.8, chapters 11-14).

use std::rc::Rc;

use crate::ast::*;
use crate::lexer::{tokenize, SyntaxError, Tok, Token};
use crate::value::{intern_str, JsStr};

pub struct Parser<'a> {
    toks: Vec<Token>,
    p: usize,
    src: &'a [char],
    in_func: bool,
    loops: u32,
    vars: Vec<JsStr>,
    funcs: Vec<Rc<FuncDecl>>,
    /// Token span [start, end) of the last LeftHandSideExpression parsed with no operator after it.
    last_lhs: (usize, usize),
}

type PResult<T> = Result<T, SyntaxError>;

fn assign_op(p: &str) -> Option<Option<BinOp>> {
    Some(match p {
        "=" => None,
        "*=" => Some(BinOp::Mul),
        "/=" => Some(BinOp::Div),
        "%=" => Some(BinOp::Mod),
        "+=" => Some(BinOp::Add),
        "-=" => Some(BinOp::Sub),
        "<<=" => Some(BinOp::Shl),
        ">>=" => Some(BinOp::Sar),
        ">>>=" => Some(BinOp::Shr),
        "&=" => Some(BinOp::BitAnd),
        "^=" => Some(BinOp::BitXor),
        "|=" => Some(BinOp::BitOr),
        _ => return None,
    })
}

impl<'a> Parser<'a> {
    pub fn new(src: &'a [char]) -> PResult<Self> {
        let toks = tokenize(src)?;
        Ok(Parser { toks, p: 0, src, in_func: false, loops: 0, vars: Vec::new(), funcs: Vec::new(), last_lhs: (usize::MAX, 0) })
    }

    fn tok(&self) -> &Tok {
        &self.toks[self.p].t
    }

    fn cur(&self) -> &Token {
        &self.toks[self.p]
    }

    fn advance(&mut self) {
        if self.p + 1 < self.toks.len() {
            self.p += 1;
        }
    }

    fn is_p(&self, s: &str) -> bool {
        matches!(self.tok(), Tok::P(p) if *p == s)
    }

    fn is_kw(&self, s: &str) -> bool {
        matches!(self.tok(), Tok::Kw(k) if *k == s)
    }

    fn err<T>(&self, msg: &str) -> PResult<T> {
        let t = self.cur();
        let eof = t.t == Tok::Eof;
        let what = match &t.t {
            Tok::Eof => "end of input".to_string(),
            Tok::Ident(s) => format!("'{}'", s),
            Tok::Kw(s) | Tok::P(s) => format!("'{}'", s),
            Tok::Num(n) => format!("number {}", n),
            Tok::Str(_) => "string".to_string(),
        };
        Err(SyntaxError { msg: format!("{} (found {})", msg, what), line: t.line, eof })
    }

    fn expect_p(&mut self, s: &str) -> PResult<()> {
        if self.is_p(s) {
            self.advance();
            Ok(())
        } else {
            self.err(&format!("expected '{}'", s))
        }
    }

    fn ident(&mut self) -> PResult<JsStr> {
        if let Tok::Ident(s) = self.tok() {
            let r = intern_str(s);
            self.advance();
            Ok(r)
        } else {
            self.err("expected an identifier")
        }
    }

    /// Consumes a semicolon, or inserts one (§7.8.1).
    fn semicolon(&mut self) -> PResult<()> {
        if self.is_p(";") {
            self.advance();
            return Ok(());
        }
        let t = self.cur();
        if t.nl || t.t == Tok::Eof || matches!(t.t, Tok::P("}")) {
            return Ok(());
        }
        self.err("expected ';'")
    }

    // Programs and functions.

    pub fn parse_program(mut self) -> PResult<Program> {
        let mut stmts = Vec::new();
        while self.tok() != &Tok::Eof {
            if self.is_kw("function") {
                let f = self.function_decl()?;
                self.funcs.push(f);
            } else {
                stmts.push(self.statement()?);
            }
        }
        Ok(Program { stmts, funcs: self.funcs, vars: self.vars })
    }

    fn function_decl(&mut self) -> PResult<Rc<FuncDecl>> {
        let start = self.cur().start;
        self.advance();
        let name = self.ident()?;
        self.expect_p("(")?;
        let mut params = Vec::new();
        if !self.is_p(")") {
            loop {
                params.push(self.ident()?);
                if self.is_p(",") {
                    self.advance();
                } else {
                    break;
                }
            }
        }
        self.expect_p(")")?;
        if !self.is_p("{") {
            return self.err("expected '{' to start the function body");
        }
        self.advance();
        let saved = (self.in_func, self.loops, std::mem::take(&mut self.vars));
        self.in_func = true;
        self.loops = 0;
        let mut body = Vec::new();
        while !self.is_p("}") {
            if self.tok() == &Tok::Eof {
                return self.err("expected '}'");
            }
            body.push(self.statement()?);
        }
        let end = self.cur().end;
        self.advance();
        let vars = std::mem::replace(&mut self.vars, saved.2);
        self.in_func = saved.0;
        self.loops = saved.1;
        let source: String = self.src[start..end].iter().collect();
        Ok(Rc::new(make_decl(name, params, body, vars, source)))
    }

    /// Parses a function body for the Function constructor: statements with return allowed.
    pub fn parse_body(mut self) -> PResult<(Vec<Stmt>, Vec<JsStr>)> {
        self.in_func = true;
        let mut body = Vec::new();
        while self.tok() != &Tok::Eof {
            body.push(self.statement()?);
        }
        Ok((body, self.vars))
    }

    /// Parses FormalParameterList(opt) for the Function constructor.
    pub fn parse_params(mut self) -> PResult<Vec<JsStr>> {
        let mut params = Vec::new();
        if self.tok() != &Tok::Eof {
            loop {
                params.push(self.ident()?);
                if self.is_p(",") {
                    self.advance();
                } else {
                    break;
                }
            }
        }
        if self.tok() != &Tok::Eof {
            return self.err("expected end of parameter list");
        }
        Ok(params)
    }

    // Statements (chapter 12).

    fn statement(&mut self) -> PResult<Stmt> {
        match self.tok().clone() {
            Tok::P("{") => {
                self.advance();
                let mut v = Vec::new();
                while !self.is_p("}") {
                    if self.tok() == &Tok::Eof {
                        return self.err("expected '}'");
                    }
                    v.push(self.statement()?);
                }
                self.advance();
                Ok(Stmt::Block(v))
            }
            Tok::P(";") => {
                self.advance();
                Ok(Stmt::Empty)
            }
            Tok::Kw("var") => {
                self.advance();
                let decls = self.var_decls()?;
                self.semicolon()?;
                Ok(Stmt::Var(decls))
            }
            Tok::Kw("if") => {
                self.advance();
                self.expect_p("(")?;
                let c = self.expression()?;
                self.expect_p(")")?;
                let t = self.statement()?;
                let e = if self.is_kw("else") {
                    self.advance();
                    Some(Box::new(self.statement()?))
                } else {
                    None
                };
                Ok(Stmt::If(c, Box::new(t), e))
            }
            Tok::Kw("while") => {
                self.advance();
                self.expect_p("(")?;
                let c = self.expression()?;
                self.expect_p(")")?;
                let body = self.loop_body()?;
                Ok(Stmt::While(c, Box::new(body)))
            }
            Tok::Kw("for") => self.for_statement(),
            Tok::Kw("continue") => {
                if self.loops == 0 {
                    return self.err("continue outside a loop");
                }
                self.advance();
                self.semicolon()?;
                Ok(Stmt::Continue)
            }
            Tok::Kw("break") => {
                if self.loops == 0 {
                    return self.err("break outside a loop");
                }
                self.advance();
                self.semicolon()?;
                Ok(Stmt::Break)
            }
            Tok::Kw("return") => {
                if !self.in_func {
                    return self.err("return outside a function");
                }
                self.advance();
                let t = self.cur();
                let e = if t.nl || t.t == Tok::Eof || matches!(t.t, Tok::P(";") | Tok::P("}")) {
                    None
                } else {
                    Some(self.expression()?)
                };
                self.semicolon()?;
                Ok(Stmt::Return(e))
            }
            Tok::Kw("with") => {
                self.advance();
                self.expect_p("(")?;
                let o = self.expression()?;
                self.expect_p(")")?;
                let body = self.statement()?;
                Ok(Stmt::With(o, Box::new(body)))
            }
            Tok::Kw("function") => self.err("a function declaration is only allowed at the top level of a program"),
            _ => {
                let e = self.expression()?;
                self.semicolon()?;
                Ok(Stmt::Expr(e))
            }
        }
    }

    fn loop_body(&mut self) -> PResult<Stmt> {
        self.loops += 1;
        let r = self.statement();
        self.loops -= 1;
        r
    }

    fn var_decls(&mut self) -> PResult<Vec<VarDecl>> {
        let mut v = Vec::new();
        loop {
            let name = self.ident()?;
            self.vars.push(name.clone());
            let init = if self.is_p("=") {
                self.advance();
                Some(self.assignment()?)
            } else {
                None
            };
            v.push(VarDecl { name, slot: NO_SLOT, init });
            if self.is_p(",") {
                self.advance();
            } else {
                return Ok(v);
            }
        }
    }

    fn for_statement(&mut self) -> PResult<Stmt> {
        self.advance();
        self.expect_p("(")?;
        let mut init = None;
        if self.is_kw("var") {
            self.advance();
            let decls = self.var_decls()?;
            if self.is_kw("in") && decls.len() == 1 {
                self.advance();
                let d = decls.into_iter().next().unwrap();
                let obj = self.expression()?;
                self.expect_p(")")?;
                let body = self.loop_body()?;
                return Ok(Stmt::ForIn(ForInTarget::Var(d), obj, Box::new(body)));
            }
            init = Some(ForInit::Var(decls));
        } else if !self.is_p(";") {
            let start = self.p;
            let e = self.expression()?;
            if self.is_kw("in") {
                if self.last_lhs != (start, self.p) {
                    return self.err("invalid left-hand side in for-in");
                }
                self.advance();
                let obj = self.expression()?;
                self.expect_p(")")?;
                let body = self.loop_body()?;
                return Ok(Stmt::ForIn(ForInTarget::Lhs(e), obj, Box::new(body)));
            }
            init = Some(ForInit::Expr(e));
        }
        self.expect_p(";")?;
        let test = if self.is_p(";") { None } else { Some(self.expression()?) };
        self.expect_p(";")?;
        let update = if self.is_p(")") { None } else { Some(self.expression()?) };
        self.expect_p(")")?;
        let body = self.loop_body()?;
        Ok(Stmt::For(init, test, update, Box::new(body)))
    }

    // Expressions (chapter 11).

    pub fn expression(&mut self) -> PResult<Expr> {
        let mut e = self.assignment()?;
        while self.is_p(",") {
            self.advance();
            let r = self.assignment()?;
            e = Expr::Comma(Box::new(e), Box::new(r));
        }
        Ok(e)
    }

    fn assignment(&mut self) -> PResult<Expr> {
        let start = self.p;
        let lhs = self.conditional()?;
        if let Tok::P(p) = self.tok() {
            if let Some(op) = assign_op(p) {
                if self.last_lhs != (start, self.p) {
                    return self.err("invalid left-hand side in assignment");
                }
                self.advance();
                let rhs = self.assignment()?;
                return Ok(match op {
                    None => Expr::Assign(Box::new(lhs), Box::new(rhs)),
                    Some(op) => Expr::OpAssign(op, Box::new(lhs), Box::new(rhs)),
                });
            }
        }
        Ok(lhs)
    }

    fn conditional(&mut self) -> PResult<Expr> {
        let c = self.logical_or()?;
        if self.is_p("?") {
            self.advance();
            let a = self.assignment()?;
            self.expect_p(":")?;
            let b = self.assignment()?;
            return Ok(Expr::Cond(Box::new(c), Box::new(a), Box::new(b)));
        }
        Ok(c)
    }

    fn logical_or(&mut self) -> PResult<Expr> {
        let mut e = self.logical_and()?;
        while self.is_p("||") {
            self.advance();
            let r = self.logical_and()?;
            e = Expr::Or(Box::new(e), Box::new(r));
        }
        Ok(e)
    }

    fn logical_and(&mut self) -> PResult<Expr> {
        let mut e = self.binary(0)?;
        while self.is_p("&&") {
            self.advance();
            let r = self.binary(0)?;
            e = Expr::And(Box::new(e), Box::new(r));
        }
        Ok(e)
    }

    fn binop_at(&self, level: usize) -> Option<BinOp> {
        let p = match self.tok() {
            Tok::P(p) => *p,
            _ => return None,
        };
        let op = match (level, p) {
            (0, "|") => BinOp::BitOr,
            (1, "^") => BinOp::BitXor,
            (2, "&") => BinOp::BitAnd,
            (3, "==") => BinOp::Eq,
            (3, "!=") => BinOp::Ne,
            (4, "<") => BinOp::Lt,
            (4, ">") => BinOp::Gt,
            (4, "<=") => BinOp::Le,
            (4, ">=") => BinOp::Ge,
            (5, "<<") => BinOp::Shl,
            (5, ">>") => BinOp::Sar,
            (5, ">>>") => BinOp::Shr,
            (6, "+") => BinOp::Add,
            (6, "-") => BinOp::Sub,
            (7, "*") => BinOp::Mul,
            (7, "/") => BinOp::Div,
            (7, "%") => BinOp::Mod,
            _ => return None,
        };
        Some(op)
    }

    fn binary(&mut self, level: usize) -> PResult<Expr> {
        if level == 8 {
            return self.unary();
        }
        let mut e = self.binary(level + 1)?;
        while let Some(op) = self.binop_at(level) {
            self.advance();
            let r = self.binary(level + 1)?;
            e = Expr::Bin(op, Box::new(e), Box::new(r));
        }
        Ok(e)
    }

    fn unary(&mut self) -> PResult<Expr> {
        let op = match self.tok() {
            Tok::Kw("delete") => UnOp::Delete,
            Tok::Kw("void") => UnOp::Void,
            Tok::Kw("typeof") => UnOp::Typeof,
            Tok::P("++") => UnOp::PreInc,
            Tok::P("--") => UnOp::PreDec,
            Tok::P("+") => UnOp::Pos,
            Tok::P("-") => UnOp::Neg,
            Tok::P("~") => UnOp::BitNot,
            Tok::P("!") => UnOp::Not,
            _ => return self.postfix(),
        };
        self.advance();
        let e = self.unary()?;
        Ok(Expr::Unary(op, Box::new(e)))
    }

    fn postfix(&mut self) -> PResult<Expr> {
        let start = self.p;
        let e = self.lhs()?;
        let t = self.cur();
        if !t.nl {
            if let Tok::P(p @ ("++" | "--")) = t.t {
                let op = if p == "++" { UnOp::PostInc } else { UnOp::PostDec };
                self.advance();
                return Ok(Expr::Unary(op, Box::new(e)));
            }
        }
        self.last_lhs = (start, self.p);
        Ok(e)
    }

    fn arguments(&mut self) -> PResult<Vec<Expr>> {
        self.expect_p("(")?;
        let mut args = Vec::new();
        if !self.is_p(")") {
            loop {
                args.push(self.assignment()?);
                if self.is_p(",") {
                    self.advance();
                } else {
                    break;
                }
            }
        }
        self.expect_p(")")?;
        Ok(args)
    }

    fn member_suffix(&mut self, e: Expr, calls: bool) -> PResult<Expr> {
        let mut e = e;
        loop {
            if self.is_p(".") {
                self.advance();
                let name = self.ident()?;
                e = Expr::Dot(Box::new(e), name);
            } else if self.is_p("[") {
                self.advance();
                let k = self.expression()?;
                self.expect_p("]")?;
                e = Expr::Index(Box::new(e), Box::new(k));
            } else if calls && self.is_p("(") {
                let args = self.arguments()?;
                e = Expr::Call(Box::new(e), args);
            } else {
                return Ok(e);
            }
        }
    }

    fn new_expr(&mut self) -> PResult<Expr> {
        self.advance();
        let callee = if self.is_kw("new") { self.new_expr()? } else { self.primary()? };
        let callee = self.member_suffix(callee, false)?;
        let args = if self.is_p("(") { self.arguments()? } else { Vec::new() };
        Ok(Expr::New(Box::new(callee), args))
    }

    fn lhs(&mut self) -> PResult<Expr> {
        let e = if self.is_kw("new") { self.new_expr()? } else { self.primary()? };
        self.member_suffix(e, true)
    }

    fn primary(&mut self) -> PResult<Expr> {
        let e = match self.tok().clone() {
            Tok::Kw("this") => Expr::This,
            Tok::Kw("null") => Expr::Null,
            Tok::Kw("true") => Expr::Bool(true),
            Tok::Kw("false") => Expr::Bool(false),
            Tok::Ident(s) => Expr::Ident(intern_str(&s), NO_SLOT),
            Tok::Num(n) => Expr::Num(n),
            Tok::Str(s) => Expr::Str(s.into()),
            Tok::P("(") => {
                self.advance();
                let e = self.expression()?;
                self.expect_p(")")?;
                return Ok(e);
            }
            _ => return self.err("expected an expression"),
        };
        self.advance();
        Ok(e)
    }
}

/// Builds a function, binding the identifiers of its body to the slots of its locals.
pub fn make_decl(name: JsStr, params: Vec<JsStr>, mut body: Vec<Stmt>, vars: Vec<JsStr>, source: String) -> FuncDecl {
    let (locals, param_slots) = layout_locals(&intern_str("arguments"), &params, &vars);
    bind_slots(&mut body, &locals);
    FuncDecl { name, params, body, locals, param_slots, source }
}

pub fn parse_program(src: &str) -> PResult<Program> {
    let chars: Vec<char> = src.chars().collect();
    Parser::new(&chars)?.parse_program()
}
